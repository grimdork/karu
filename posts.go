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

func (k *Karu) UpdatePost(ctx context.Context, postID, authorID, title, content, permString string) error {
	perms, err := ParsePermissions(permString)
	if err != nil {
		return err
	}

	post, err := k.getPostByID(ctx, postID)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrPostNotFound
		}
		return err
	}

	allowed := false
	if post.AuthorID == authorID && perms.Has(post.Path, 'd') {
		allowed = true
	}
	if perms.Has(post.Path, 'D') {
		allowed = true
	}
	if !allowed {
		return &PermissionError{Path: post.Path, Code: 'd'}
	}

	now := k.nowMilli()
	_, err = k.db.ExecContext(ctx,
		`UPDATE posts SET title = $1, content = $2, updated_at = $3 WHERE id = $4`,
		title, content, now, postID)
	if err != nil {
		return fmt.Errorf("updating post: %w", err)
	}
	return nil
}

func (k *Karu) DeletePost(ctx context.Context, postID, authorID, permString string) error {
	perms, err := ParsePermissions(permString)
	if err != nil {
		return err
	}

	post, err := k.getPostByID(ctx, postID)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrPostNotFound
		}
		return err
	}

	allowed := false
	if post.AuthorID == authorID && perms.Has(post.Path, 'd') {
		allowed = true
	}
	if perms.Has(post.Path, 'D') {
		allowed = true
	}
	if !allowed {
		return &PermissionError{Path: post.Path, Code: 'd'}
	}

	var childCount int
	err = k.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM posts WHERE parent_id = $1`, postID).Scan(&childCount)
	if err != nil {
		return fmt.Errorf("checking for replies: %w", err)
	}
	if childCount > 0 {
		return ErrPostHasReplies
	}

	_, err = k.db.ExecContext(ctx, `DELETE FROM posts WHERE id = $1`, postID)
	if err != nil {
		return fmt.Errorf("deleting post: %w", err)
	}
	return nil
}

func (k *Karu) DeleteThread(ctx context.Context, postID, authorID, permString string) error {
	perms, err := ParsePermissions(permString)
	if err != nil {
		return err
	}

	post, err := k.getPostByID(ctx, postID)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrPostNotFound
		}
		return err
	}

	if post.ParentID != nil {
		return fmt.Errorf("post is not a thread root")
	}

	allowed := false
	if post.AuthorID == authorID && perms.Has(post.Path, 't') {
		allowed = true
	}
	if perms.Has(post.Path, 'D') {
		allowed = true
	}
	if !allowed {
		return &PermissionError{Path: post.Path, Code: 't'}
	}

	_, err = k.db.ExecContext(ctx,
		`WITH RECURSIVE thread_ids AS (
			SELECT id FROM posts WHERE id = $1
			UNION ALL
			SELECT p.id FROM posts p JOIN thread_ids t ON p.parent_id = t.id
		)
		DELETE FROM posts WHERE id IN (SELECT id FROM thread_ids)`,
		postID)
	if err != nil {
		return fmt.Errorf("deleting thread: %w", err)
	}
	return nil
}

func (k *Karu) LockThread(ctx context.Context, postID, authorID, permString string) error {
	perms, err := ParsePermissions(permString)
	if err != nil {
		return err
	}

	post, err := k.getPostByID(ctx, postID)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrPostNotFound
		}
		return err
	}

	if !perms.Has(post.Path, 'l') {
		return &PermissionError{Path: post.Path, Code: 'l'}
	}

	_, err = k.db.ExecContext(ctx, `UPDATE posts SET is_locked = 1, updated_at = $1 WHERE id = $2`, k.nowMilli(), postID)
	if err != nil {
		return fmt.Errorf("locking thread: %w", err)
	}
	return nil
}

func (k *Karu) UnlockThread(ctx context.Context, postID, authorID, permString string) error {
	perms, err := ParsePermissions(permString)
	if err != nil {
		return err
	}

	post, err := k.getPostByID(ctx, postID)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrPostNotFound
		}
		return err
	}

	if !perms.Has(post.Path, 'l') {
		return &PermissionError{Path: post.Path, Code: 'l'}
	}

	_, err = k.db.ExecContext(ctx, `UPDATE posts SET is_locked = 0, updated_at = $1 WHERE id = $2`, k.nowMilli(), postID)
	if err != nil {
		return fmt.Errorf("unlocking thread: %w", err)
	}
	return nil
}

func (k *Karu) MoveThread(ctx context.Context, postID, newPath, authorID, permString string) error {
	if err := validatePath(newPath); err != nil {
		return err
	}

	perms, err := ParsePermissions(permString)
	if err != nil {
		return err
	}

	post, err := k.getPostByID(ctx, postID)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrPostNotFound
		}
		return err
	}

	if !perms.Has(post.Path, 'm') {
		return &PermissionError{Path: post.Path, Code: 'm'}
	}

	now := k.nowMilli()
	_, err = k.db.ExecContext(ctx,
		`WITH RECURSIVE thread_ids AS (
			SELECT id FROM posts WHERE id = $1
			UNION ALL
			SELECT p.id FROM posts p JOIN thread_ids t ON p.parent_id = t.id
		)
		UPDATE posts SET path = $2, updated_at = $3
		WHERE id IN (SELECT id FROM thread_ids)`,
		postID, newPath, now)
	if err != nil {
		return fmt.Errorf("moving thread: %w", err)
	}
	return nil
}

func (k *Karu) MovePost(ctx context.Context, postID, newParentID, authorID, permString string) error {
	perms, err := ParsePermissions(permString)
	if err != nil {
		return err
	}

	post, err := k.getPostByID(ctx, postID)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrPostNotFound
		}
		return err
	}

	if !perms.Has(post.Path, 'm') {
		return &PermissionError{Path: post.Path, Code: 'm'}
	}

	newParent, err := k.getPostByID(ctx, newParentID)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrInvalidParent
		}
		return err
	}

	// moving to another thread requires 'm' on source path
	// and 'w' on destination path (to be able to post there)
	if !perms.Has(newParent.Path, 'w') {
		return &PermissionError{Path: newParent.Path, Code: 'w'}
	}

	now := k.nowMilli()
	_, err = k.db.ExecContext(ctx,
		`UPDATE posts SET parent_id = $1, path = $2, updated_at = $3 WHERE id = $4`,
		newParentID, newParent.Path, now, postID)
	if err != nil {
		return fmt.Errorf("moving post: %w", err)
	}
	return nil
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
