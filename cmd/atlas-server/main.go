package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"

	"github.com/nesbite/atlas/internal/auth"
	"github.com/nesbite/atlas/internal/catalog"
	"github.com/nesbite/atlas/internal/dependency"
	"github.com/nesbite/atlas/internal/graph"
	"github.com/nesbite/atlas/internal/impact"
	"github.com/nesbite/atlas/internal/org"
	"github.com/nesbite/atlas/internal/ownership"
	"github.com/nesbite/atlas/internal/platform/config"
	"github.com/nesbite/atlas/internal/platform/database"
	"github.com/nesbite/atlas/internal/vuln"
	"github.com/nesbite/atlas/migrations"
)

// orgStoreResolver adapts org.OrgStore to satisfy dependency.OrgResolver.
// It resolves a slug to an org UUID using the org store.
type orgStoreResolver struct {
	store org.OrgStore
}

func (r *orgStoreResolver) GetOrgIDBySlug(ctx context.Context, slug string) (uuid.UUID, bool, error) {
	o, err := r.store.GetOrgBySlug(ctx, slug)
	if err != nil {
		return uuid.Nil, false, err
	}
	if o == nil {
		return uuid.Nil, false, nil
	}
	return o.ID, true, nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := database.RunMigrations(ctx, pool, migrations.FS); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	authStore := auth.NewStore(pool)
	authHandler := auth.NewHandler(
		cfg.GitHubClientID,
		cfg.GitHubClientSecret,
		cfg.JWTSecret,
		cfg.WebURL,
		authStore,
	)

	orgStore := org.NewStore(pool)
	catalogStore := catalog.NewStore(pool)
	catalogHandler := catalog.NewHandler(catalogStore, &orgStoreResolver{store: orgStore})

	depStore := dependency.NewStore(pool)
	depService := dependency.NewService(depStore)
	depHandler := dependency.NewHandler(depStore, &orgStoreResolver{store: orgStore})

	ownershipStore := ownership.NewStore(pool)
	ownershipService := ownership.NewService(ownershipStore)
	ownershipHandler := ownership.NewHandler(ownershipStore, &orgStoreResolver{store: orgStore})

	impactStore := impact.NewStore(pool)
	impactHandler := impact.NewHandler(impactStore, &orgStoreResolver{store: orgStore})

	graphStore := graph.NewStore(pool)
	graphHandler := graph.NewHandler(graphStore, &orgStoreResolver{store: orgStore})

	vulnStore := vuln.NewStore(pool)
	vulnService := vuln.NewService(vulnStore, vuln.NewOSVClient())
	vulnHandler := vuln.NewHandler(vulnStore, &orgStoreResolver{store: orgStore})

	orgHandler := org.NewHandler(
		orgStore,
		catalogStore,
		depService,
		ownershipService,
		vulnService,
		cfg.GitHubAppID,
		cfg.GitHubAppPrivateKey,
		cfg.GitHubWebhookSecret,
	)

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{cfg.WebURL},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/version", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"version":"0.1.0","name":"Atlas"}`))
		})

		r.Get("/auth/github/login", authHandler.HandleLogin)
		r.Get("/auth/github/callback", authHandler.HandleCallback)
		r.Post("/auth/refresh", authHandler.HandleRefresh)

		r.Post("/webhooks/github", orgHandler.HandleGitHubWebhook)

		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(cfg.JWTSecret))
			r.Get("/auth/me", authHandler.HandleMe)
			r.Route("/orgs", orgHandler.Routes())
			r.Get("/orgs/{slug}/repos", catalogHandler.HandleListRepos)
			r.Get("/orgs/{slug}/repos/{name}", catalogHandler.HandleGetRepo)
			r.Get("/orgs/{slug}/repos/{name}/dependencies", depHandler.HandleGetRepoDependencies)
			r.Get("/orgs/{slug}/dependencies", depHandler.HandleListDependencies)
			r.Get("/orgs/{slug}/dependencies/{ecosystem}/*", depHandler.HandleGetDependency)
			r.Get("/orgs/{slug}/ownership", ownershipHandler.HandleListOwnership)
			r.Get("/orgs/{slug}/ownership/{repo}", ownershipHandler.HandleGetRepoOwnership)
			r.Post("/orgs/{slug}/impact", impactHandler.HandleAnalyzeImpact)
			r.Get("/orgs/{slug}/graph", graphHandler.HandleGetGraph)
			r.Get("/orgs/{slug}/vulnerabilities", vulnHandler.HandleListVulnerabilities)
			r.Get("/orgs/{slug}/vulnerabilities/{id}", vulnHandler.HandleGetVulnerability)
		})
	})

	addr := ":" + cfg.ServerPort
	server := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("starting atlas server", "addr", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	server.Shutdown(shutdownCtx)
}
