package karu

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

func (k *Karu) CreateThread(ctx context.Context, path, authorID, title, content, permString string) (*Post, error) {
	if err := validatePath(path); err != nil {
		return nil, err
	}

	perms, err := ParsePermissions(permString)
	if err != nil {
		return nil, err
	}
	if err := k.requirePerm(perms, path, 'w'); err != nil {
		return nil, err
	}

	now := k.nowMilli()
	post := &Post{
		ID:        k.generateID(),
		Path:      path,
		AuthorID:  authorID,
		Title:     title,
		Content:   content,
		CreatedAt: time.UnixMilli(now),
		UpdatedAt: time.UnixMilli(now),
		Metadata:  make(map[string]any),
	}

	_, err = k.db.ExecContext(ctx,
		`INSERT INTO posts (id, path, author_id, title, content, created_at, updated_at, metadata)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, '{}')`,
		post.ID, post.Path, post.AuthorID, post.Title, post.Content, now, now)
	if err != nil {
		return nil, fmt.Errorf("creating thread: %w", err)
	}

	return post, nil
}

func (k *Karu) CreatePost(ctx context.Context, parentID, authorID, content, permString string) (*Post, error) {
	perms, err := ParsePermissions(permString)
	if err != nil {
		return nil, err
	}

	parent, err := k.getPostByID(ctx, parentID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrPostNotFound
		}
		return nil, err
	}

	if err := k.requirePerm(perms, parent.Path, 'w'); err != nil {
		return nil, err
	}

	if parent.IsLocked {
		return nil, ErrThreadLocked
	}

	now := k.nowMilli()
	post := &Post{
		ID:        k.generateID(),
		Path:      parent.Path,
		ParentID:  &parent.ID,
		AuthorID:  authorID,
		Content:   content,
		CreatedAt: time.UnixMilli(now),
		UpdatedAt: time.UnixMilli(now),
		Metadata:  make(map[string]any),
	}

	_, err = k.db.ExecContext(ctx,
		`INSERT INTO posts (id, path, parent_id, author_id, content, created_at, updated_at, metadata)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, '{}')`,
		post.ID, post.Path, post.ParentID, post.AuthorID,
		post.Content, now, now)
	if err != nil {
		return nil, fmt.Errorf("creating post: %w", err)
	}

	return post, nil
}

func (k *Karu) GetThread(ctx context.Context, postID, permString string) (*Post, error) {
	root, err := k.getPostByID(ctx, postID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrPostNotFound
		}
		return nil, err
	}

	perms, err := ParsePermissions(permString)
	if err != nil {
		return nil, err
	}
	if err := k.requirePerm(perms, root.Path, 'r'); err != nil {
		return nil, ErrPostNotFound
	}

	rows, err := k.db.QueryContext(ctx,
		`WITH RECURSIVE thread AS (
			SELECT id, path, parent_id, author_id, title, content,
			       created_at, updated_at, is_locked, is_sticky, metadata, 0 AS depth
			FROM posts WHERE id = $1
			UNION ALL
			SELECT p.id, p.path, p.parent_id, p.author_id, p.title, p.content,
			       p.created_at, p.updated_at, p.is_locked, p.is_sticky, p.metadata, t.depth + 1
			FROM posts p
			JOIN thread t ON p.parent_id = t.id
		)
		SELECT id, path, parent_id, author_id, title, content,
		       created_at, updated_at, is_locked, is_sticky, metadata
		FROM thread
		ORDER BY depth, created_at`,
		postID)
	if err != nil {
		return nil, fmt.Errorf("fetching thread: %w", err)
	}
	defer rows.Close()

	var posts []*Post
	for rows.Next() {
		p, err := scanPost(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning thread post: %w", err)
		}
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(posts) == 0 {
		return nil, ErrPostNotFound
	}

	return buildTree(posts), nil
}

func (k *Karu) ListThreads(ctx context.Context, path, permString string, opts ListOptions) ([]*Post, error) {
	if err := validatePath(path); err != nil {
		return nil, err
	}

	perms, err := ParsePermissions(permString)
	if err != nil {
		return nil, err
	}
	if err := k.requirePerm(perms, path, 'r'); err != nil {
		return nil, nil
	}

	opts.clamp()

	var rows *sql.Rows
	escaped := k.escapeLike(path)

	if opts.IncludeSubpaths {
		rows, err = k.db.QueryContext(ctx,
			`SELECT id, path, parent_id, author_id, title, content,
			        created_at, updated_at, is_locked, is_sticky, metadata
			 FROM posts
			 WHERE (path = $1 OR path LIKE $2 ESCAPE '\') AND parent_id IS NULL
			 ORDER BY is_sticky DESC, created_at DESC
			 LIMIT $3 OFFSET $4`,
			path, escaped+"/%", opts.Limit, opts.Offset)
	} else {
		rows, err = k.db.QueryContext(ctx,
			`SELECT id, path, parent_id, author_id, title, content,
			        created_at, updated_at, is_locked, is_sticky, metadata
			 FROM posts
			 WHERE path = $1 AND parent_id IS NULL
			 ORDER BY is_sticky DESC, created_at DESC
			 LIMIT $2 OFFSET $3`,
			path, opts.Limit, opts.Offset)
	}
	if err != nil {
		return nil, fmt.Errorf("listing threads: %w", err)
	}
	defer rows.Close()

	var posts []*Post
	for rows.Next() {
		p, err := scanPost(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning thread: %w", err)
		}
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return posts, nil
}

func (k *Karu) getPostByID(ctx context.Context, id string) (*Post, error) {
	row := k.db.QueryRowContext(ctx,
		`SELECT id, path, parent_id, author_id, title, content,
		        created_at, updated_at, is_locked, is_sticky, metadata
		 FROM posts WHERE id = $1`, id)
	return scanPost(row)
}

func buildTree(posts []*Post) *Post {
	if len(posts) == 0 {
		return nil
	}

	root := posts[0]
	postMap := make(map[string]*Post, len(posts))
	for _, p := range posts {
		postMap[p.ID] = p
	}
	for _, p := range posts[1:] {
		if p.ParentID != nil {
			if parent, ok := postMap[*p.ParentID]; ok {
				parent.Children = append(parent.Children, p)
			}
		}
	}
	return root
}
