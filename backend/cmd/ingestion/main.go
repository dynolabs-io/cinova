package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/foundrylab-app/cinova/backend/internal/config"
	"github.com/foundrylab-app/cinova/backend/internal/enrichment"
	"github.com/foundrylab-app/cinova/backend/internal/graph"
	"github.com/foundrylab-app/cinova/backend/internal/models"
	"github.com/foundrylab-app/cinova/backend/internal/scoring"
	"github.com/foundrylab-app/cinova/backend/internal/tmdb"
)

func main() {
	mode := flag.String("mode", "delta", "ingestion mode: full or delta")
	country := flag.String("country", "US", "ISO 3166-1 alpha-2 country code for streaming providers")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	ctx := context.Background()

	neo, err := graph.NewDriver(cfg.Neo4jURI, cfg.Neo4jUser, cfg.Neo4jPassword)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to neo4j")
	}
	defer neo.Close(ctx)

	if err := neo.EnsureSchema(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to ensure neo4j schema")
	}

	tmdbClient := tmdb.NewClient(cfg.TMDBAPIKey)
	enrichClient := enrichment.NewClient(cfg.AxonURL, cfg.AxonAPIKey)
	movieRepo := graph.NewMovieRepository(neo)

	log.Info().Str("mode", *mode).Str("country", *country).Msg("starting ingestion")

	switch *mode {
	case "full":
		runFullIngestion(ctx, tmdbClient, enrichClient, movieRepo, *country)
	case "delta":
		runDeltaIngestion(ctx, tmdbClient, enrichClient, movieRepo, *country)
	default:
		log.Fatal().Str("mode", *mode).Msg("unknown mode; use full or delta")
	}

	log.Info().Msg("ingestion complete")
}

func runFullIngestion(ctx context.Context, tmdbClient *tmdb.Client, enrichClient *enrichment.Client, repo *graph.MovieRepository, country string) {
	log.Info().Msg("fetching bulk movie ID export from TMDB")

	ids, err := tmdbClient.GetBulkMovieIDs(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to fetch bulk movie IDs")
	}

	log.Info().Int("total", len(ids)).Msg("bulk ID export fetched")

	processed := 0
	errors := 0
	batch := make([]models.Movie, 0, 100)

	for _, id := range ids {
		select {
		case <-ctx.Done():
			log.Warn().Msg("context cancelled, stopping ingestion")
			return
		default:
		}

		movie, err := enrichMovie(ctx, tmdbClient, id, country)
		if err != nil {
			log.Error().Err(err).Int("tmdb_id", id).Msg("failed to enrich movie")
			errors++
			continue
		}

		if err := repo.UpsertMovie(ctx, movie); err != nil {
			log.Error().Err(err).Int64("tmdb_id", movie.TMDBID).Msg("failed to upsert movie")
			errors++
			continue
		}

		batch = append(batch, *movie)
		processed++

		if processed%100 == 0 {
			log.Info().Int("processed", processed).Int("errors", errors).Int("total", len(ids)).Msg("ingestion progress")

			// Enrich batch with AI themes and moods
			if enrichErr := enrichClient.ProcessMovieBatch(ctx, batch, repo); enrichErr != nil {
				log.Error().Err(enrichErr).Msg("batch enrichment failed")
			}
			batch = batch[:0]
		}

		// Respect TMDB rate limit: ~40 req/s
		time.Sleep(25 * time.Millisecond)
	}

	// Process remaining batch
	if len(batch) > 0 {
		if enrichErr := enrichClient.ProcessMovieBatch(ctx, batch, repo); enrichErr != nil {
			log.Error().Err(enrichErr).Msg("final batch enrichment failed")
		}
	}

	log.Info().Int("processed", processed).Int("errors", errors).Msg("full ingestion complete")
}

func runDeltaIngestion(ctx context.Context, tmdbClient *tmdb.Client, enrichClient *enrichment.Client, repo *graph.MovieRepository, country string) {
	log.Info().Msg("fetching movies changed in last 24h")

	// Fetch up to 5 pages of trending (used as delta proxy since TMDB changes API requires v4)
	allIDs := make([]int, 0, 500)
	for page := 1; page <= 5; page++ {
		ids, err := tmdbClient.GetTrendingMovies(ctx, page)
		if err != nil {
			log.Error().Err(err).Int("page", page).Msg("failed to fetch trending page")
			break
		}
		allIDs = append(allIDs, ids...)
	}

	log.Info().Int("titles", len(allIDs)).Msg("delta IDs collected")

	processed := 0
	errors := 0
	batch := make([]models.Movie, 0, 100)

	for _, id := range allIDs {
		select {
		case <-ctx.Done():
			log.Warn().Msg("context cancelled")
			return
		default:
		}

		// Update streaming providers for known movies
		providers, err := tmdbClient.GetWatchProviders(ctx, id, "movie")
		if err != nil {
			log.Error().Err(err).Int("tmdb_id", id).Msg("failed to get providers")
			errors++
			continue
		}

		countryProviders := providers[country]
		modelProviders := make([]models.Provider, 0, len(countryProviders))
		for _, p := range countryProviders {
			modelProviders = append(modelProviders, models.Provider{
				ProviderID:      int64(p.ProviderID),
				ProviderName:    p.ProviderName,
				LogoPath:        p.LogoPath,
				DisplayPriority: p.DisplayPriority,
				Type:            p.Type,
				Country:         country,
			})
		}

		if err := repo.UpsertProvider(ctx, id, modelProviders, country); err != nil {
			log.Error().Err(err).Int("tmdb_id", id).Msg("failed to upsert providers")
			errors++
			continue
		}

		processed++

		if processed%100 == 0 {
			log.Info().Int("processed", processed).Int("errors", errors).Msg("delta ingestion progress")

			if enrichErr := enrichClient.ProcessMovieBatch(ctx, batch, repo); enrichErr != nil {
				log.Error().Err(enrichErr).Msg("batch enrichment failed")
			}
			batch = batch[:0]
		}

		time.Sleep(25 * time.Millisecond)
	}

	log.Info().Int("processed", processed).Int("errors", errors).Msg("delta ingestion complete")
}

// enrichMovie fetches full TMDB details and computes the Cinova score.
func enrichMovie(ctx context.Context, client *tmdb.Client, id int, country string) (*models.Movie, error) {
	details, err := client.GetMovieDetails(ctx, id)
	if err != nil {
		return nil, err
	}

	movie := details.ToModel()
	movie.CinovaScore = scoring.ComputeCinovaScore(movie.VoteAverage, int(movie.VoteCount), movie.Popularity/1000)

	return movie, nil
}
