package karu

import "time"

type Post struct {
	ID        string         `json:"id"`
	Path      string         `json:"path"`
	ParentID  *string        `json:"parent_id,omitempty"`
	AuthorID  string         `json:"author_id"`
	Title     string         `json:"title"`
	Content   string         `json:"content"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	IsLocked  bool           `json:"is_locked"`
	IsSticky  bool           `json:"is_sticky"`
	Children  []*Post        `json:"children,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type ListOptions struct {
	Offset          int
	Limit           int
	ExcludedPaths   []string
	IncludeSubpaths bool
}

type SearchOptions struct {
	Offset        int
	Limit         int
	ExcludedPaths []string
}

const (
	DefaultPageSize  = 25
	MaxPageSize      = 100
	MaxTitleLength   = 500
	MaxContentLength = 100000
	MaxPathLength    = 255
	MaxAuthorLength  = 255
)

func (o *ListOptions) clamp() {
	if o.Limit <= 0 || o.Limit > MaxPageSize {
		o.Limit = DefaultPageSize
	}
	if o.Offset < 0 {
		o.Offset = 0
	}
}

func (o *SearchOptions) clamp() {
	if o.Limit <= 0 || o.Limit > MaxPageSize {
		o.Limit = DefaultPageSize
	}
	if o.Offset < 0 {
		o.Offset = 0
	}
}
