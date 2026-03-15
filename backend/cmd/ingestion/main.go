package main

import (
	"context"
	"flag"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"

	"github.com/foundrylab-app/cinova/backend/internal/config"
	"github.com/foundrylab-app/cinova/backend/internal/enrichment"
	"github.com/foundrylab-app/cinova/backend/internal/graph"
	"github.com/foundrylab-app/cinova/backend/internal/models"
	"github.com/foundrylab-app/cinova/backend/internal/scoring"
	"github.com/foundrylab-app/cinova/backend/internal/tmdb"
	"github.com/foundrylab-app/cinova/backend/internal/wikidata"
)

const (
	workers       = 3   // concurrent TMDB fetch workers
	tmdbRatePerSec = 4  // 4 req/s = 40 req/10s (TMDB free tier limit)
	enrichBatch   = 30  // movies per Axon call
	logEvery      = 1000 // progress log interval
)

func main() {
	mode           := flag.String("mode", "delta", "ingestion mode: full or delta")
	mediaType      := flag.String("media-type", "all", "media type: movie, tvshow, or all")
	country        := flag.String("country", "US", "ISO 3166-1 alpha-2 country code for streaming providers")
	minVotes       := flag.Int("min-votes", 100, "minimum vote_count to include (quality filter)")
	minPopularity  := flag.Float64("min-popularity", 5.0, "minimum TMDB popularity to include from bulk export (0=all)")
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

	tmdbClient   := tmdb.NewClient(cfg.TMDBAPIKey)
	enrichClient := enrichment.NewClient(cfg.AxonURL, cfg.AxonAPIKey)
	movieRepo    := graph.NewMovieRepository(neo)
	wikiClient   := wikidata.NewClient()

	log.Info().Str("mode", *mode).Str("media_type", *mediaType).Str("country", *country).Int("min_votes", *minVotes).Float64("min_popularity", *minPopularity).Msg("starting ingestion")

	// Rate limiter shared across all workers
	limiter := rate.NewLimiter(rate.Limit(tmdbRatePerSec), tmdbRatePerSec)

	switch *mode {
	case "full":
		if *mediaType == "movie" || *mediaType == "all" {
			runFullIngestion(ctx, tmdbClient, enrichClient, movieRepo, wikiClient, *country, *minVotes, *minPopularity, limiter)
		}
		if *mediaType == "tvshow" || *mediaType == "all" {
			runFullTVIngestion(ctx, tmdbClient, wikiClient, enrichClient, movieRepo, *country, *minVotes, *minPopularity, limiter)
		}
	case "delta":
		if *mediaType == "movie" || *mediaType == "all" {
			runDeltaIngestion(ctx, tmdbClient, enrichClient, movieRepo, wikiClient, *country, limiter)
		}
		if *mediaType == "tvshow" || *mediaType == "all" {
			runDeltaTVIngestion(ctx, tmdbClient, wikiClient, enrichClient, movieRepo, *country, limiter)
		}
	case "enrich-only":
		if *mediaType == "movie" || *mediaType == "all" {
			runEnrichOnly(ctx, enrichClient, movieRepo)
		}
		if *mediaType == "tvshow" || *mediaType == "all" {
			runEnrichOnlyTV(ctx, enrichClient, movieRepo)
		}
	default:
		log.Fatal().Str("mode", *mode).Msg("unknown mode; use full, delta, or enrich-only")
	}

	log.Info().Msg("ingestion complete")
}

// ── Full Movie Ingestion ───────────────────────────────────────────────────────

func runFullIngestion(ctx context.Context, tmdbClient *tmdb.Client, enrichClient *enrichment.Client, repo *graph.MovieRepository, wikiClient *wikidata.Client, country string, minVotes int, minPopularity float64, limiter *rate.Limiter) {
	log.Info().Float64("min_popularity", minPopularity).Msg("fetching bulk movie ID export from TMDB")

	ids, err := tmdbClient.GetBulkMovieIDs(ctx, minPopularity)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to fetch bulk movie IDs")
	}

	log.Info().Int("total_ids", len(ids)).Int("min_votes", minVotes).Msg("bulk ID export fetched")

	// Async enrichment: dedicated goroutine so TMDB fetching is never blocked.
	enrichCh := make(chan []models.Movie, 20)
	var enrichWg sync.WaitGroup
	enrichWg.Add(1)
	go func() {
		defer enrichWg.Done()
		for b := range enrichCh {
			if enrichErr := enrichClient.ProcessMovieBatch(ctx, b, repo); enrichErr != nil {
				log.Warn().Err(enrichErr).Msg("async batch enrichment failed")
			}
		}
	}()

	var (
		processed  atomic.Int64
		skipped    atomic.Int64
		errors     atomic.Int64
		mu         sync.Mutex
		batch      = make([]models.Movie, 0, enrichBatch)
		start      = time.Now()
	)

	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup

	for _, id := range ids {
		select {
		case <-ctx.Done():
			log.Warn().Msg("context cancelled, stopping")
			wg.Wait()
			close(enrichCh)
			enrichWg.Wait()
			return
		default:
		}

		sem <- struct{}{}
		wg.Add(1)

		go func(movieID int) {
			defer func() { <-sem; wg.Done() }()

			if err := limiter.Wait(ctx); err != nil {
				return
			}

			movie, err := fetchMovie(ctx, tmdbClient, wikiClient, movieID, country)
			if err != nil {
				log.Error().Err(err).Int("tmdb_id", movieID).Msg("fetch failed")
				errors.Add(1)
				return
			}

			if int(movie.VoteCount) < minVotes {
				skipped.Add(1)
				return
			}

			if err := repo.UpsertMovie(ctx, movie); err != nil {
				log.Error().Err(err).Int64("tmdb_id", movie.TMDBID).Msg("upsert failed")
				errors.Add(1)
				return
			}

			// Upsert Wikidata awards
			for _, award := range movie.Awards {
				if err := repo.UpsertMovieAward(ctx, movie.TMDBID, award); err != nil {
					log.Debug().Err(err).Int64("tmdb_id", movie.TMDBID).Str("award", award.AwardName).Msg("award upsert failed")
				}
			}

			n := processed.Add(1)
			mu.Lock()
			batch = append(batch, *movie)
			var sendBatch []models.Movie
			if len(batch) >= enrichBatch {
				sendBatch = batch
				batch = make([]models.Movie, 0, enrichBatch)
			}
			mu.Unlock()

			if sendBatch != nil {
				select {
				case enrichCh <- sendBatch:
				default:
					log.Warn().Int("batch_size", len(sendBatch)).Msg("enrichment queue full, skipping batch")
				}
			}

			if n%int64(logEvery) == 0 {
				log.Info().
					Int64("processed", n).
					Int64("skipped", skipped.Load()).
					Int64("errors", errors.Load()).
					Int("total", len(ids)).
					Str("elapsed", time.Since(start).Round(time.Second).String()).
					Msg("ingestion progress")
			}
		}(id)
	}

	wg.Wait()

	// Flush remaining batch
	mu.Lock()
	remaining := batch
	mu.Unlock()
	if len(remaining) > 0 {
		enrichCh <- remaining
	}
	close(enrichCh)
	log.Info().Msg("waiting for enrichment to drain...")
	enrichWg.Wait()

	log.Info().
		Int64("processed", processed.Load()).
		Int64("skipped", skipped.Load()).
		Int64("errors", errors.Load()).
		Str("elapsed", time.Since(start).Round(time.Second).String()).
		Msg("full movie ingestion complete")

	// Wire Wikidata director influences (post-ingestion, run once)
	runWikidataInfluences(ctx, repo, wikiClient)

	// Recompute prestige-informed CinovaScores
	log.Info().Msg("computing graph prestige scores")
	if err := repo.ComputeAndUpdatePageRank(ctx); err != nil {
		log.Warn().Err(err).Msg("PageRank computation failed (non-fatal)")
	} else {
		log.Info().Msg("graph prestige scores updated")
	}
}

// ── Full TV Ingestion ──────────────────────────────────────────────────────────

func runFullTVIngestion(ctx context.Context, tmdbClient *tmdb.Client, wikiClient *wikidata.Client, enrichClient *enrichment.Client, repo *graph.MovieRepository, country string, minVotes int, minPopularity float64, limiter *rate.Limiter) {
	log.Info().Float64("min_popularity", minPopularity).Msg("fetching bulk TV show ID export from TMDB")

	ids, err := tmdbClient.GetBulkTVShowIDs(ctx, minPopularity)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to fetch bulk TV IDs")
	}

	log.Info().Int("total_ids", len(ids)).Int("min_votes", minVotes).Msg("TV bulk ID export fetched")

	// Async enrichment: dedicated goroutine so TMDB fetching is never blocked.
	enrichCh := make(chan []models.TVShow, 20)
	var enrichWg sync.WaitGroup
	enrichWg.Add(1)
	go func() {
		defer enrichWg.Done()
		for b := range enrichCh {
			if enrichErr := enrichClient.ProcessTVBatch(ctx, b, repo); enrichErr != nil {
				log.Warn().Err(enrichErr).Msg("async TV batch enrichment failed")
			}
		}
	}()

	var (
		processed atomic.Int64
		skipped   atomic.Int64
		errors    atomic.Int64
		mu        sync.Mutex
		batch     = make([]models.TVShow, 0, enrichBatch)
		start     = time.Now()
	)

	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup

	for _, id := range ids {
		select {
		case <-ctx.Done():
			wg.Wait()
			close(enrichCh)
			enrichWg.Wait()
			return
		default:
		}

		sem <- struct{}{}
		wg.Add(1)

		go func(showID int) {
			defer func() { <-sem; wg.Done() }()

			if err := limiter.Wait(ctx); err != nil {
				return
			}

			show, err := fetchTVShow(ctx, tmdbClient, wikiClient, showID, country)
			if err != nil {
				log.Error().Err(err).Int("tmdb_id", showID).Msg("TV fetch failed")
				errors.Add(1)
				return
			}

			if int(show.VoteCount) < minVotes {
				skipped.Add(1)
				return
			}

			if err := repo.UpsertTVShow(ctx, show); err != nil {
				log.Error().Err(err).Int64("tmdb_id", show.TMDBID).Msg("TV upsert failed")
				errors.Add(1)
				return
			}

			n := processed.Add(1)
			mu.Lock()
			batch = append(batch, *show)
			var sendBatch []models.TVShow
			if len(batch) >= enrichBatch {
				sendBatch = batch
				batch = make([]models.TVShow, 0, enrichBatch)
			}
			mu.Unlock()

			if sendBatch != nil {
				select {
				case enrichCh <- sendBatch:
				default:
					log.Warn().Int("batch_size", len(sendBatch)).Msg("TV enrichment queue full, skipping batch")
				}
			}

			if n%int64(logEvery) == 0 {
				log.Info().
					Int64("processed", n).
					Int64("skipped", skipped.Load()).
					Int64("errors", errors.Load()).
					Int("total", len(ids)).
					Str("elapsed", time.Since(start).Round(time.Second).String()).
					Msg("TV ingestion progress")
			}
		}(id)
	}

	wg.Wait()

	mu.Lock()
	remaining := batch
	mu.Unlock()
	if len(remaining) > 0 {
		enrichCh <- remaining
	}
	close(enrichCh)
	log.Info().Msg("waiting for TV enrichment to drain...")
	enrichWg.Wait()

	log.Info().
		Int64("processed", processed.Load()).
		Int64("skipped", skipped.Load()).
		Int64("errors", errors.Load()).
		Str("elapsed", time.Since(start).Round(time.Second).String()).
		Msg("full TV ingestion complete")
}

// ── Delta Movie Ingestion ──────────────────────────────────────────────────────

func runDeltaIngestion(ctx context.Context, tmdbClient *tmdb.Client, enrichClient *enrichment.Client, repo *graph.MovieRepository, wikiClient *wikidata.Client, country string, limiter *rate.Limiter) {
	log.Info().Msg("fetching trending movies (delta)")

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

	var (
		processed atomic.Int64
		errors    atomic.Int64
		mu        sync.Mutex
		batch     = make([]models.Movie, 0, enrichBatch)
		start     = time.Now()
	)

	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup

	for _, id := range allIDs {
		select {
		case <-ctx.Done():
			wg.Wait()
			return
		default:
		}

		sem <- struct{}{}
		wg.Add(1)

		go func(movieID int) {
			defer func() { <-sem; wg.Done() }()

			if err := limiter.Wait(ctx); err != nil {
				return
			}

			movie, err := fetchMovie(ctx, tmdbClient, wikiClient, movieID, country)
			if err != nil {
				log.Error().Err(err).Int("tmdb_id", movieID).Msg("fetch failed")
				errors.Add(1)
				return
			}

			if err := repo.UpsertMovie(ctx, movie); err != nil {
				log.Error().Err(err).Int64("tmdb_id", movie.TMDBID).Msg("upsert failed")
				errors.Add(1)
				return
			}

			for _, award := range movie.Awards {
				if err := repo.UpsertMovieAward(ctx, movie.TMDBID, award); err != nil {
					log.Debug().Err(err).Int64("tmdb_id", movie.TMDBID).Str("award", award.AwardName).Msg("award upsert failed")
				}
			}

			processed.Add(1)
			mu.Lock()
			batch = append(batch, *movie)
			currentBatch := batch
			if len(batch) >= enrichBatch {
				batch = make([]models.Movie, 0, enrichBatch)
			} else {
				currentBatch = nil
			}
			mu.Unlock()

			if currentBatch != nil {
				if enrichErr := enrichClient.ProcessMovieBatch(ctx, currentBatch, repo); enrichErr != nil {
					log.Warn().Err(enrichErr).Msg("batch enrichment failed")
				}
			}
		}(id)
	}

	wg.Wait()

	mu.Lock()
	remaining := batch
	mu.Unlock()
	if len(remaining) > 0 {
		enrichClient.ProcessMovieBatch(ctx, remaining, repo)
	}

	log.Info().
		Int64("processed", processed.Load()).
		Int64("errors", errors.Load()).
		Str("elapsed", time.Since(start).Round(time.Second).String()).
		Msg("delta movie ingestion complete")
}

// ── Delta TV Ingestion ─────────────────────────────────────────────────────────

func runDeltaTVIngestion(ctx context.Context, tmdbClient *tmdb.Client, wikiClient *wikidata.Client, enrichClient *enrichment.Client, repo *graph.MovieRepository, country string, limiter *rate.Limiter) {
	log.Info().Msg("fetching trending TV shows (delta)")

	allIDs := make([]int, 0, 500)
	for page := 1; page <= 5; page++ {
		ids, err := tmdbClient.GetTrendingTVShows(ctx, page)
		if err != nil {
			log.Error().Err(err).Int("page", page).Msg("failed to fetch trending TV page")
			break
		}
		allIDs = append(allIDs, ids...)
	}

	log.Info().Int("titles", len(allIDs)).Msg("delta TV IDs collected")

	var (
		processed atomic.Int64
		errors    atomic.Int64
		mu        sync.Mutex
		batch     = make([]models.TVShow, 0, enrichBatch)
		start     = time.Now()
	)

	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup

	for _, id := range allIDs {
		select {
		case <-ctx.Done():
			wg.Wait()
			return
		default:
		}

		sem <- struct{}{}
		wg.Add(1)

		go func(showID int) {
			defer func() { <-sem; wg.Done() }()

			if err := limiter.Wait(ctx); err != nil {
				return
			}

			show, err := fetchTVShow(ctx, tmdbClient, wikiClient, showID, country)
			if err != nil {
				log.Error().Err(err).Int("tmdb_id", showID).Msg("TV fetch failed")
				errors.Add(1)
				return
			}

			if err := repo.UpsertTVShow(ctx, show); err != nil {
				log.Error().Err(err).Int64("tmdb_id", show.TMDBID).Msg("TV upsert failed")
				errors.Add(1)
				return
			}

			processed.Add(1)
			mu.Lock()
			batch = append(batch, *show)
			currentBatch := batch
			if len(batch) >= enrichBatch {
				batch = make([]models.TVShow, 0, enrichBatch)
			} else {
				currentBatch = nil
			}
			mu.Unlock()

			if currentBatch != nil {
				enrichClient.ProcessTVBatch(ctx, currentBatch, repo)
			}
		}(id)
	}

	wg.Wait()

	mu.Lock()
	remaining := batch
	mu.Unlock()
	if len(remaining) > 0 {
		enrichClient.ProcessTVBatch(ctx, remaining, repo)
	}

	log.Info().
		Int64("processed", processed.Load()).
		Int64("errors", errors.Load()).
		Str("elapsed", time.Since(start).Round(time.Second).String()).
		Msg("delta TV ingestion complete")
}

// ── Helpers ────────────────────────────────────────────────────────────────────

// fetchMovie fetches full TMDB movie details + providers + Wikidata enrichment,
// then computes a CinovaScore incorporating critic scores and awards when available.
func fetchMovie(ctx context.Context, tmdbClient *tmdb.Client, wikiClient *wikidata.Client, id int, country string) (*models.Movie, error) {
	details, err := tmdbClient.GetMovieDetails(ctx, id)
	if err != nil {
		return nil, err
	}

	movie := details.ToModel()

	// Wikidata enrichment: awards, critic scores, Wikipedia plot
	params := scoring.ScoreParams{
		VoteAverage:   movie.VoteAverage,
		VoteCount:     int(movie.VoteCount),
		CriticScore:   -1,  // unknown until Wikidata confirms
		AwardScore:    0.5, // neutral until Wikidata confirms
		GraphPrestige: 0,   // set later by ComputeAndUpdatePageRank
		Budget:        movie.Budget,
		Revenue:       movie.Revenue,
	}

	if movie.WikidataID != "" {
		enrichCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		enrichData, enrichErr := wikiClient.GetMovieEnrichment(enrichCtx, movie.WikidataID)
		cancel()

		if enrichErr == nil && enrichData != nil {
			movie.Awards = enrichData.Awards
			params.AwardScore = scoring.ComputeAwardScore(enrichData.Awards)

			// Blend available critic signals
			var criticSignals []float64
			if enrichData.RTScore >= 0 {
				criticSignals = append(criticSignals, enrichData.RTScore)
			}
			if enrichData.MetaScore >= 0 {
				criticSignals = append(criticSignals, enrichData.MetaScore)
			}
			if enrichData.IMDbScore >= 0 {
				criticSignals = append(criticSignals, enrichData.IMDbScore)
			}
			if len(criticSignals) > 0 {
				sum := 0.0
				for _, s := range criticSignals {
					sum += s
				}
				params.CriticScore = sum / float64(len(criticSignals))
			}
		}

		// Wikipedia plot (best-effort)
		plotCtx, plotCancel := context.WithTimeout(ctx, 20*time.Second)
		plot, plotErr := wikiClient.GetWikipediaPlot(plotCtx, movie.WikidataID)
		plotCancel()
		if plotErr == nil && plot != "" {
			movie.PlotSummary = plot
		}
	}

	movie.CinovaScore = scoring.ComputeFullScore(params, scoring.DefaultWeights())

	// Extract watch providers from already-fetched details — no extra API call needed.
	movie.Providers = details.ProvidersForCountry(country)

	return movie, nil
}

// fetchTVShow fetches full TMDB TV show details + providers + Wikidata enrichment,
// then computes CinovaScore.
func fetchTVShow(ctx context.Context, tmdbClient *tmdb.Client, wikiClient *wikidata.Client, id int, country string) (*models.TVShow, error) {
	details, err := tmdbClient.GetTVShowDetails(ctx, id)
	if err != nil {
		return nil, err
	}

	show := details.ToModel()

	params := scoring.ScoreParams{
		VoteAverage:   show.VoteAverage,
		VoteCount:     int(show.VoteCount),
		CriticScore:   -1,
		AwardScore:    0.5,
		GraphPrestige: 0,
	}

	if show.WikidataID != "" {
		enrichCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		enrichData, enrichErr := wikiClient.GetMovieEnrichment(enrichCtx, show.WikidataID)
		cancel()

		if enrichErr == nil && enrichData != nil {
			var criticSignals []float64
			if enrichData.RTScore >= 0 {
				criticSignals = append(criticSignals, enrichData.RTScore)
			}
			if enrichData.MetaScore >= 0 {
				criticSignals = append(criticSignals, enrichData.MetaScore)
			}
			if enrichData.IMDbScore >= 0 {
				criticSignals = append(criticSignals, enrichData.IMDbScore)
			}
			if len(criticSignals) > 0 {
				sum := 0.0
				for _, s := range criticSignals {
					sum += s
				}
				params.CriticScore = sum / float64(len(criticSignals))
			}
		}

		plotCtx, plotCancel := context.WithTimeout(ctx, 20*time.Second)
		plot, plotErr := wikiClient.GetWikipediaPlot(plotCtx, show.WikidataID)
		plotCancel()
		if plotErr == nil && plot != "" {
			show.PlotSummary = plot
		}
	}

	show.CinovaScore = scoring.ComputeFullScore(params, scoring.DefaultWeights())

	// Extract watch providers from already-fetched details — no extra API call.
	show.Providers = details.ProvidersForCountry(country)

	return show, nil
}

// runWikidataInfluences fetches all director influence pairs from Wikidata and
// upserts INFLUENCED_BY relationships in Neo4j. Called once after full ingestion.
func runWikidataInfluences(ctx context.Context, repo *graph.MovieRepository, wikiClient *wikidata.Client) {
	log.Info().Msg("fetching director influence graph from Wikidata")

	influences, err := wikiClient.GetDirectorInfluencesWithLabels(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("Wikidata influence fetch failed (non-fatal)")
		return
	}

	log.Info().Int("pairs", len(influences)).Msg("Wikidata influences fetched")

	upserted := 0
	for _, inf := range influences {
		if err := repo.UpsertInfluenceRelationship(ctx, inf.DirectorName, inf.InfluencerName); err != nil {
			log.Debug().Err(err).Str("director", inf.DirectorName).Msg("upsert influence failed")
			continue
		}
		upserted++
	}

	log.Info().Int("upserted", upserted).Msg("Wikidata influence graph wired")
}

// ── Enrich-Only Mode ───────────────────────────────────────────────────────────

const enrichOnlyBatch = 30 // movies per Neo4j query

// runEnrichOnly iterates all Movie nodes without cinova_synopsis and enriches them.
// Designed to run after full ingestion completes — enriches in popularity order
// so the most-watched films get synopses first.
func runEnrichOnly(ctx context.Context, enrichClient *enrichment.Client, repo *graph.MovieRepository) {
	log.Info().Msg("starting enrich-only pass for movies without synopsis")

	var (
		total      int
		consecutive int // consecutive failures (potential infinite-loop guard)
		start      = time.Now()
	)

	for {
		movies, err := repo.GetMoviesWithoutSynopsis(ctx, enrichOnlyBatch)
		if err != nil {
			log.Error().Err(err).Msg("GetMoviesWithoutSynopsis failed")
			return
		}
		if len(movies) == 0 {
			break
		}

		if err := enrichClient.ProcessMovieBatch(ctx, movies, repo); err != nil {
			consecutive++
			log.Warn().Err(err).Int("consecutive_failures", consecutive).Msg("enrich-only movie batch failed")
			if consecutive >= 10 {
				log.Error().Msg("10 consecutive enrichment failures — aborting enrich-only pass")
				return
			}
		} else {
			consecutive = 0
			total += len(movies)
			log.Info().Int("enriched_so_far", total).Str("elapsed", time.Since(start).Round(time.Second).String()).Msg("enrich-only progress")
		}

		select {
		case <-ctx.Done():
			return
		default:
		}
	}

	log.Info().Int("total", total).Str("elapsed", time.Since(start).Round(time.Second).String()).Msg("movie enrich-only complete")
}

// runEnrichOnlyTV is the TV show counterpart of runEnrichOnly.
func runEnrichOnlyTV(ctx context.Context, enrichClient *enrichment.Client, repo *graph.MovieRepository) {
	log.Info().Msg("starting enrich-only pass for TV shows without synopsis")

	var (
		total      int
		consecutive int
		start      = time.Now()
	)

	for {
		shows, err := repo.GetTVShowsWithoutSynopsis(ctx, enrichOnlyBatch)
		if err != nil {
			log.Error().Err(err).Msg("GetTVShowsWithoutSynopsis failed")
			return
		}
		if len(shows) == 0 {
			break
		}

		if err := enrichClient.ProcessTVBatch(ctx, shows, repo); err != nil {
			consecutive++
			log.Warn().Err(err).Int("consecutive_failures", consecutive).Msg("enrich-only TV batch failed")
			if consecutive >= 10 {
				log.Error().Msg("10 consecutive TV enrichment failures — aborting")
				return
			}
		} else {
			consecutive = 0
			total += len(shows)
			log.Info().Int("enriched_so_far", total).Str("elapsed", time.Since(start).Round(time.Second).String()).Msg("TV enrich-only progress")
		}

		select {
		case <-ctx.Done():
			return
		default:
		}
	}

	log.Info().Int("total", total).Str("elapsed", time.Since(start).Round(time.Second).String()).Msg("TV enrich-only complete")
}
