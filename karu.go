package karu

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

type Karu struct {
	db     *sql.DB
	driver string
}

func New(driver, connString string) (*Karu, error) {
	var db *sql.DB
	var err error

	switch driver {
	case "postgres":
		db, err = sql.Open("pgx", connString)
	case "sqlite":
		db, err = sql.Open("sqlite", connString)
	default:
		return nil, fmt.Errorf("unsupported driver: %s", driver)
	}

	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	k := &Karu{db: db, driver: driver}

	if err := k.db.PingContext(context.Background()); err != nil {
		db.Close()
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	return k, nil
}

func (k *Karu) Close() error {
	return k.db.Close()
}

func (k *Karu) DB() *sql.DB {
	return k.db
}

func (k *Karu) escapeLike(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}

type postScanner struct {
	ID        string
	Path      string
	ParentID  sql.NullString
	AuthorID  string
	Title     string
	Content   string
	CreatedAt int64
	UpdatedAt int64
	IsLocked  int64
	IsSticky  int64
	Metadata  string
}

func scanPost(scanner interface{ Scan(...interface{}) error }) (*Post, error) {
	var ps postScanner
	err := scanner.Scan(
		&ps.ID, &ps.Path, &ps.ParentID, &ps.AuthorID,
		&ps.Title, &ps.Content,
		&ps.CreatedAt, &ps.UpdatedAt,
		&ps.IsLocked, &ps.IsSticky, &ps.Metadata,
	)
	if err != nil {
		return nil, err
	}

	p := &Post{
		ID:        ps.ID,
		Path:      ps.Path,
		AuthorID:  ps.AuthorID,
		Title:     ps.Title,
		Content:   ps.Content,
		CreatedAt: time.UnixMilli(ps.CreatedAt),
		UpdatedAt: time.UnixMilli(ps.UpdatedAt),
		IsLocked:  ps.IsLocked != 0,
		IsSticky:  ps.IsSticky != 0,
		Metadata:  make(map[string]any),
	}

	if ps.ParentID.Valid {
		p.ParentID = &ps.ParentID.String
	}

	if ps.Metadata != "" {
		if err := json.Unmarshal([]byte(ps.Metadata), &p.Metadata); err != nil {
			return nil, fmt.Errorf("parsing metadata: %w", err)
		}
	}

	return p, nil
}

func (k *Karu) generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func validatePath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" || strings.HasPrefix(path, "/") || strings.HasSuffix(path, "/") {
		return fmt.Errorf("%w: must be non-empty with no leading or trailing slashes", ErrInvalidPath)
	}
	return nil
}

func (k *Karu) requirePerm(perms Permissions, path string, code byte) error {
	if !perms.Has(path, code) {
		return &PermissionError{Path: path, Code: code}
	}
	return nil
}

func (k *Karu) nowMilli() int64 {
	return time.Now().UnixMilli()
}
