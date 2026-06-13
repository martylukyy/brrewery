package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	gcrypt "github.com/GehirnInc/crypt"
	yescrypt "github.com/openwall/yescrypt-go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/autobrr/brrewery/internal/api"
	appsdomain "github.com/autobrr/brrewery/internal/apps"
	"github.com/autobrr/brrewery/internal/apps/ansible"
	"github.com/autobrr/brrewery/internal/apps/detect"
	"github.com/autobrr/brrewery/internal/apps/jobs"
	"github.com/autobrr/brrewery/internal/auth"
	"github.com/autobrr/brrewery/internal/buildinfo"
	"github.com/autobrr/brrewery/internal/paths"
	"github.com/autobrr/brrewery/internal/system"
	"github.com/autobrr/brrewery/internal/vnstat"
	webapp "github.com/autobrr/brrewery/internal/web"
)

var errOSPasswordVerificationFailed = errors.New("OS password verification failed")

func main() {
	root := &cobra.Command{
		Use:   "brrewery",
		Short: "App management web interface",
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
			appsService := appsdomain.NewServiceWithDeps(
				detect.NewEvaluator(),
				ansible.NewRunner(paths.ResolveAnsibleRoot()),
				jobs.NewStoreAt(paths.ResolveJobsDir()),
			)

			embedFS, err := webapp.DistFS()
			if err != nil {
				return fmt.Errorf("load embedded frontend: %w", err)
			}

			server := api.NewServer(
				&logger,
				authService,
				session,
				appsService,
				system.NewCollector(),
				vnstat.NewCollector(),
				embedFS,
			)
			httpServer := &http.Server{
				Addr:              paths.ListenAddress(),
				Handler:           server.Handler(),
				ReadHeaderTimeout: 10 * time.Second,
				ReadTimeout:       30 * time.Second,
				WriteTimeout:      30 * time.Second,
				IdleTimeout:       60 * time.Second,
			}

			errCh := make(chan error, 1)
			go func() {
				logger.Info().Str("addr", paths.ListenAddress()).Msg("starting server")
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
	if term.IsTerminal(int(os.Stdin.Fd())) {
		username, err = promptOSUserSelection()
		if err != nil {
			return "", "", err
		}
	} else {
		fmt.Print("Username: ")
		if _, err = fmt.Scanln(&username); err != nil {
			return "", "", fmt.Errorf("read username: %w", err)
		}
		username = strings.TrimSpace(username)
	}

	if username == "" {
		return "", "", errors.New("username cannot be empty")
	}

	const maxPasswordAttempts = 3
	for attempt := 1; attempt <= maxPasswordAttempts; attempt++ {
		fmt.Printf("Password for '%s': ", username)
		password, err = readPassword()
		if err != nil {
			return "", "", err
		}
		fmt.Println()

		if err := verifyOSPassword(username, password); err != nil {
			if !errors.Is(err, errOSPasswordVerificationFailed) {
				return "", "", err
			}
			if attempt == maxPasswordAttempts {
				return "", "", err
			}
			fmt.Fprintf(os.Stderr, "Wrong password, try again (%d/%d)\n", attempt, maxPasswordAttempts)
			continue
		}
		break
	}

	return username, password, nil
}

func promptOSUserSelection() (string, error) {
	users, err := listOSUsers()
	if err != nil {
		return "", err
	}
	if len(users) == 0 {
		return "", errors.New("no OS users found")
	}

	fmt.Println("Select OS user for initial admin account:")
	for i, user := range users {
		fmt.Printf("  %d) %s\n", i+1, user)
	}

	defaultUser, hasDefault, err := userByUID(1000)
	if err != nil {
		return "", err
	}
	if hasDefault {
		fmt.Print("Choice [1-", len(users), ", default uid 1000 (", defaultUser, ")]: ")
	} else {
		fmt.Print("Choice [1-", len(users), "]: ")
	}

	choiceReader := bufio.NewReader(os.Stdin)
	choiceRaw, err := choiceReader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read user selection: %w", err)
	}
	choiceRaw = strings.TrimSpace(choiceRaw)
	if choiceRaw == "" {
		if hasDefault {
			return defaultUser, nil
		}
		return "", errors.New("selection cannot be empty")
	}

	choice, err := strconv.Atoi(choiceRaw)
	if err != nil {
		return "", errors.New("selection must be a number")
	}
	if choice < 1 || choice > len(users) {
		return "", fmt.Errorf("selection must be between 1 and %d", len(users))
	}

	return users[choice-1], nil
}

func listOSUsers() ([]string, error) {
	passwdBytes, err := os.ReadFile("/etc/passwd")
	if err != nil {
		return nil, fmt.Errorf("read /etc/passwd: %w", err)
	}

	seen := make(map[string]struct{})
	users := make([]string, 0)
	for _, line := range strings.Split(string(passwdBytes), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) < 7 {
			continue
		}

		username := strings.TrimSpace(parts[0])
		if username == "" {
			continue
		}
		uid, err := strconv.Atoi(strings.TrimSpace(parts[2]))
		if err != nil {
			continue
		}
		shell := strings.TrimSpace(parts[6])
		if uid < 1000 || !hasLoginShell(shell) {
			continue
		}
		if _, exists := seen[username]; exists {
			continue
		}
		seen[username] = struct{}{}
		users = append(users, username)
	}
	sort.Strings(users)
	return users, nil
}

func hasLoginShell(shell string) bool {
	switch shell {
	case "", "/usr/sbin/nologin", "/sbin/nologin", "/bin/false", "nologin", "false":
		return false
	default:
		return true
	}
}

func userByUID(wantUID int) (string, bool, error) {
	passwdBytes, err := os.ReadFile("/etc/passwd")
	if err != nil {
		return "", false, fmt.Errorf("read /etc/passwd: %w", err)
	}

	for _, line := range strings.Split(string(passwdBytes), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) < 7 {
			continue
		}

		uid, err := strconv.Atoi(strings.TrimSpace(parts[2]))
		if err != nil || uid != wantUID {
			continue
		}

		username := strings.TrimSpace(parts[0])
		shell := strings.TrimSpace(parts[6])
		if username == "" || !hasLoginShell(shell) {
			return "", false, nil
		}
		return username, true, nil
	}

	return "", false, nil
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

func verifyOSPassword(username, password string) error {
	if username == "" {
		return errors.New("username cannot be empty")
	}
	if password == "" {
		return errors.New("password cannot be empty")
	}

	shadowBytes, err := os.ReadFile("/etc/shadow")
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			return errors.New("cannot verify OS password: permission denied reading shadow password database")
		}
		return errors.New("cannot verify OS password: verification backend unavailable")
	}

	hashValue, found := shadowHashForUser(string(shadowBytes), username)
	if !found {
		return fmt.Errorf("OS user '%s' not found", username)
	}
	if isUnusableShadowHash(hashValue) {
		return fmt.Errorf("OS user '%s' does not have a usable password", username)
	}

	ok, err := verifyShadowHash(password, hashValue)
	if err != nil {
		return errors.New("cannot verify OS password: verification backend unavailable")
	}
	if !ok {
		return errOSPasswordVerificationFailed
	}

	return nil
}

func shadowHashForUser(shadowContent, username string) (string, bool) {
	for _, line := range strings.Split(shadowContent, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) < 2 {
			continue
		}
		if parts[0] != username {
			continue
		}
		return parts[1], true
	}
	return "", false
}

func isUnusableShadowHash(hashValue string) bool {
	if hashValue == "" {
		return true
	}
	if hashValue == "!" || hashValue == "*" || hashValue == "x" {
		return true
	}
	return strings.HasPrefix(hashValue, "!") || strings.HasPrefix(hashValue, "*")
}

func verifyShadowHash(password, hashValue string) (bool, error) {
	if isYescryptHash(hashValue) {
		computed, err := yescrypt.Hash([]byte(password), []byte(hashValue))
		if err != nil {
			return false, err
		}
		return string(computed) == hashValue, nil
	}

	if !isSupportedClassicCryptHash(hashValue) {
		return false, errors.New("unsupported shadow hash format")
	}

	crypter := gcrypt.NewFromHash(hashValue)
	if crypter == nil {
		return false, errors.New("unsupported shadow hash format")
	}

	if err := crypter.Verify(hashValue, []byte(password)); err != nil {
		if errors.Is(err, gcrypt.ErrKeyMismatch) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func isYescryptHash(hashValue string) bool {
	return strings.HasPrefix(hashValue, "$y$") || strings.HasPrefix(hashValue, "$gy$")
}

func isSupportedClassicCryptHash(hashValue string) bool {
	return strings.HasPrefix(hashValue, "$1$") ||
		strings.HasPrefix(hashValue, "$2a$") ||
		strings.HasPrefix(hashValue, "$2y$") ||
		strings.HasPrefix(hashValue, "$5$") ||
		strings.HasPrefix(hashValue, "$6$")
}
