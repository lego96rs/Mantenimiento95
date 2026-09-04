package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"mantenimiento/internal/auth"
	"mantenimiento/internal/config"
	"mantenimiento/internal/db"
	"mantenimiento/internal/middleware"
	"mantenimiento/web"
)

type Server struct {
	cfg         config.Config
	db          *db.DB
	log         *slog.Logger
	sessions    *auth.Sessions
	userLimiter *auth.Limiter
	ipLimiter   *auth.Limiter
	pages       map[string]*template.Template
}

type homeViewData struct {
	Title       string
	AppName     string
	Environment string
	UserName    string
	RoleLabel   string
	CSRF        string
}

func New(cfg config.Config, database *db.DB, log *slog.Logger) (*Server, error) {
	pages, err := parsePages()
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}

	return &Server{
		cfg:         cfg,
		db:          database,
		log:         log,
		sessions:    auth.NewSessions(database),
		userLimiter: auth.NewLimiter(5, 15*time.Minute),
		ipLimiter:   auth.NewLimiter(20, 15*time.Minute),
		pages:       pages,
	}, nil
}

func (s *Server) Sessions() *auth.Sessions {
	return s.sessions
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	staticFiles, err := fs.Sub(web.FS, "static")
	if err != nil {
		panic(err)
	}

	mux.Handle("GET /static/", http.StripPrefix("/static/", cacheStatic(http.FileServerFS(staticFiles))))
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /login", s.handleLoginPage)
	mux.HandleFunc("POST /login", s.handleLoginPost)
	mux.HandleFunc("POST /logout", s.handleLogout)

	requireUser := func(handler http.HandlerFunc) http.Handler {
		return middleware.RequireUser(handler)
	}
	requireAdmin := func(handler http.HandlerFunc) http.Handler {
		return middleware.RequireAdmin(handler)
	}

	mux.Handle("GET /{$}", requireUser(s.handleHome))
	mux.Handle("GET /password", requireUser(s.handlePasswordPage))
	mux.Handle("POST /password", requireUser(s.handlePasswordPost))
	mux.Handle("GET /admin", requireAdmin(s.handleAdminHome))

	return middleware.Chain(mux,
		middleware.Recover(s.log),
		middleware.RequestLog(s.log),
		middleware.SecurityHeaders,
		middleware.Auth(s.sessions),
		middleware.CSRF,
	)
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	w.Header().Set("Content-Type", "application/json")
	if err := s.db.Ping(ctx); err != nil {
		s.log.Error("health check failed", "err", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "db unavailable"})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFrom(r)

	s.render(w, http.StatusOK, "home", homeViewData{
		Title:       "Inicio",
		AppName:     "Sistema de Mantenimiento",
		Environment: s.cfg.Env,
		UserName:    session.User.DisplayName,
		RoleLabel:   session.User.Role,
		CSRF:        session.CSRF,
	})
}

func (s *Server) handleAdminHome(w http.ResponseWriter, r *http.Request) {
	session, _ := middleware.SessionFrom(r)
	s.render(w, http.StatusOK, "admin", homeViewData{
		Title:       "Administración",
		AppName:     "Sistema de Mantenimiento",
		Environment: s.cfg.Env,
		UserName:    session.User.DisplayName,
		RoleLabel:   session.User.Role,
		CSRF:        session.CSRF,
	})
}

func (s *Server) render(w http.ResponseWriter, status int, page string, data any) {
	tmpl, ok := s.pages[page]
	if !ok {
		s.log.Error("unknown template", "page", page)
		http.Error(w, "error interno", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		s.log.Error("render template", "page", page, "err", err)
	}
}

func parsePages() (map[string]*template.Template, error) {
	staticVersion, err := assetVersion()
	if err != nil {
		return nil, err
	}

	functions := template.FuncMap{
		"asset": func(path string) string {
			return path + "?v=" + staticVersion
		},
	}

	layout, err := template.New("layout.tmpl").Funcs(functions).ParseFS(web.FS, "templates/layout.tmpl")
	if err != nil {
		return nil, err
	}

	paths, err := fs.Glob(web.FS, "templates/*.tmpl")
	if err != nil {
		return nil, err
	}

	pages := make(map[string]*template.Template)
	for _, path := range paths {
		name := strings.TrimSuffix(strings.TrimPrefix(path, "templates/"), ".tmpl")
		if name == "layout" {
			continue
		}

		page, err := layout.Clone()
		if err != nil {
			return nil, err
		}

		if _, err := page.ParseFS(web.FS, path); err != nil {
			return nil, err
		}

		pages[name] = page
	}

	return pages, nil
}

func assetVersion() (string, error) {
	hash := sha256.New()
	err := fs.WalkDir(web.FS, "static", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}

		content, err := web.FS.ReadFile(path)
		if err != nil {
			return err
		}

		_, _ = io.WriteString(hash, path)
		_, _ = hash.Write(content)
		return nil
	})
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil))[:12], nil
}

func cacheStatic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Has("v") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "public, max-age=300")
		}
		next.ServeHTTP(w, r)
	})
}
