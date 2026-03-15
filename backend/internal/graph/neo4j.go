package graph

import (
	"context"
	"fmt"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Driver wraps the Neo4j driver and provides connection lifecycle management.
type Driver struct {
	driver neo4j.DriverWithContext
}

// NewDriver connects to Neo4j using Bolt protocol and verifies connectivity.
func NewDriver(uri, user, password string) (*Driver, error) {
	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(user, password, ""))
	if err != nil {
		return nil, fmt.Errorf("neo4j new driver: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := driver.VerifyConnectivity(ctx); err != nil {
		return nil, fmt.Errorf("neo4j verify connectivity: %w", err)
	}

	return &Driver{driver: driver}, nil
}

// Close shuts down the Neo4j driver.
func (d *Driver) Close(ctx context.Context) error {
	return d.driver.Close(ctx)
}

// Ping checks Neo4j connectivity.
func (d *Driver) Ping(ctx context.Context) error {
	return d.driver.VerifyConnectivity(ctx)
}

// RunQuery executes a Cypher statement in a new auto-commit session and
// returns all records. It is suitable for read-only or single-write operations.
// For multi-statement transactions use RunTx.
func (d *Driver) RunQuery(ctx context.Context, cypher string, params map[string]interface{}) ([]*neo4j.Record, error) {
	session := d.driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode: neo4j.AccessModeRead,
	})
	defer session.Close(ctx)

	result, err := session.Run(ctx, cypher, params)
	if err != nil {
		return nil, fmt.Errorf("neo4j run query: %w", err)
	}
	records, err := result.Collect(ctx)
	if err != nil {
		return nil, fmt.Errorf("neo4j collect: %w", err)
	}
	return records, nil
}

// RunWrite executes a Cypher statement that modifies the graph (write transaction).
func (d *Driver) RunWrite(ctx context.Context, cypher string, params map[string]interface{}) ([]*neo4j.Record, error) {
	session := d.driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode: neo4j.AccessModeWrite,
	})
	defer session.Close(ctx)

	result, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		r, err := tx.Run(ctx, cypher, params)
		if err != nil {
			return nil, err
		}
		return r.Collect(ctx)
	})
	if err != nil {
		return nil, fmt.Errorf("neo4j write: %w", err)
	}
	records, ok := result.([]*neo4j.Record)
	if !ok {
		return nil, fmt.Errorf("neo4j write: unexpected result type")
	}
	return records, nil
}

// RunWriteUnit executes a Cypher write statement where the result is not needed.
func (d *Driver) RunWriteUnit(ctx context.Context, cypher string, params map[string]interface{}) error {
	_, err := d.RunWrite(ctx, cypher, params)
	return err
}

// EnsureSchema creates Neo4j constraints and indexes needed by Cinova.
// Safe to call on startup; uses CREATE CONSTRAINT IF NOT EXISTS.
func (d *Driver) EnsureSchema(ctx context.Context) error {
	statements := []string{
		`CREATE CONSTRAINT movie_tmdb_id  IF NOT EXISTS FOR (m:Movie)   REQUIRE m.tmdb_id IS UNIQUE`,
		`CREATE CONSTRAINT tv_tmdb_id     IF NOT EXISTS FOR (t:TVShow)  REQUIRE t.tmdb_id IS UNIQUE`,
		`CREATE CONSTRAINT person_tmdb    IF NOT EXISTS FOR (p:Person)  REQUIRE p.tmdb_id IS UNIQUE`,
		`CREATE CONSTRAINT genre_id       IF NOT EXISTS FOR (g:Genre)   REQUIRE g.id IS UNIQUE`,
		`CREATE CONSTRAINT keyword_id     IF NOT EXISTS FOR (k:Keyword) REQUIRE k.id IS UNIQUE`,
		`CREATE CONSTRAINT award_wikidata IF NOT EXISTS FOR (a:Award)   REQUIRE a.wikidata_id IS UNIQUE`,
		`CREATE CONSTRAINT theme_name     IF NOT EXISTS FOR (t:Theme)   REQUIRE t.name IS UNIQUE`,
		`CREATE CONSTRAINT mood_name      IF NOT EXISTS FOR (m:Mood)    REQUIRE m.name IS UNIQUE`,
		`CREATE CONSTRAINT user_id        IF NOT EXISTS FOR (u:User)    REQUIRE u.id IS UNIQUE`,
		`CREATE CONSTRAINT session_uuid   IF NOT EXISTS FOR (s:Session) REQUIRE s.uuid IS UNIQUE`,
		`CREATE INDEX movie_popularity   IF NOT EXISTS FOR (m:Movie)   ON (m.popularity)`,
		`CREATE INDEX tv_popularity      IF NOT EXISTS FOR (t:TVShow)  ON (t.popularity)`,
		`CREATE INDEX movie_score        IF NOT EXISTS FOR (m:Movie)   ON (m.cinova_score)`,
		`CREATE INDEX tv_score           IF NOT EXISTS FOR (t:TVShow)  ON (t.cinova_score)`,
	}

	session := d.driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode: neo4j.AccessModeWrite,
	})
	defer session.Close(ctx)

	for _, stmt := range statements {
		if _, err := session.Run(ctx, stmt, nil); err != nil {
			return fmt.Errorf("neo4j schema %q: %w", stmt, err)
		}
	}
	return nil
}

// ---- Helpers ----

// int64Val extracts an int64 from a Neo4j record field.
func int64Val(v interface{}) int64 {
	if v == nil {
		return 0
	}
	switch t := v.(type) {
	case int64:
		return t
	case float64:
		return int64(t)
	}
	return 0
}

// float64Val extracts a float64 from a Neo4j record field.
func float64Val(v interface{}) float64 {
	if v == nil {
		return 0
	}
	switch t := v.(type) {
	case float64:
		return t
	case int64:
		return float64(t)
	}
	return 0
}

// strVal extracts a string from a Neo4j record field.
func strVal(v interface{}) string {
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

// boolVal extracts a bool from a Neo4j record field.
func boolVal(v interface{}) bool {
	if v == nil {
		return false
	}
	b, _ := v.(bool)
	return b
}
