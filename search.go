package karu

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

func (k *Karu) Search(ctx context.Context, query, path string, permString string, opts SearchOptions) ([]*Post, error) {
	perms, err := ParsePermissions(permString)
	if err != nil {
		return nil, err
	}

	searchPath := strings.TrimSpace(path)
	if searchPath != "" {
		if err := validatePath(searchPath); err != nil {
			return nil, err
		}
		if err := k.requirePerm(perms, searchPath, 'r'); err != nil {
			return nil, nil
		}
	}

	opts.clamp()

	likePattern := "%" + k.escapeLike(query) + "%"

	var rows *sql.Rows
	baseWhere := `(title LIKE $1 ESCAPE '\' OR content LIKE $1 ESCAPE '\')`

	args := []interface{}{likePattern}
	argIdx := 2

	if searchPath != "" {
		baseWhere += fmt.Sprintf(` AND path = $%d`, argIdx)
		args = append(args, searchPath)
		argIdx++
	}

	if len(opts.ExcludedPaths) > 0 {
		var excluded []string
		for _, ep := range opts.ExcludedPaths {
			excluded = append(excluded, fmt.Sprintf(`path != $%d AND path NOT LIKE $%d ESCAPE '\'`, argIdx, argIdx+1))
			args = append(args, ep, ep+"/%")
			argIdx += 2
		}
		baseWhere += " AND " + strings.Join(excluded, " AND ")
	}

	args = append(args, opts.Limit, opts.Offset)

	querySQL := fmt.Sprintf(`SELECT id, path, parent_id, author_id, title, content,
		created_at, updated_at, is_locked, is_sticky, metadata
		FROM posts
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, baseWhere, argIdx, argIdx+1)

	rows, err = k.db.QueryContext(ctx, querySQL, args...)
	if err != nil {
		return nil, fmt.Errorf("searching posts: %w", err)
	}
	defer rows.Close()

	var posts []*Post
	for rows.Next() {
		p, err := scanPost(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning search result: %w", err)
		}
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return posts, nil
}
