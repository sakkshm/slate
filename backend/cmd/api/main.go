package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"slate-backend/internal/api"
	"slate-backend/internal/clients"
	"slate-backend/internal/gateway"
	"slate-backend/internal/logging"
	"slate-backend/pkg/config"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {

	// Configure System
	cfg := config.LoadConfig()
	slog.SetDefault(logging.New(cfg))

	c, err := clients.New(cfg)
	if err != nil {
		slog.Error("unable to initialize clients", "error", err)
		os.Exit(1)
	}

	apiEngine := api.NewAPIEngine(cfg, c)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Resolve the real client IP so rate limits key off the user, not the
	// proxy. Without TRUSTED_PROXY_IPS (direct or same-host proxy) we trust
	// RemoteAddr only.
	if len(cfg.TrustedProxyIPs) > 0 {
		r.Use(middleware.ClientIPFromXFF(cfg.TrustedProxyIPs...))
	} else {
		r.Use(middleware.ClientIPFromRemoteAddr)
	}

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSAllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	gw := gateway.New(&gateway.Resolver{
		DB:    c.DB,
		Redis: c.Redis,
		MinIO: c.MinIO,
	}, cfg)

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if _, ok := gateway.IsDeploymentHost(req.Host, cfg.SiteBaseDomain, cfg.ReservedHosts...); ok {
				gw.ServeHTTP(w, req)
				return
			}
			next.ServeHTTP(w, req)
		})
	})

	// The above short-circuits deployment hosts. Everything below (rate limits,
	// security headers) therefore applies only to API traffic.
	r.Use(api.SecurityHeaders(cfg.SiteScheme))
	r.Use(api.RateLimitGlobal(cfg.RateGlobalRps))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World!"))
	})

	r.Get("/health", apiEngine.HandleHealth)

	// Auth Routes (per-IP limit prevents OAuth abuse)
	r.With(api.RateLimitAuth(cfg.RateAuthRpm)).Get(api.AuthInitRoute, apiEngine.HandleInitiateLogin)
	r.With(api.RateLimitAuth(cfg.RateAuthRpm)).Post(api.AuthCallbackRoute, apiEngine.HandleCallback)
	r.Get(api.AuthInstallRoute, apiEngine.HandleInstallURL)

	// Webhook Route (outside auth middleware, verified by HMAC). Generous limit
	// so GitHub retries are never cut off.
	r.With(api.RateLimitWebhook(cfg.RateWebhookRpm)).Post(api.WebhookRoute, apiEngine.HandleGithubWebhook)

	// Protected Routes
	r.Group(func(r chi.Router) {
		r.Use(apiEngine.AuthMiddleware)

		// Auth
		r.Post(api.AuthLogoutRoute, apiEngine.HandleAuthLogout)

		// User
		r.Get(api.UserRoute, apiEngine.HandleGetUserProfile)
		r.Get(api.UserRepoRoute, apiEngine.HandleGetUserRepos)

		// Repo
		r.Get(api.RepoBranchesRoute, apiEngine.HandleGetRepoBranches)
		r.Get(api.RepoContentsRoute, apiEngine.HandleGetRepoContents)

		// Projects
		r.Post(api.ProjectRoute, apiEngine.HandleCreateProject)
		r.Get(api.ProjectRoute, apiEngine.HandleListProjects)
		r.Get(api.ProjectByIDRoute, apiEngine.HandleGetProject)
		r.Put(api.ProjectByIDRoute, apiEngine.HandleUpdateProject)
		r.Delete(api.ProjectByIDRoute, apiEngine.HandleDeleteProject)

		// Builds
		r.With(api.RateLimitBuild(cfg.RateBuildRph)).Post(api.BuildsRoute, apiEngine.HandleTriggerBuild)
		r.Get(api.BuildsRoute, apiEngine.HandleListBuilds)
		r.Get(api.BuildByIDRoute, apiEngine.HandleGetBuild)
		r.Get(api.BuildLogsRoute, apiEngine.HandleBuildLogs)
		r.Post(api.CancelBuildRoute, apiEngine.HandleCancelBuild)

		// Env vars
		r.Get(api.ProjectEnvVarsRoute, apiEngine.HandleListEnvVars)
		r.Put(api.ProjectEnvVarsRoute, apiEngine.HandleUpsertEnvVar)
		r.Delete(api.ProjectEnvVarByKeyRoute, apiEngine.HandleDeleteEnvVar)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		IdleTimeout:       60 * time.Second,
		// No WriteTimeout: SSE log streams and artifact downloads must be able
		// to stay open for their full duration.
	}

	// Graceful shutdown channel
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("api listening", "addr", server.Addr, "env", cfg.Environment)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	<-stop
	slog.Info("shutting down api")
	gateway.StopPruner()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}
