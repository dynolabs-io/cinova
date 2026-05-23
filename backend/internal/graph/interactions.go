package graph

import (
	"context"
	"fmt"
	"time"

	"github.com/dynolabs-io/cinova/backend/internal/models"
)

// RateTitle creates or updates a RATED relationship from an owner (User or Session)
// to a Movie or TVShow node. ownerType should be "User" or "Session".
func (r *MovieRepository) RateTitle(ctx context.Context, ownerID string, ownerType string, tmdbID int, score float64) error {
	cypher := fmt.Sprintf(`
		MERGE (owner:%s {id: $owner_id})
		WITH owner
		MATCH (n) WHERE (n:Movie OR n:TVShow) AND n.tmdb_id = $tmdb_id
		MERGE (owner)-[rel:RATED]->(n)
		SET rel.score      = $score,
		    rel.updated_at = $updated_at
	`, ownerType)

	return r.driver.RunWriteUnit(ctx, cypher, map[string]interface{}{
		"owner_id":   ownerID,
		"tmdb_id":    tmdbID,
		"score":      score,
		"updated_at": time.Now().UTC().Format(time.RFC3339),
	})
}

// SaveTitle creates a SAVED relationship from an owner to a title (watchlist).
func (r *MovieRepository) SaveTitle(ctx context.Context, ownerID string, ownerType string, tmdbID int) error {
	cypher := fmt.Sprintf(`
		MERGE (owner:%s {id: $owner_id})
		WITH owner
		MATCH (n) WHERE (n:Movie OR n:TVShow) AND n.tmdb_id = $tmdb_id
		MERGE (owner)-[rel:SAVED]->(n)
		SET rel.saved_at = $saved_at
	`, ownerType)

	return r.driver.RunWriteUnit(ctx, cypher, map[string]interface{}{
		"owner_id": ownerID,
		"tmdb_id":  tmdbID,
		"saved_at": time.Now().UTC().Format(time.RFC3339),
	})
}

// DismissTitle creates a DISMISSED relationship from an owner to a title.
// Dismissed titles are excluded from future recommendations.
func (r *MovieRepository) DismissTitle(ctx context.Context, ownerID string, ownerType string, tmdbID int) error {
	cypher := fmt.Sprintf(`
		MERGE (owner:%s {id: $owner_id})
		WITH owner
		MATCH (n) WHERE (n:Movie OR n:TVShow) AND n.tmdb_id = $tmdb_id
		MERGE (owner)-[rel:DISMISSED]->(n)
		SET rel.dismissed_at = $dismissed_at
	`, ownerType)

	return r.driver.RunWriteUnit(ctx, cypher, map[string]interface{}{
		"owner_id":     ownerID,
		"tmdb_id":      tmdbID,
		"dismissed_at": time.Now().UTC().Format(time.RFC3339),
	})
}

// GetWatchlist returns all titles (Movies and TVShows) saved by an owner, ordered by save date descending.
func (r *MovieRepository) GetWatchlist(ctx context.Context, ownerID string, ownerType string) ([]models.Movie, error) {
	cypher := fmt.Sprintf(`
		MATCH (owner:%s {id: $owner_id})-[rel:SAVED]->(n)
		WHERE n:Movie OR n:TVShow
		OPTIONAL MATCH (n)-[:IN_GENRE]->(g:Genre)
		OPTIONAL MATCH (n)-[avail:AVAILABLE_ON]->(prov:Provider)
		RETURN n,
		       labels(n)                                                  AS node_labels,
		       rel.saved_at                                               AS saved_at,
		       collect(DISTINCT {id: g.id, name: g.name})                AS genres,
		       collect(DISTINCT {provider_id: prov.provider_id,
		                          provider_name: prov.provider_name,
		                          logo_path: prov.logo_path,
		                          type: avail.type})                      AS providers
		ORDER BY rel.saved_at DESC
	`, ownerType)

	records, err := r.driver.RunQuery(ctx, cypher, map[string]interface{}{
		"owner_id": ownerID,
	})
	if err != nil {
		return nil, fmt.Errorf("GetWatchlist query: %w", err)
	}

	movies := make([]models.Movie, 0, len(records))
	for _, rec := range records {
		node, _ := rec.Get("n")
		// Determine media type from labels to set correct title field
		mediaType := "movie"
		if labels, ok := rec.Get("node_labels"); ok {
			if labelList, ok := labels.([]interface{}); ok {
				for _, l := range labelList {
					if s, ok := l.(string); ok && s == "TVShow" {
						mediaType = "tv"
						break
					}
				}
			}
		}
		var m *models.Movie
		if mediaType == "tv" {
			// Convert TVShow node to Movie-shaped response for unified list
			show := tvNodeToModel(node)
			m = &models.Movie{
				TMDBID:         show.TMDBID,
				MediaType:      "tv",
				Title:          show.Name,
				OriginalTitle:  show.OriginalName,
				Tagline:        show.Tagline,
				Overview:       show.Overview,
				ReleaseDate:    show.FirstAirDate,
				VoteAverage:    show.VoteAverage,
				VoteCount:      show.VoteCount,
				Popularity:     show.Popularity,
				PosterPath:     show.PosterPath,
				BackdropPath:   show.BackdropPath,
				CinovaScore:    show.CinovaScore,
				CinovaSynopsis: show.CinovaSynopsis,
			}
		} else {
			m = movieNodeToModel(node)
		}
		if v, ok := rec.Get("genres"); ok {
			m.Genres = toGenres(v)
		}
		if v, ok := rec.Get("providers"); ok {
			m.Providers = toProviders(v)
		}
		movies = append(movies, *m)
	}
	return movies, nil
}

// MergeSessionToUser re-assigns all interaction edges from a Session node to a User node.
// This is called when an anonymous user completes signup or login.
// After merging, the Session node is deleted.
func (r *MovieRepository) MergeSessionToUser(ctx context.Context, sessionUUID string, userID string) error {
	// Ensure the User node exists
	if err := r.driver.RunWriteUnit(ctx, `
		MERGE (u:User {id: $user_id})
	`, map[string]interface{}{"user_id": userID}); err != nil {
		return fmt.Errorf("MergeSessionToUser ensure user: %w", err)
	}

	// Re-assign RATED edges
	if err := r.driver.RunWriteUnit(ctx, `
		MATCH (s:Session {id: $session_id})-[old:RATED]->(n)
		MERGE (u:User {id: $user_id})
		MERGE (u)-[newRel:RATED]->(n)
		SET newRel.score      = old.score,
		    newRel.updated_at = old.updated_at
		DELETE old
	`, map[string]interface{}{"session_id": sessionUUID, "user_id": userID}); err != nil {
		return fmt.Errorf("MergeSessionToUser rated: %w", err)
	}

	// Re-assign SAVED edges
	if err := r.driver.RunWriteUnit(ctx, `
		MATCH (s:Session {id: $session_id})-[old:SAVED]->(n)
		MERGE (u:User {id: $user_id})
		MERGE (u)-[newRel:SAVED]->(n)
		SET newRel.saved_at = old.saved_at
		DELETE old
	`, map[string]interface{}{"session_id": sessionUUID, "user_id": userID}); err != nil {
		return fmt.Errorf("MergeSessionToUser saved: %w", err)
	}

	// Re-assign DISMISSED edges
	if err := r.driver.RunWriteUnit(ctx, `
		MATCH (s:Session {id: $session_id})-[old:DISMISSED]->(n)
		MERGE (u:User {id: $user_id})
		MERGE (u)-[newRel:DISMISSED]->(n)
		SET newRel.dismissed_at = old.dismissed_at
		DELETE old
	`, map[string]interface{}{"session_id": sessionUUID, "user_id": userID}); err != nil {
		return fmt.Errorf("MergeSessionToUser dismissed: %w", err)
	}

	// Delete the now-empty Session node
	if err := r.driver.RunWriteUnit(ctx, `
		MATCH (s:Session {id: $session_id})
		DETACH DELETE s
	`, map[string]interface{}{"session_id": sessionUUID}); err != nil {
		return fmt.Errorf("MergeSessionToUser delete session: %w", err)
	}

	return nil
}
