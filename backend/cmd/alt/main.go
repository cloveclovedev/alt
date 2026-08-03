// Command alt runs the web application, background worker, or health check.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/cloveclovedev/alt/internal/ai"
	"github.com/cloveclovedev/alt/internal/core/config"
	"github.com/cloveclovedev/alt/internal/core/database"
	"github.com/cloveclovedev/alt/internal/core/httpserver"
	"github.com/cloveclovedev/alt/internal/core/logging"
	"github.com/cloveclovedev/alt/internal/core/objectstore"
	"github.com/cloveclovedev/alt/internal/identity"
	"github.com/cloveclovedev/alt/internal/nutrition"
	"github.com/cloveclovedev/alt/internal/planning"
	"github.com/cloveclovedev/alt/internal/platform/openrouter"
	"github.com/cloveclovedev/alt/internal/routine"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: alt <web|healthcheck>")
		os.Exit(2)
	}
	command := os.Args[1]
	cfg, err := config.Load(command != "healthcheck")
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}
	if command == "healthcheck" {
		os.Exit(runHealthcheck(cfg))
	}

	logger := logging.New(cfg.LogLevel)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	switch command {
	case "web":
		if err := runWeb(ctx, cfg, logger); err != nil {
			logger.Error("web command failed", "error", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", command)
		os.Exit(2)
	}
}

func runWeb(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	pool, err := database.Connect(ctx, cfg.DatabaseURL, cfg.MaxConnections)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := identity.NewStore(pool).BootstrapLocalUser(
		ctx,
		cfg.UserID,
		cfg.UserDisplayName,
		cfg.UserTimezone,
	); err != nil {
		return err
	}

	planningStore := planning.NewStore(pool)
	service, err := planning.NewService(
		planningStore,
		cfg.UserID,
		cfg.UserTimezone,
	)
	if err != nil {
		return err
	}
	planningHandler, err := planning.NewHandler(service, logger)
	if err != nil {
		return err
	}
	routineService, err := routine.NewService(routine.NewStore(pool), cfg.UserID, cfg.UserTimezone)
	if err != nil {
		return err
	}
	routineHandler, err := routine.NewHandler(routineService, logger)
	if err != nil {
		return err
	}
	githubClient := planning.NewGitHubClient()
	openRouterClient := openrouter.New(cfg.OpenRouterAPIKey)
	aiService := ai.NewService(ai.NewStore(pool), openRouterClient, cfg.UserID)
	var calendarClient *planning.GoogleCalendarClient
	var calendarReader planning.CalendarPlanningReader
	if cfg.GoogleOAuthClientID != "" && cfg.GoogleOAuthClientSecret != "" && cfg.GoogleOAuthRedirectURL != "" && cfg.CalendarTokenEncryptionKey != "" {
		cipher, cipherErr := planning.NewTokenCipher(cfg.CalendarTokenEncryptionKey)
		if cipherErr != nil {
			return cipherErr
		}
		calendarClient = planning.NewGoogleCalendarClient(planningStore, cipher, planning.CalendarOAuthConfig{
			ClientID: cfg.GoogleOAuthClientID, ClientSecret: cfg.GoogleOAuthClientSecret, RedirectURL: cfg.GoogleOAuthRedirectURL,
		})
		calendarReader = calendarClient
	}
	dailyService, err := planning.NewDailyService(
		planningStore, cfg.UserID, cfg.UserTimezone,
		planning.NewDailyGatherer(planningStore, routineService, githubClient, calendarReader),
		aiService,
	)
	if err != nil {
		return err
	}
	dailyHandler, err := planning.NewDailyHandler(dailyService, logger)
	if err != nil {
		return err
	}
	settingsHandler, err := planning.NewSettingsHandler(planningStore, cfg.UserID, githubClient, calendarClient, logger)
	if err != nil {
		return err
	}
	aiSettingsHandler, err := ai.NewSettingsHandler(aiService, logger)
	if err != nil {
		return err
	}
	photoStore, err := objectstore.New(objectstore.Config{
		Endpoint:        cfg.ObjectStoreEndpoint,
		Region:          cfg.ObjectStoreRegion,
		Bucket:          cfg.ObjectStoreBucket,
		AccessKeyID:     cfg.ObjectStoreAccessKeyID,
		SecretAccessKey: cfg.ObjectStoreSecretAccessKey,
	})
	if err != nil {
		return err
	}
	// Pass a true nil interface when object storage is unconfigured; a typed-nil
	// *Store would read as non-nil and panic when the photo path calls it.
	var nutritionPhotos nutrition.PhotoStore
	if photoStore != nil {
		nutritionPhotos = photoStore
	}
	nutritionService, err := nutrition.NewService(nutrition.NewStore(pool), cfg.UserID, cfg.UserTimezone, aiService, nutritionPhotos)
	if err != nil {
		return err
	}
	nutritionHandler, err := nutrition.NewHandler(nutritionService, logger)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	planningHandler.Register(mux)
	dailyHandler.Register(mux)
	settingsHandler.Register(mux)
	nutritionHandler.Register(mux)
	aiSettingsHandler.Register(mux)
	routineHandler.Register(mux)
	mux.HandleFunc("GET /livez", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		readyCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(readyCtx); err != nil {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})

	server := httpserver.New(
		cfg.Port,
		httpserver.Middleware(logger, httpserver.SameOrigin(mux)),
	)
	return httpserver.Run(ctx, logger, server, cfg.ShutdownTimeout)
}

func runHealthcheck(cfg config.Config) int {
	client := &http.Client{Timeout: 3 * time.Second}
	response, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/livez", cfg.Port))
	if err != nil {
		return 1
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
