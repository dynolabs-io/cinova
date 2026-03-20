package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/foundrylab-app/cinova/backend/internal/auth"
	"github.com/foundrylab-app/cinova/backend/internal/chat"
	"github.com/foundrylab-app/cinova/backend/internal/config"
	"github.com/foundrylab-app/cinova/backend/internal/graph"
	"github.com/foundrylab-app/cinova/backend/internal/handlers"
	"github.com/foundrylab-app/cinova/backend/internal/langflow"
	"github.com/foundrylab-app/cinova/backend/internal/search"
	"github.com/foundrylab-app/cinova/backend/internal/store"
	"github.com/foundrylab-app/cinova/backend/internal/workflow"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	// Configure zerolog
	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	ctx := context.Background()

	// Connect to Postgres
	pg, err := store.NewPostgresStore(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to postgres")
	}
	defer pg.Close()

	// Connect to Redis
	rdb, err := store.NewRedisStore(ctx, cfg.RedisURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to redis")
	}
	defer rdb.Close()

	// Connect to Neo4j
	neo, err := graph.NewDriver(cfg.Neo4jURI, cfg.Neo4jUser, cfg.Neo4jPassword)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to neo4j")
	}
	defer neo.Close(ctx)

	// Ensure Neo4j schema
	if err := neo.EnsureSchema(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to ensure neo4j schema")
	}

	// Auth
	jwtSvc := auth.NewJWTService(cfg.JWTSecret, cfg.JWTAccessTTLSec, cfg.JWTRefreshTTLSec)
	authHandler := auth.NewHandler(pg, rdb, jwtSvc)

	// Graph repository
	movieRepo := graph.NewMovieRepository(neo)

	// Handlers
	movieHandler := handlers.NewMovieHandler(movieRepo, rdb)
	interactionHandler := handlers.NewInteractionHandler(movieRepo, pg, rdb)
	recommendHandler := handlers.NewRecommendHandler(movieRepo, rdb)
	personHandler := handlers.NewPersonHandler(movieRepo)
	pushTokenHandler := handlers.NewPushTokenHandler(pg)
	scoringHandler := handlers.NewScoringHandler(movieRepo)
	sttHandler := handlers.NewSTTHandler(cfg)

	// Search
	searchHandler := search.NewHandler(neo, rdb, cfg)

	// Chat service
	chatSvc := chat.New(movieRepo, pg, cfg)

	// Langflow client (optional — preferred runtime when LANGFLOW_URL + LANGFLOW_FLOW_ID are set)
	var langflowClient *langflow.Client
	if cfg.LangflowURL != "" && cfg.LangflowFlowID != "" {
		langflowClient = langflow.New(cfg.LangflowURL, cfg.LangflowFlowID, cfg.LangflowAPIKey)
		log.Info().Str("url", cfg.LangflowURL).Str("flow_id", cfg.LangflowFlowID).Msg("langflow: chat pipeline enabled")
	}

	// Temporal worker (optional — disabled when TEMPORAL_ADDRESS is unset or Langflow is active)
	temporalShutdown, temporalClient, err := workflow.Start(cfg.TemporalAddress, chatSvc)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to start temporal worker")
	}
	defer temporalShutdown()

	candidatesHandler := handlers.NewCandidatesHandler(chatSvc)
	chatHandler := handlers.NewChatHandler(chatSvc, pg, langflowClient, temporalClient)

	// Router
	r := chi.NewRouter()

	// CORS — allow any origin for mobile clients
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Device-ID", "X-Session-ID"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Standard middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(zerologMiddleware)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(90 * time.Second))

	// Internal routes (cluster-only — not exposed via ingress)
	r.Route("/internal/v1", func(r chi.Router) {
		r.Post("/candidates", candidatesHandler.GetCandidates)
	})

	// Health endpoints (no auth)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		rCtx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		if err := pg.Ping(rCtx); err != nil {
			log.Error().Err(err).Msg("readyz: postgres not ready")
			http.Error(w, `{"status":"not ready","reason":"postgres"}`, http.StatusServiceUnavailable)
			return
		}
		if err := rdb.Ping(rCtx); err != nil {
			log.Error().Err(err).Msg("readyz: redis not ready")
			http.Error(w, `{"status":"not ready","reason":"redis"}`, http.StatusServiceUnavailable)
			return
		}
		if err := neo.Ping(rCtx); err != nil {
			log.Error().Err(err).Msg("readyz: neo4j not ready")
			http.Error(w, `{"status":"not ready","reason":"neo4j"}`, http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	})

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		// Auth routes (no session required)
		r.Route("/auth", func(r chi.Router) {
			r.Post("/anonymous", authHandler.AnonymousHandler)
			r.Post("/signup", authHandler.SignupHandler)
			r.Post("/login", authHandler.LoginHandler)
			r.Post("/refresh", authHandler.RefreshHandler)
		})

		// Public content routes (no auth required)
		r.Get("/trending", movieHandler.GetTrending)
		r.Get("/popular", movieHandler.GetPopular)
		r.Get("/discover/reels", movieHandler.GetReels)
		r.Get("/movie/{id}", movieHandler.GetMovie)
		r.Get("/movie/{id}/providers", movieHandler.GetMovieProviders)
		r.Get("/tv/{id}", movieHandler.GetTV)
		r.Get("/person/{id}", personHandler.GetPerson)

		// Protected routes (require valid JWT)
		r.Group(func(r chi.Router) {
			r.Use(authHandler.ExtractSession)
			r.Get("/search", searchHandler.SearchHandler)
			r.Get("/recommend", recommendHandler.GetRecommendations)

			// User interaction routes
			r.Route("/me", func(r chi.Router) {
				r.Post("/rate", interactionHandler.Rate)
				r.Post("/save", interactionHandler.Save)
				r.Delete("/save", interactionHandler.Unsave)
				r.Post("/dismiss", interactionHandler.Dismiss)
				r.Get("/watchlist", interactionHandler.GetWatchlist)
				r.Post("/push-token", pushTokenHandler.RegisterPushToken)
				r.Get("/scoring-profile", scoringHandler.GetScoringProfile)
				r.Put("/scoring-profile", scoringHandler.SetScoringProfile)
				r.Post("/chat", chatHandler.Chat)
				r.Post("/chat/stream", chatHandler.ChatStream)
				r.Post("/stt", sttHandler.Transcribe)
			})
		})
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in background
	serverErr := make(chan error, 1)
	go func() {
		log.Info().Str("addr", srv.Addr).Msg("cinova api listening")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	select {
	case sig := <-quit:
		log.Info().Str("signal", sig.String()).Msg("shutdown signal received")
	case err := <-serverErr:
		log.Fatal().Err(err).Msg("server error")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("graceful shutdown failed")
	}
	log.Info().Msg("server stopped")
}

// zerologMiddleware logs each request with zerolog.
func zerologMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		log.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("remote", r.RemoteAddr).
			Str("request_id", middleware.GetReqID(r.Context())).
			Int("status", ww.Status()).
			Int("bytes", ww.BytesWritten()).
			Dur("duration", time.Since(start)).
			Msg("request")
	})
}
