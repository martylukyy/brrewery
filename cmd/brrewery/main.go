package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/autobrr/brrewery/internal/api"
	"github.com/autobrr/brrewery/internal/auth"
	"github.com/autobrr/brrewery/internal/buildinfo"
	pkgdomain "github.com/autobrr/brrewery/internal/packages"
	"github.com/autobrr/brrewery/internal/paths"
	"github.com/autobrr/brrewery/internal/system"
	"github.com/autobrr/brrewery/internal/vnstat"
	webapp "github.com/autobrr/brrewery/internal/web"
)

func main() {
	root := &cobra.Command{
		Use:   "brrewery",
		Short: "Package management web interface",
	}
	root.AddCommand(runServe())
	root.AddCommand(runVersion())
	root.AddCommand(runCreateAdmin())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runServe() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the HTTP server",
		RunE: func(_ *cobra.Command, _ []string) error {
			logger := setupLogger()

			secret, err := auth.LoadOrCreateSessionSecret(paths.SessionSecretPath)
			if err != nil {
				return err
			}

			session := auth.NewSessionManager(secret)
			store := auth.NewFileStore(paths.UserStorePath)
			authService := auth.NewService(store, session)
			packagesService := pkgdomain.NewService()

			embedFS, err := webapp.DistFS()
			if err != nil {
				return fmt.Errorf("load embedded frontend: %w", err)
			}

			server := api.NewServer(
				&logger,
				authService,
				session,
				packagesService,
				system.NewCollector(),
				vnstat.NewCollector(),
				embedFS,
			)
			httpServer := &http.Server{
				Addr:              paths.BackendListenAddress,
				Handler:           server.Handler(),
				ReadHeaderTimeout: 10 * time.Second,
				ReadTimeout:       30 * time.Second,
				WriteTimeout:      30 * time.Second,
				IdleTimeout:       60 * time.Second,
			}

			errCh := make(chan error, 1)
			go func() {
				logger.Info().Str("addr", paths.BackendListenAddress).Msg("starting server")
				if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					errCh <- err
				}
			}()

			sigCh := make(chan os.Signal, 2)
			signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
			defer signal.Stop(sigCh)

			select {
			case sig := <-sigCh:
				logger.Info().Str("signal", sig.String()).Msg("shutdown requested")
				fmt.Fprintln(os.Stderr, "shutting down...")
			case err := <-errCh:
				return err
			}

			return shutdownHTTPServer(httpServer, sigCh, &logger)
		},
	}
}

func runVersion() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println(buildinfo.Version)
		},
	}
}

func runCreateAdmin() *cobra.Command {
	return &cobra.Command{
		Use:   "create-admin",
		Short: "Create the initial admin user",
		RunE: func(cmd *cobra.Command, _ []string) error {
			store := auth.NewFileStore(paths.UserStorePath)
			has, err := store.HasUsers()
			if err != nil {
				return err
			}
			if has {
				cmd.Println("Admin user already exists.")
				return nil
			}

			username, password, err := promptCredentials()
			if err != nil {
				return err
			}

			secret, err := auth.LoadOrCreateSessionSecret(paths.SessionSecretPath)
			if err != nil {
				return err
			}
			authService := auth.NewService(store, auth.NewSessionManager(secret))

			user, err := authService.CreateAdmin(username, password)
			if err != nil {
				return err
			}

			cmd.Printf("Admin user '%s' created (id=%s)\n", user.Username, user.ID)
			return nil
		},
	}
}

func shutdownHTTPServer(httpServer *http.Server, sigCh <-chan os.Signal, logger *zerolog.Logger) error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- httpServer.Shutdown(shutdownCtx)
	}()

	select {
	case err := <-done:
		if err != nil {
			logger.Warn().Err(err).Msg("graceful shutdown failed, forcing close")
			if closeErr := httpServer.Close(); closeErr != nil {
				return closeErr
			}
			return err
		}
		logger.Info().Msg("server stopped")
		return nil
	case sig := <-sigCh:
		logger.Warn().Str("signal", sig.String()).Msg("forcing shutdown")
		fmt.Fprintln(os.Stderr, "forcing shutdown...")
		_ = httpServer.Close()
		<-done
		return nil
	case <-shutdownCtx.Done():
		logger.Warn().Msg("shutdown timed out, forcing close")
		fmt.Fprintln(os.Stderr, "shutdown timed out, forcing close...")
		_ = httpServer.Close()
		<-done
		return shutdownCtx.Err()
	}
}

func setupLogger() zerolog.Logger {
	logFile, err := os.OpenFile(paths.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o640)
	if err != nil {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
		log.Warn().Err(err).Str("path", paths.LogFile).Msg("logging to stdout")
		return log.Logger
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: logFile, NoColor: true, TimeFormat: time.RFC3339})
	return log.Logger
}

func promptCredentials() (username, password string, err error) {
	fmt.Print("Username: ")
	if _, err = fmt.Scanln(&username); err != nil {
		return "", "", fmt.Errorf("read username: %w", err)
	}
	username = strings.TrimSpace(username)
	if username == "" {
		return "", "", errors.New("username cannot be empty")
	}

	fmt.Print("Password: ")
	password, err = readPassword()
	if err != nil {
		return "", "", err
	}
	fmt.Println()

	if len(password) < 8 {
		return "", "", errors.New("password must be at least 8 characters")
	}

	return username, password, nil
}

func readPassword() (string, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		var password string
		if _, err := fmt.Scanln(&password); err != nil {
			return "", fmt.Errorf("read password: %w", err)
		}
		return password, nil
	}

	bytes, err := term.ReadPassword(fd)
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}
	return string(bytes), nil
}
