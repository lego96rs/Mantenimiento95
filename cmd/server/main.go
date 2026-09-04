package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"mantenimiento/internal/auth"
	"mantenimiento/internal/config"
	"mantenimiento/internal/db"
	"mantenimiento/internal/models"
	"mantenimiento/internal/server"
)

func main() {
	createAdmin := flag.String("create-admin", "", "create an admin user with a temporary password and exit")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(log, *createAdmin); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger, createAdmin string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	database, err := db.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer database.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := database.Migrate(ctx); err != nil {
		return err
	}

	if createAdmin != "" {
		return bootstrapAdmin(ctx, database, createAdmin)
	}

	app, err := server.New(cfg, database, log)
	if err != nil {
		return err
	}

	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			if n, err := app.Sessions().DeleteExpired(ctx); err != nil {
				log.Error("session cleanup", "err", err)
			} else if n > 0 {
				log.Info("session cleanup", "deleted", n)
			}

			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()

	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           app.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", cfg.Addr, "env", cfg.Env, "db", cfg.DBPath)
		errCh <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		log.Info("shutting down")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return err
		}

		if err := <-errCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}

func bootstrapAdmin(ctx context.Context, database *db.DB, username string) error {
	username = strings.ToLower(strings.TrimSpace(username))
	if username == "" {
		return errors.New("create-admin: username must not be empty")
	}

	if n, err := models.CountAdmins(ctx, database); err != nil {
		return err
	} else if n > 0 {
		fmt.Fprintf(os.Stderr, "aviso: ya existen %d admin(s) activos; creando otro.\n", n)
	}

	tempPassword, err := auth.GenerateTempPassword()
	if err != nil {
		return err
	}

	hash, err := auth.HashPassword(tempPassword)
	if err != nil {
		return err
	}

	if _, err := models.CreateUser(ctx, database, username, username, models.RoleAdmin, hash, true); err != nil {
		return fmt.Errorf("create-admin: %w", err)
	}

	fmt.Printf("Admin creado.\n  usuario:              %s\n  contraseña temporal:  %s\n", username, tempPassword)
	fmt.Println("Deberá cambiarla en el primer login. Esta contraseña no se vuelve a mostrar.")
	return nil
}
