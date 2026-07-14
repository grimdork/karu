package karu

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestUpdateOwnPost(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	post, err := k.CreateThread(ctx, "general", "user1", "Original", "Original", "general:rwd")
	if err != nil {
		t.Fatal(err)
	}

	err = k.UpdatePost(ctx, post.ID, "user1", "Updated", "Updated content", "general:rwd")
	if err != nil {
		t.Fatal(err)
	}

	thread, err := k.GetThread(ctx, post.ID, "general:r")
	if err != nil {
		t.Fatal(err)
	}
	if thread.Title != "Updated" {
		t.Fatalf("title = %q, want %q", thread.Title, "Updated")
	}
	if thread.Content != "Updated content" {
		t.Fatalf("content = %q, want %q", thread.Content, "Updated content")
	}
}

func TestUpdateNonExistentPost(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	err := k.UpdatePost(ctx, "no-such-id", "user1", "Title", "Content", "general:rwd")
	if !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("expected ErrPostNotFound, got %v", err)
	}
}

func TestNonAuthorCannotUpdateWithoutD(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	post, _ := k.CreateThread(ctx, "general", "user1", "T", "B", "general:rwd")

	err := k.UpdatePost(ctx, post.ID, "user2", "New", "Content", "general:rw")
	if err == nil {
		t.Fatal("expected error for non-author updating without 'D'")
	}
	var pe *PermissionError
	if !errors.As(err, &pe) {
		t.Fatalf("expected PermissionError, got %T", err)
	}
}

func TestModeratorCanUpdateAnyPostWithD(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	post, _ := k.CreateThread(ctx, "general", "user1", "T", "B", "general:rwd")

	err := k.UpdatePost(ctx, post.ID, "mod1", "Updated", "By mod", "general:D")
	if err != nil {
		t.Fatal(err)
	}

	updated, _ := k.GetThread(ctx, post.ID, "general:r")
	if updated.Title != "Updated" {
		t.Fatalf("expected 'Updated', got %q", updated.Title)
	}
}

func TestDeleteOwnPost(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	post, err := k.CreateThread(ctx, "general", "user1", "T", "B", "general:rwd")
	if err != nil {
		t.Fatal(err)
	}

	err = k.DeletePost(ctx, post.ID, "user1", "general:rwd")
	if err != nil {
		t.Fatal(err)
	}

	_, err = k.GetThread(ctx, post.ID, "general:r")
	if err != ErrPostNotFound {
		t.Fatalf("expected ErrPostNotFound after delete, got %v", err)
	}
}

func TestDeleteNonExistentPost(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	err := k.DeletePost(ctx, "no-such-id", "user1", "general:rwd")
	if !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("expected ErrPostNotFound, got %v", err)
	}
}

func TestDeletePostHasReplies(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	root, _ := k.CreateThread(ctx, "general", "user1", "T", "B", "general:rwd")
	k.CreatePost(ctx, root.ID, "user2", "Reply", "general:rwd")

	err := k.DeletePost(ctx, root.ID, "user1", "general:rwd")
	if err != ErrPostHasReplies {
		t.Fatalf("expected ErrPostHasReplies, got %v", err)
	}
}

func TestModeratorDeleteAnyPost(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	post, _ := k.CreateThread(ctx, "general", "user1", "T", "B", "general:rwd")

	err := k.DeletePost(ctx, post.ID, "mod1", "general:D")
	if err != nil {
		t.Fatal(err)
	}
}

func TestAuthorCannotDeleteWithoutDPermission(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	post, _ := k.CreateThread(ctx, "general", "user1", "T", "B", "general:rw")

	err := k.DeletePost(ctx, post.ID, "user1", "general:rw")
	if err == nil {
		t.Fatal("expected error: user has 'rw' but no 'd'")
	}
}

func TestDeleteThread(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	root, _ := k.CreateThread(ctx, "general", "user1", "T", "B", "general:rwdDtlm")
	k.CreatePost(ctx, root.ID, "user2", "R1", "general:rwd")
	r2, _ := k.CreatePost(ctx, root.ID, "user3", "R2", "general:rwd")
	k.CreatePost(ctx, r2.ID, "user2", "Nested", "general:rwd")

	err := k.DeleteThread(ctx, root.ID, "user1", "general:rwdDtlm")
	if err != nil {
		t.Fatal(err)
	}

	_, err = k.GetThread(ctx, root.ID, "general:r")
	if err != ErrPostNotFound {
		t.Fatalf("expected ErrPostNotFound after DeleteThread, got %v", err)
	}
}

func TestDeleteThreadNonRoot(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	root, _ := k.CreateThread(ctx, "general", "user1", "T", "B", "general:rwd")
	reply, _ := k.CreatePost(ctx, root.ID, "user2", "R", "general:rwd")

	err := k.DeleteThread(ctx, reply.ID, "user1", "general:rwdDtlm")
	if err == nil {
		t.Fatal("expected error when deleting non-root post with DeleteThread")
	}
}

func TestDeleteThreadNonExistent(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	err := k.DeleteThread(ctx, "no-such-id", "user1", "general:rwdDtlm")
	if !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("expected ErrPostNotFound, got %v", err)
	}
}

func TestDeleteThreadWithCascade(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	root, _ := k.CreateThread(ctx, "general", "user1", "T", "B", "general:rwdDtlm")
	r1, _ := k.CreatePost(ctx, root.ID, "user2", "R1", "general:rwd")
	k.CreatePost(ctx, r1.ID, "user3", "Nested", "general:rwd")

	err := k.DeleteThread(ctx, root.ID, "user1", "general:rwdDtlm")
	if err != nil {
		t.Fatal(err)
	}

	_, err = k.GetThread(ctx, root.ID, "general:r")
	if !errors.Is(err, ErrPostNotFound) {
		t.Fatal("expected thread root to be deleted")
	}
	_, err = k.getPostByID(ctx, r1.ID)
	if err == nil {
		t.Fatal("expected reply to be deleted")
	}
}

func TestUpdatedAtChangesOnEdit(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	post, _ := k.CreateThread(ctx, "general", "u1", "T", "B", "general:rwd")
	origUpdated := post.UpdatedAt

	time.Sleep(2 * time.Millisecond)

	k.UpdatePost(ctx, post.ID, "u1", "New", "Content", "general:rwd")

	fetched, _ := k.getPostByID(ctx, post.ID)
	if !fetched.UpdatedAt.After(origUpdated) {
		t.Fatal("updated_at should advance after edit")
	}
}
