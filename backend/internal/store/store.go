package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"github.com/bueti/status-aggregator/backend/internal/providers"
)

var ErrNotFound = errors.New("provider not found")

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(context.Background()); err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
	CREATE TABLE IF NOT EXISTS providers (
		id         TEXT PRIMARY KEY,
		name       TEXT NOT NULL,
		kind       TEXT NOT NULL,
		params     TEXT NOT NULL,
		sort_order INTEGER NOT NULL DEFAULT 0,
		created_at INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS providers_sort ON providers(sort_order, id);
	`)
	return err
}

func (s *Store) Count(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM providers`).Scan(&n)
	return n, err
}

func (s *Store) List(ctx context.Context) ([]providers.Config, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, kind, params, sort_order
		FROM providers
		ORDER BY sort_order, id
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []providers.Config
	for rows.Next() {
		var c providers.Config
		var kind string
		var params string
		if err := rows.Scan(&c.ID, &c.Name, &kind, &params, &c.SortOrder); err != nil {
			return nil, err
		}
		c.Kind = providers.Kind(kind)
		c.Params = json.RawMessage(params)
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) Get(ctx context.Context, id string) (providers.Config, error) {
	var c providers.Config
	var kind, params string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, kind, params, sort_order FROM providers WHERE id = ?
	`, id).Scan(&c.ID, &c.Name, &kind, &params, &c.SortOrder)
	if errors.Is(err, sql.ErrNoRows) {
		return c, ErrNotFound
	}
	if err != nil {
		return c, err
	}
	c.Kind = providers.Kind(kind)
	c.Params = json.RawMessage(params)
	return c, nil
}

func (s *Store) Create(ctx context.Context, c providers.Config) error {
	if len(c.Params) == 0 {
		c.Params = json.RawMessage(`{}`)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO providers (id, name, kind, params, sort_order, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, c.ID, c.Name, string(c.Kind), string(c.Params), c.SortOrder, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("insert provider: %w", err)
	}
	return nil
}

func (s *Store) Update(ctx context.Context, c providers.Config) error {
	if len(c.Params) == 0 {
		c.Params = json.RawMessage(`{}`)
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE providers SET name = ?, kind = ?, params = ?, sort_order = ?
		WHERE id = ?
	`, c.Name, string(c.Kind), string(c.Params), c.SortOrder, c.ID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM providers WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) SeedIfEmpty(ctx context.Context, defaults []providers.Config) error {
	n, err := s.Count(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	for i, c := range defaults {
		if c.SortOrder == 0 {
			c.SortOrder = i
		}
		if err := s.Create(ctx, c); err != nil {
			return err
		}
	}
	return nil
}
