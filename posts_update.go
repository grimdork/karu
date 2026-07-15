package karu

import (
	"context"
	"database/sql"
	"fmt"
)

func (k *Karu) UpdatePost(ctx context.Context, postID, authorID, title, content, permString string) error {
	if err := validateLength("title", title, MaxTitleLength); err != nil {
		return err
	}
	if err := validateLength("content", content, MaxContentLength); err != nil {
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

	allowed := false
	needCode := byte('d')
	if post.AuthorID == authorID && perms.Has(post.Path, 'd') {
		allowed = true
	}
	if perms.Has(post.Path, 'D') {
		allowed = true
	}
	if !allowed {
		if post.AuthorID != authorID {
			needCode = 'D'
		}
		return &PermissionError{Path: post.Path, Code: needCode}
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
	needCode := byte('d')
	if post.AuthorID == authorID && perms.Has(post.Path, 'd') {
		allowed = true
	}
	if perms.Has(post.Path, 'D') {
		allowed = true
	}
	if !allowed {
		if post.AuthorID != authorID {
			needCode = 'D'
		}
		return &PermissionError{Path: post.Path, Code: needCode}
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
	needCode := byte('t')
	if post.AuthorID == authorID && perms.Has(post.Path, 't') {
		allowed = true
	}
	if perms.Has(post.Path, 'D') {
		allowed = true
	}
	if !allowed {
		if post.AuthorID != authorID {
			needCode = 'D'
		}
		return &PermissionError{Path: post.Path, Code: needCode}
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
