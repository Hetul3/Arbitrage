package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/hetulpatel/Arbitrage/internal/collectors"
	"github.com/hetulpatel/Arbitrage/internal/hashutil"
)

const (
	defaultPath = "data/arb.db"
)

// Store wraps a SQLite DB connection.
type Store struct {
	path string
	db   *sql.DB
}

// Open creates (if needed) and opens the SQLite database.
func Open(path string) (*Store, error) {
	if path == "" {
		path = defaultPath
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("ensure data dir: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := ensureWAL(db); err != nil {
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}
	return &Store{path: path, db: db}, nil
}

func ensureWAL(db *sql.DB) error {
	const (
		maxAttempts = 5
		delay       = 200 * time.Millisecond
	)
	for i := 0; i < maxAttempts; i++ {
		if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
			if strings.Contains(err.Error(), "database is locked") {
				time.Sleep(delay)
				continue
			}
			return err
		}
		return nil
	}
	return fmt.Errorf("database is locked after retries")
}

// Path returns the path backing the store.
func (s *Store) Path() string {
	return s.path
}

// Close closes the DB.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// CreateTables ensures the unified markets table exists.
func (s *Store) CreateTables(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, unifiedSchemaSQL)
	return err
}

// DropTables removes the unified table.
func (s *Store) DropTables(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DROP TABLE IF EXISTS markets;`)
	return err
}

// ClearTables truncates the unified table.
func (s *Store) ClearTables(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM markets;`)
	return err
}

// MigrateToUnifiedSchema drops old per-venue tables (if any) and creates the unified schema.
func (s *Store) MigrateToUnifiedSchema(ctx context.Context) error {
	stmts := []string{
		`DROP TABLE IF EXISTS markets;`,
		`DROP TABLE IF EXISTS polymarket_markets;`,
		`DROP TABLE IF EXISTS kalshi_markets;`,
		unifiedSchemaSQL,
	}
	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

const unifiedSchemaSQL = `
CREATE TABLE IF NOT EXISTS markets (
	venue TEXT NOT NULL,
	market_id TEXT NOT NULL,
	event_id TEXT,
	event_title TEXT,
	event_description TEXT,
	event_category TEXT,
	event_status TEXT,
	resolution_source TEXT,
	resolution_details TEXT,
	settlement_sources_json TEXT,
	contract_terms_url TEXT,
	question TEXT,
	subtitle TEXT,
	reference_url TEXT,
	close_time TEXT,
	tick_size REAL,
	yes_bid REAL,
	yes_ask REAL,
	no_bid REAL,
	no_ask REAL,
	volume REAL,
	volume_24h REAL,
	open_interest REAL,
	clob_token_yes TEXT,
	clob_token_no TEXT,
	yes_bids_json TEXT,
	yes_asks_json TEXT,
	no_bids_json TEXT,
	no_asks_json TEXT,
	book_captured_at TEXT,
	book_hash TEXT,
	text_hash TEXT,
	resolution_hash TEXT,
	last_seen_at TEXT,
	raw_json TEXT,
	PRIMARY KEY (venue, market_id)
);
CREATE INDEX IF NOT EXISTS markets_event_idx ON markets(venue, event_id);
`

// UpsertPolymarketEvents inserts/updates markets for each fetched event.
func (s *Store) UpsertPolymarketEvents(ctx context.Context, events []collectors.Event) error {
	if len(events) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, unifiedUpsertSQL)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, ev := range events {
		for _, m := range ev.Markets {
			if err := s.execUpsert(ctx, stmt, collectors.VenuePolymarket, ev, m, now); err != nil {
				tx.Rollback()
				return err
			}
		}
	}
	return tx.Commit()
}

// UpsertKalshiEvents inserts/updates Kalshi markets.
func (s *Store) UpsertKalshiEvents(ctx context.Context, events []collectors.Event) error {
	if len(events) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, unifiedUpsertSQL)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, ev := range events {
		for _, m := range ev.Markets {
			if err := s.execUpsert(ctx, stmt, collectors.VenueKalshi, ev, m, now); err != nil {
				tx.Rollback()
				return err
			}
		}
	}
	return tx.Commit()
}

const unifiedUpsertSQL = `
INSERT INTO markets (
	venue, market_id, event_id, event_title, event_description, event_category, event_status,
	resolution_source, resolution_details, settlement_sources_json, contract_terms_url,
	question, subtitle, reference_url, close_time, tick_size, yes_bid, yes_ask, no_bid, no_ask,
	volume, volume_24h, open_interest, clob_token_yes, clob_token_no,
	yes_bids_json, yes_asks_json, no_bids_json, no_asks_json, book_captured_at, book_hash,
	text_hash, resolution_hash, last_seen_at, raw_json
) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(venue, market_id) DO UPDATE SET
	event_id=excluded.event_id,
	event_title=excluded.event_title,
	event_description=excluded.event_description,
	event_category=excluded.event_category,
	event_status=excluded.event_status,
	resolution_source=excluded.resolution_source,
	resolution_details=excluded.resolution_details,
	settlement_sources_json=excluded.settlement_sources_json,
	contract_terms_url=excluded.contract_terms_url,
	question=excluded.question,
	subtitle=excluded.subtitle,
	reference_url=excluded.reference_url,
	close_time=excluded.close_time,
	tick_size=excluded.tick_size,
	yes_bid=excluded.yes_bid,
	yes_ask=excluded.yes_ask,
	no_bid=excluded.no_bid,
	no_ask=excluded.no_ask,
	volume=excluded.volume,
	volume_24h=excluded.volume_24h,
	open_interest=excluded.open_interest,
	clob_token_yes=excluded.clob_token_yes,
	clob_token_no=excluded.clob_token_no,
	yes_bids_json=excluded.yes_bids_json,
	yes_asks_json=excluded.yes_asks_json,
	no_bids_json=excluded.no_bids_json,
	no_asks_json=excluded.no_asks_json,
	book_captured_at=excluded.book_captured_at,
	book_hash=excluded.book_hash,
	text_hash=excluded.text_hash,
	resolution_hash=excluded.resolution_hash,
	last_seen_at=excluded.last_seen_at,
	raw_json=excluded.raw_json;
`

func (s *Store) execUpsert(ctx context.Context, stmt *sql.Stmt, venue collectors.Venue, ev collectors.Event, m collectors.Market, ts string) error {
	raw := map[string]any{
		"event":  ev,
		"market": m,
	}
	rawJSON, _ := json.Marshal(raw)
	settlementJSON, _ := json.Marshal(ev.SettlementSources)
	textHash := hashutil.HashStrings(ev.Title, ev.Description, m.Question, m.Subtitle)
	resHash := hashutil.HashStrings(ev.ResolutionSource, ev.ResolutionDetails, ev.ContractTermsURL)

	var clobYes, clobNo string
	if len(m.ClobTokenIDs) > 0 {
		clobYes = m.ClobTokenIDs[0]
	}
	if len(m.ClobTokenIDs) > 1 {
		clobNo = m.ClobTokenIDs[1]
	}

	yesBook, noBook := splitOrderbooks(m)
	yesBidsJSON, yesAsksJSON := serializeOrderbookJSON(yesBook)
	noBidsJSON, noAsksJSON := serializeOrderbookJSON(noBook)
	bookCapturedAt := ts
	bookHash := hashutil.HashStrings(yesBidsJSON, yesAsksJSON, noBidsJSON, noAsksJSON)

	_, err := stmt.ExecContext(
		ctx,
		string(venue),
		m.MarketID,
		ev.EventID,
		ev.Title,
		ev.Description,
		ev.Category,
		ev.Status,
		ev.ResolutionSource,
		ev.ResolutionDetails,
		string(settlementJSON),
		ev.ContractTermsURL,
		m.Question,
		m.Subtitle,
		m.ReferenceURL,
		formatTime(m.CloseTime),
		m.TickSize,
		m.Price.YesBid,
		m.Price.YesAsk,
		m.Price.NoBid,
		m.Price.NoAsk,
		m.Volume,
		m.Volume24h,
		m.OpenInterest,
		clobYes,
		clobNo,
		yesBidsJSON,
		yesAsksJSON,
		noBidsJSON,
		noAsksJSON,
		bookCapturedAt,
		bookHash,
		textHash,
		resHash,
		ts,
		string(rawJSON),
	)
	return err
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func splitOrderbooks(m collectors.Market) (*collectors.Orderbook, *collectors.Orderbook) {
	if len(m.Orderbooks) == 0 {
		return nil, nil
	}

	var yes, no *collectors.Orderbook

	if len(m.ClobTokenIDs) > 0 {
		if ob, ok := m.Orderbooks[m.ClobTokenIDs[0]]; ok {
			copy := ob
			yes = &copy
		}
	}
	if len(m.ClobTokenIDs) > 1 {
		if ob, ok := m.Orderbooks[m.ClobTokenIDs[1]]; ok {
			copy := ob
			no = &copy
		}
	}

	if yes == nil {
		if ob, ok := m.Orderbooks["yes"]; ok {
			copy := ob
			yes = &copy
		}
	}
	if no == nil {
		if ob, ok := m.Orderbooks["no"]; ok {
			copy := ob
			no = &copy
		}
	}

	return yes, no
}

func serializeOrderbookJSON(ob *collectors.Orderbook) (string, string) {
	if ob == nil {
		return "", ""
	}
	return serializeLevels(ob.Bids), serializeLevels(ob.Asks)
}

func serializeLevels(levels []collectors.OrderbookLevel) string {
	if len(levels) == 0 {
		return ""
	}
	b, err := json.Marshal(levels)
	if err != nil {
		return ""
	}
	return string(b)
}
