package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"slate-backend/internal/api"
	"slate-backend/internal/clients"
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
	c, err := clients.New(cfg)
	if err != nil {
		log.Fatalf("Unable to initialize clients: %v", err)
	}

	apiEngine := api.NewAPIEngine(cfg, c)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World!"))
	})

	// Auth Routes
	r.Get(api.AuthInitRoute, apiEngine.HandleInitiateLogin)
	r.Post(api.AuthCallbackRoute, apiEngine.HandleCallback)
	r.Get(api.AuthInstallRoute, apiEngine.HandleInstallURL)

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
		r.Post(api.BuildsRoute, apiEngine.HandleTriggerBuild)
		r.Get(api.BuildsRoute, apiEngine.HandleListBuilds)
		r.Get(api.BuildByIDRoute, apiEngine.HandleGetBuild)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{Addr: ":" + port, Handler: r}

	// Graceful shutdown channel
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}
