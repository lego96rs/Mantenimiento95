package middleware

import (
	"context"
	"crypto/subtle"
	"net/http"

	"mantenimiento/internal/auth"
)

type ctxKey int

const sessionKey ctxKey = 0

func SessionFrom(r *http.Request) (*auth.Session, bool) {
	session, ok := r.Context().Value(sessionKey).(*auth.Session)
	return session, ok
}

func Auth(sessions *auth.Sessions) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cookie, err := r.Cookie(auth.CookieName); err == nil && cookie.Value != "" {
				session, err := sessions.Validate(r.Context(), cookie.Value)
				if err != nil {
					http.Error(w, "error interno", http.StatusInternalServerError)
					return
				}
				if session != nil {
					r = r.WithContext(context.WithValue(r.Context(), sessionKey, session))
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, ok := SessionFrom(r)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		if session.User.MustChangePassword && r.URL.Path != "/password" && r.URL.Path != "/logout" {
			http.Redirect(w, r, "/password", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireAdmin(next http.Handler) http.Handler {
	return RequireUser(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _ := SessionFrom(r)
		if !session.User.IsAdmin() {
			http.Error(w, "acceso denegado", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}))
}

func CSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}

		if site := r.Header.Get("Sec-Fetch-Site"); site != "" && site != "same-origin" && site != "none" {
			http.Error(w, "solicitud rechazada", http.StatusForbidden)
			return
		}

		if session, ok := SessionFrom(r); ok {
			got := r.FormValue("csrf")
			if subtle.ConstantTimeCompare([]byte(got), []byte(session.CSRF)) != 1 {
				http.Error(w, "solicitud rechazada", http.StatusForbidden)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
