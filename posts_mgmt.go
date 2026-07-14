package karu

import (
	"context"
	"database/sql"
	"fmt"
)

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

	if post.ParentID != nil {
		return fmt.Errorf("post is not a thread root")
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

	if post.ParentID != nil {
		return fmt.Errorf("post is not a thread root")
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

func (k *Karu) StickyThread(ctx context.Context, postID, authorID, permString string) error {
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

	if !perms.Has(post.Path, 'l') {
		return &PermissionError{Path: post.Path, Code: 'l'}
	}

	_, err = k.db.ExecContext(ctx, `UPDATE posts SET is_sticky = 1, updated_at = $1 WHERE id = $2`, k.nowMilli(), postID)
	if err != nil {
		return fmt.Errorf("sticking thread: %w", err)
	}
	return nil
}

func (k *Karu) UnstickyThread(ctx context.Context, postID, authorID, permString string) error {
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

	if !perms.Has(post.Path, 'l') {
		return &PermissionError{Path: post.Path, Code: 'l'}
	}

	_, err = k.db.ExecContext(ctx, `UPDATE posts SET is_sticky = 0, updated_at = $1 WHERE id = $2`, k.nowMilli(), postID)
	if err != nil {
		return fmt.Errorf("unsticking thread: %w", err)
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
