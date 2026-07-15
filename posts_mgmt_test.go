package karu

import (
	"context"
	"errors"
	"testing"
)

func TestCreateReplyLockedThread(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	root, err := k.CreateThread(ctx, "general", "user1", "Thread", "Content", "general:rwDtlm")
	if err != nil {
		t.Fatal(err)
	}

	err = k.LockThread(ctx, root.ID, "general:rwDtlm")
	if err != nil {
		t.Fatal(err)
	}

	_, err = k.CreatePost(ctx, root.ID, "user2", "Reply", "general:rw")
	if err != ErrThreadLocked {
		t.Fatalf("expected ErrThreadLocked, got %v", err)
	}

	err = k.UnlockThread(ctx, root.ID, "general:rwDtlm")
	if err != nil {
		t.Fatal(err)
	}

	_, err = k.CreatePost(ctx, root.ID, "user2", "Reply after unlock", "general:rw")
	if err != nil {
		t.Fatal(err)
	}
}

func TestLockUnlock(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	root, _ := k.CreateThread(ctx, "general", "user1", "T", "B", "general:rwd")

	err := k.LockThread(ctx, root.ID, "general:rwd")
	if err == nil {
		t.Fatal("expected permission error without 'l'")
	}

	err = k.LockThread(ctx, root.ID, "general:rwdDtl")
	if err != nil {
		t.Fatal(err)
	}

	thread, _ := k.GetThread(ctx, root.ID, "general:r")
	if !thread.IsLocked {
		t.Fatal("expected thread to be locked")
	}

	err = k.UnlockThread(ctx, root.ID, "general:rwdDtl")
	if err != nil {
		t.Fatal(err)
	}

	thread, _ = k.GetThread(ctx, root.ID, "general:r")
	if thread.IsLocked {
		t.Fatal("expected thread to be unlocked")
	}
}

func TestLockThreadWithoutLPermission(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	root, _ := k.CreateThread(ctx, "general", "user1", "T", "B", "general:rw")

	err := k.LockThread(ctx, root.ID, "general:rw")
	if err == nil {
		t.Fatal("expected error for locking without 'l'")
	}
}

func TestUnlockThreadWithoutLPermission(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	root, _ := k.CreateThread(ctx, "general", "user1", "T", "B", "general:rwdDtl")

	k.LockThread(ctx, root.ID, "general:rwdDtl")

	err := k.UnlockThread(ctx, root.ID, "general:rwd")
	if err == nil {
		t.Fatal("expected error for unlocking without 'l' in perms")
	}
}

func TestLockNonExistentThread(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	err := k.LockThread(ctx, "no-such-id", "general:rwdDtl")
	if !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("expected ErrPostNotFound, got %v", err)
	}
}

func TestMoveThread(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	root, _ := k.CreateThread(ctx, "general", "user1", "T", "B", "general:rwd")
	reply, _ := k.CreatePost(ctx, root.ID, "user2", "R", "general:rwd")

	err := k.MoveThread(ctx, root.ID, "other", "general:m")
	if err != nil {
		t.Fatal(err)
	}

	thread, err := k.GetThread(ctx, root.ID, "other:r")
	if err != nil {
		t.Fatal(err)
	}
	if thread.Path != "other" {
		t.Fatalf("thread root path = %q, want %q", thread.Path, "other")
	}

	replyPost, _ := k.getPostByID(ctx, reply.ID)
	if replyPost.Path != "other" {
		t.Fatalf("reply path = %q, want %q", replyPost.Path, "other")
	}
}

func TestMoveThreadWithoutPermission(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	root, _ := k.CreateThread(ctx, "general", "user1", "T", "B", "general:rwd")
	err := k.MoveThread(ctx, root.ID, "other", "general:rw")
	if err == nil {
		t.Fatal("expected error for moving without 'm'")
	}
}

func TestMoveThreadNonRoot(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	root, _ := k.CreateThread(ctx, "general", "user1", "T", "B", "general:rwd")
	reply, _ := k.CreatePost(ctx, root.ID, "user2", "R", "general:rwd")

	err := k.MoveThread(ctx, reply.ID, "other", "general:m")
	if err != nil {
		t.Fatal(err)
	}

	moved, _ := k.getPostByID(ctx, reply.ID)
	if moved.Path != "other" {
		t.Fatalf("reply path = %q, want %q", moved.Path, "other")
	}
}

func TestMovePost(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	root1, _ := k.CreateThread(ctx, "general", "user1", "T1", "B1", "general:rwd")
	root2, _ := k.CreateThread(ctx, "general", "user1", "T2", "B2", "general:rwd")

	reply, _ := k.CreatePost(ctx, root1.ID, "user2", "Moving this", "general:rwd")

	err := k.MovePost(ctx, reply.ID, root2.ID, "general:mw")
	if err != nil {
		t.Fatal(err)
	}

	moved, _ := k.getPostByID(ctx, reply.ID)
	if *moved.ParentID != root2.ID {
		t.Fatalf("parent_id = %s, want %s", *moved.ParentID, root2.ID)
	}

	thread2, _ := k.GetThread(ctx, root2.ID, "general:r")
	found := false
	for _, c := range thread2.Children {
		if c.ID == reply.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("moved post should be in thread2's children")
	}
}

func TestMovePostWithoutPermission(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	root1, _ := k.CreateThread(ctx, "general", "user1", "T1", "B1", "general:rwd")
	root2, _ := k.CreateThread(ctx, "general", "user1", "T2", "B2", "general:rwd")
	reply, _ := k.CreatePost(ctx, root1.ID, "user2", "Moving", "general:rwd")

	err := k.MovePost(ctx, reply.ID, root2.ID, "general:rw")
	if err == nil {
		t.Fatal("expected error for moving without 'm'")
	}
}

func TestMovePostToNonExistentParent(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	root, _ := k.CreateThread(ctx, "general", "user1", "T", "B", "general:rwd")
	reply, _ := k.CreatePost(ctx, root.ID, "user2", "Moving", "general:rwd")

	err := k.MovePost(ctx, reply.ID, "no-such-id", "general:mw")
	if !errors.Is(err, ErrInvalidParent) && !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("expected ErrInvalidParent or ErrPostNotFound, got %v", err)
	}
}

func TestStickyThread(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	now := k.nowMilli()
	_, err := k.db.ExecContext(ctx,
		`INSERT INTO posts (id, path, author_id, title, content, created_at, updated_at, is_sticky, metadata)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, 1, '{}')`,
		"sticky-id", "general", "user1", "Sticky", "Body", now, now)
	if err != nil {
		t.Fatal(err)
	}

	post, _ := k.getPostByID(ctx, "sticky-id")
	if !post.IsSticky {
		t.Fatal("expected sticky post")
	}
}
