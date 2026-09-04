package server

import (
	"net"
	"net/http"
	"strings"

	"mantenimiento/internal/auth"
	"mantenimiento/internal/middleware"
	"mantenimiento/internal/models"
)

const dummyHash = "$argon2id$v=19$m=19456,t=2,p=1$sxMwc3vhFa1ymBCZo7wS/A$7m+wddg01TJtSC0eoP2gpnjCBBylQ+TTYOUNetpTLsk"
const genericLoginError = "Usuario o contraseña incorrectos."

type loginData struct {
	Title       string
	AppName     string
	Environment string
	UserName    string
	RoleLabel   string
	Error       string
	Username    string
}

type passwordData struct {
	Title       string
	AppName     string
	Environment string
	UserName    string
	RoleLabel   string
	User        *models.User
	CSRF        string
	Error       string
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (s *Server) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	if _, ok := middleware.SessionFrom(r); ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	s.render(w, http.StatusOK, "login", loginData{
		Title:       "Ingresar",
		AppName:     "Sistema de Mantenimiento",
		Environment: s.cfg.Env,
	})
}

func (s *Server) handleLoginPost(w http.ResponseWriter, r *http.Request) {
	username := strings.ToLower(strings.TrimSpace(r.FormValue("username")))
	password := r.FormValue("password")
	ip := clientIP(r)
	comboKey := username + "|" + ip

	if s.userLimiter.Blocked(comboKey) || s.ipLimiter.Blocked(ip) {
		s.render(w, http.StatusTooManyRequests, "login", loginData{
			Title:       "Ingresar",
			AppName:     "Sistema de Mantenimiento",
			Environment: s.cfg.Env,
			Error:       "Demasiados intentos. Espera unos minutos y vuelve a probar.",
			Username:    username,
		})
		return
	}

	user, hash, err := models.UserByUsername(r.Context(), s.db, username)
	if err != nil {
		s.log.Error("login lookup", "err", err)
		s.render(w, http.StatusInternalServerError, "login", loginData{
			Title:       "Ingresar",
			AppName:     "Sistema de Mantenimiento",
			Environment: s.cfg.Env,
			Error:       "Error interno. Intenta de nuevo.",
			Username:    username,
		})
		return
	}

	if user == nil {
		hash = dummyHash
	}

	ok, err := auth.VerifyPassword(password, hash)
	if err != nil {
		s.log.Error("login verify", "user", username, "err", err)
	}
	if !ok || user == nil || !user.Active {
		s.userLimiter.Fail(comboKey)
		s.ipLimiter.Fail(ip)
		s.log.Warn("login failed", "user", username, "ip", ip)
		s.render(w, http.StatusUnauthorized, "login", loginData{
			Title:       "Ingresar",
			AppName:     "Sistema de Mantenimiento",
			Environment: s.cfg.Env,
			Error:       genericLoginError,
			Username:    username,
		})
		return
	}

	s.userLimiter.Reset(comboKey)
	token, err := s.sessions.Create(r.Context(), user.ID, ip, r.UserAgent())
	if err != nil {
		s.log.Error("create session", "err", err)
		s.render(w, http.StatusInternalServerError, "login", loginData{
			Title:       "Ingresar",
			AppName:     "Sistema de Mantenimiento",
			Environment: s.cfg.Env,
			Error:       "Error interno. Intenta de nuevo.",
			Username:    username,
		})
		return
	}

	auth.SetCookie(w, token, s.cfg.IsProd())
	if user.MustChangePassword {
		http.Redirect(w, r, "/password", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(auth.CookieName); err == nil && cookie.Value != "" {
		if err := s.sessions.Delete(r.Context(), cookie.Value); err != nil {
			s.log.Error("delete session", "err", err)
		}
	}
	auth.ClearCookie(w, s.cfg.IsProd())
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) handlePasswordPage(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFrom(r)
	s.render(w, http.StatusOK, "password", passwordData{
		Title:       "Cambiar contraseña",
		AppName:     "Sistema de Mantenimiento",
		Environment: s.cfg.Env,
		UserName:    session.User.DisplayName,
		RoleLabel:   session.User.Role,
		User:        &session.User,
		CSRF:        session.CSRF,
	})
}

func (s *Server) handlePasswordPost(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFrom(r)
	fail := func(message string) {
		s.render(w, http.StatusUnprocessableEntity, "password", passwordData{
			Title:       "Cambiar contraseña",
			AppName:     "Sistema de Mantenimiento",
			Environment: s.cfg.Env,
			UserName:    session.User.DisplayName,
			RoleLabel:   session.User.Role,
			User:        &session.User,
			CSRF:        session.CSRF,
			Error:       message,
		})
	}

	current := r.FormValue("current")
	newPassword := r.FormValue("new")
	confirm := r.FormValue("confirm")

	_, hash, err := models.UserByUsername(r.Context(), s.db, session.User.Username)
	if err != nil || hash == "" {
		s.log.Error("password change lookup", "err", err)
		fail("Error interno. Intenta de nuevo.")
		return
	}

	if ok, _ := auth.VerifyPassword(current, hash); !ok {
		fail("La contraseña actual no es correcta.")
		return
	}

	switch {
	case len(newPassword) < 8:
		fail("La contraseña nueva debe tener al menos 8 caracteres.")
		return
	case len(newPassword) > 256:
		fail("La contraseña nueva es demasiado larga.")
		return
	case newPassword != confirm:
		fail("Las contraseñas no coinciden.")
		return
	case strings.EqualFold(newPassword, session.User.Username):
		fail("La contraseña no puede ser tu nombre de usuario.")
		return
	case newPassword == current:
		fail("La contraseña nueva debe ser distinta de la actual.")
		return
	}

	newHash, err := auth.HashPassword(newPassword)
	if err != nil {
		s.log.Error("hash new password", "err", err)
		fail("Error interno. Intenta de nuevo.")
		return
	}

	if err := models.UpdatePassword(r.Context(), s.db, session.User.ID, newHash, false); err != nil {
		s.log.Error("update password", "err", err)
		fail("Error interno. Intenta de nuevo.")
		return
	}

	if err := s.sessions.DeleteAllForUser(r.Context(), session.User.ID); err != nil {
		s.log.Error("revoke sessions", "err", err)
	}

	token, err := s.sessions.Create(r.Context(), session.User.ID, clientIP(r), r.UserAgent())
	if err != nil {
		s.log.Error("recreate session", "err", err)
		auth.ClearCookie(w, s.cfg.IsProd())
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	auth.SetCookie(w, token, s.cfg.IsProd())
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
