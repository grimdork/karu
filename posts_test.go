package karu

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCreateThread(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	post, err := k.CreateThread(ctx, "general", "user1", "Hello", "World", "general:rw")
	if err != nil {
		t.Fatal(err)
	}
	if post.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if post.Path != "general" {
		t.Fatalf("path = %q, want %q", post.Path, "general")
	}
	if post.Title != "Hello" {
		t.Fatalf("title = %q, want %q", post.Title, "Hello")
	}
	if post.AuthorID != "user1" {
		t.Fatalf("author = %q, want %q", post.AuthorID, "user1")
	}
	if post.ParentID != nil {
		t.Fatal("thread root should have nil parent")
	}
	if post.IsLocked {
		t.Fatal("new thread should not be locked")
	}
}

func TestCreateThreadPermissionDenied(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	_, err := k.CreateThread(ctx, "general", "user1", "Hello", "World", "other:rw")
	if err == nil {
		t.Fatal("expected permission error")
	}
	var pe *PermissionError
	if !errors.As(err, &pe) {
		t.Fatalf("expected PermissionError, got %T", err)
	}
}

func TestCreateReply(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	root, err := k.CreateThread(ctx, "general", "user1", "Thread", "Content", "general:rw")
	if err != nil {
		t.Fatal(err)
	}

	reply, err := k.CreatePost(ctx, root.ID, "user2", "Reply content", "general:rw")
	if err != nil {
		t.Fatal(err)
	}
	if reply.ParentID == nil || *reply.ParentID != root.ID {
		t.Fatal("reply should have parent_id set")
	}
	if reply.Path != root.Path {
		t.Fatalf("reply path = %q, want %q", reply.Path, root.Path)
	}
}

func TestCreateReplyToNonExistentPost(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	_, err := k.CreatePost(ctx, "no-such-id", "user1", "Reply", "general:rw")
	if !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("expected ErrPostNotFound, got %v", err)
	}
}

func TestCreateReplyInheritsPath(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	root, _ := k.CreateThread(ctx, "general/off-topic", "u1", "T", "B", "general:rwd")

	reply, err := k.CreatePost(ctx, root.ID, "u2", "R", "general:rwd")
	if err != nil {
		t.Fatal(err)
	}
	if reply.Path != "general/off-topic" {
		t.Fatalf("reply path = %q, want 'general/off-topic'", reply.Path)
	}
}

func TestGetThread(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	root, err := k.CreateThread(ctx, "general", "user1", "Thread", "Root", "general:rwd")
	if err != nil {
		t.Fatal(err)
	}

	r1, _ := k.CreatePost(ctx, root.ID, "user2", "Reply 1", "general:rwd")
	_, _ = k.CreatePost(ctx, root.ID, "user3", "Reply 2", "general:rwd")
	r3, _ := k.CreatePost(ctx, r1.ID, "user2", "Nested reply", "general:rwd")

	thread, err := k.GetThread(ctx, root.ID, "general:r")
	if err != nil {
		t.Fatal(err)
	}
	if thread.ID != root.ID {
		t.Fatalf("got thread ID %q, want %q", thread.ID, root.ID)
	}
	if len(thread.Children) != 2 {
		t.Fatalf("expected 2 direct children, got %d", len(thread.Children))
	}

	found := false
	for _, c := range thread.Children {
		if c.ID == r1.ID {
			found = true
			if len(c.Children) != 1 {
				t.Fatalf("expected r1 to have 1 child, got %d", len(c.Children))
			}
			if c.Children[0].ID != r3.ID {
				t.Fatalf("r1's child should be r3, got %q", c.Children[0].ID)
			}
		}
	}
	if !found {
		t.Fatal("expected to find r1 in thread children")
	}
}

func TestGetThreadSilentDeny(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	root, err := k.CreateThread(ctx, "general", "user1", "Thread", "Root", "general:rw")
	if err != nil {
		t.Fatal(err)
	}

	_, err = k.GetThread(ctx, root.ID, "other:r")
	if err != ErrPostNotFound {
		t.Fatalf("expected ErrPostNotFound (silent deny), got %v", err)
	}
}

func TestGetNonExistentThread(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	_, err := k.GetThread(ctx, "no-such-id", "general:r")
	if !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("expected ErrPostNotFound, got %v", err)
	}
}

func TestGetThreadOtherPathAccess(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	root, _ := k.CreateThread(ctx, "general", "u1", "T", "B", "general:rwd")

	_, err := k.GetThread(ctx, root.ID, "a:r")
	if !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("expected ErrPostNotFound, got %v", err)
	}
}

func TestListThreads(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		_, err := k.CreateThread(ctx, "general", "user1", "Thread", "Body", "general:rwd")
		if err != nil {
			t.Fatal(err)
		}
	}

	threads, err := k.ListThreads(ctx, "general", "general:r", ListOptions{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(threads) != 5 {
		t.Fatalf("expected 5 threads, got %d", len(threads))
	}
}

func TestListThreadsSilentDeny(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	k.CreateThread(ctx, "general", "user1", "T", "B", "general:rw")

	threads, err := k.ListThreads(ctx, "general", "other:r", ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(threads) != 0 {
		t.Fatalf("expected 0 threads (silent deny), got %d", len(threads))
	}
}

func TestListThreadsPagination(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		_, err := k.CreateThread(ctx, "general", "user1", "T", "B", "general:rwd")
		if err != nil {
			t.Fatal(err)
		}
	}

	page1, err := k.ListThreads(ctx, "general", "general:r", ListOptions{Limit: 3, Offset: 0})
	if err != nil {
		t.Fatal(err)
	}
	if len(page1) != 3 {
		t.Fatalf("page1: expected 3, got %d", len(page1))
	}

	page2, err := k.ListThreads(ctx, "general", "general:r", ListOptions{Limit: 3, Offset: 3})
	if err != nil {
		t.Fatal(err)
	}
	if len(page2) != 3 {
		t.Fatalf("page2: expected 3, got %d", len(page2))
	}

	ids := make(map[string]bool)
	for _, p := range page1 {
		ids[p.ID] = true
	}
	for _, p := range page2 {
		if ids[p.ID] {
			t.Fatal("page overlap detected")
		}
	}
}

func TestListThreadsWithSubpaths(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	k.CreateThread(ctx, "general", "user1", "Root", "B", "general:rwd")
	k.CreateThread(ctx, "general/off-topic", "user2", "Sub", "B", "general:rwd")
	k.CreateThread(ctx, "other", "user3", "Other", "B", "other:rwd")

	threads, _ := k.ListThreads(ctx, "general", "general:r", ListOptions{Limit: 10})
	if len(threads) != 1 {
		t.Fatalf("without subpaths: expected 1, got %d", len(threads))
	}

	threads, _ = k.ListThreads(ctx, "general", "general:r", ListOptions{Limit: 10, IncludeSubpaths: true})
	if len(threads) != 2 {
		t.Fatalf("with subpaths: expected 2, got %d", len(threads))
	}
}

func TestListThreadsDefaults(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	threads, err := k.ListThreads(ctx, "general", "general:r", ListOptions{Offset: -1, Limit: 0})
	if err != nil {
		t.Fatal(err)
	}
	_ = threads
}

func TestListThreadsOrder(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	now := k.nowMilli()
	ids := make([]string, 3)
	for i := 0; i < 3; i++ {
		id := k.generateID()
		ids[i] = id
		_, err := k.db.ExecContext(ctx,
			`INSERT INTO posts (id, path, author_id, title, content, created_at, updated_at, metadata)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, '{}')`,
			id, "general", "u", "T", "B", now+int64(i+1)*1000, now+int64(i+1)*1000)
		if err != nil {
			t.Fatal(err)
		}
	}

	threads, err := k.ListThreads(ctx, "general", "general:r", ListOptions{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(threads) < 2 {
		t.Fatal("need at least 2 threads for ordering test")
	}

	if threads[0].CreatedAt.Before(threads[1].CreatedAt) {
		t.Fatal("threads should be ordered by created_at DESC")
	}
}

func TestMultiplePaths(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	paths := []struct {
		path  string
		perm  string
		title string
	}{
		{"blog/post-1", "blog:rwd", "Blog Post 1"},
		{"blog/post-2", "blog:rwd", "Blog Post 2"},
		{"gallery/photo-1", "gallery:rwd", "Photo 1"},
		{"forum/general", "forum:rwd", "General Chat"},
	}

	for _, p := range paths {
		_, err := k.CreateThread(ctx, p.path, "user1", p.title, "Body", p.perm)
		if err != nil {
			t.Fatalf("creating thread in %q: %v", p.path, err)
		}
	}

	blogThreads, _ := k.ListThreads(ctx, "blog/post-1", "other:r", ListOptions{})
	if len(blogThreads) != 0 {
		t.Fatal("expected 0 blog threads without blog permission")
	}

	blogThreads, _ = k.ListThreads(ctx, "blog/post-1", "blog:r", ListOptions{})
	if len(blogThreads) != 1 {
		t.Fatalf("expected 1 blog thread, got %d", len(blogThreads))
	}
}

func TestDeepNestedReplies(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	root, _ := k.CreateThread(ctx, "general", "user1", "Root", "B", "general:rwd")
	a, _ := k.CreatePost(ctx, root.ID, "u2", "A", "general:rwd")
	b, _ := k.CreatePost(ctx, a.ID, "u3", "B", "general:rwd")
	c, _ := k.CreatePost(ctx, b.ID, "u2", "C", "general:rwd")
	d, _ := k.CreatePost(ctx, c.ID, "u3", "D", "general:rwd")

	thread, err := k.GetThread(ctx, root.ID, "general:r")
	if err != nil {
		t.Fatal(err)
	}

	if len(thread.Children) != 1 || thread.Children[0].ID != a.ID {
		t.Fatal("root should have 1 direct child")
	}
	if len(thread.Children[0].Children) != 1 || thread.Children[0].Children[0].ID != b.ID {
		t.Fatal("a should have 1 child (b)")
	}
	if len(thread.Children[0].Children[0].Children) != 1 ||
		thread.Children[0].Children[0].Children[0].ID != c.ID {
		t.Fatal("b should have 1 child (c)")
	}
	if len(thread.Children[0].Children[0].Children[0].Children) != 1 ||
		thread.Children[0].Children[0].Children[0].Children[0].ID != d.ID {
		t.Fatal("c should have 1 child (d)")
	}

	_ = b
}

func TestInvalidPath(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	_, err := k.CreateThread(ctx, "/leading", "u", "T", "B", "leading:rw")
	if err == nil {
		t.Fatal("expected error for leading slash")
	}

	_, err = k.CreateThread(ctx, "trailing/", "u", "T", "B", "trailing:rw")
	if err == nil {
		t.Fatal("expected error for trailing slash")
	}

	_, err = k.CreateThread(ctx, "", "u", "T", "B", "empty:rw")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestCreatedAtPopulated(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	post, _ := k.CreateThread(ctx, "general", "u1", "T", "B", "general:rwd")

	if post.CreatedAt.IsZero() {
		t.Fatal("created_at should be populated")
	}
	now := time.Now()
	if post.CreatedAt.After(now.Add(time.Second)) {
		t.Fatal("created_at should not be in the future")
	}
	if now.Sub(post.CreatedAt) > time.Minute {
		t.Fatal("created_at should be recent (within 1 minute)")
	}
}

func TestEmptyStringPermissions(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	threads, err := k.ListThreads(ctx, "general", "", ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(threads) != 0 {
		t.Log("expected 0 threads with empty permissions")
	}
}

func TestEmptyPermissionsListThreads(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	k.CreateThread(ctx, "general", "u", "T", "B", "general:rw")

	threads, err := k.ListThreads(ctx, "general", "", ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(threads) != 0 {
		t.Fatalf("expected 0 threads with empty permissions, got %d", len(threads))
	}
}

func TestMetadataPersistence(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	post, _ := k.CreateThread(ctx, "general", "user1", "T", "B", "general:rwd")

	meta := `{"key":"value","number":42}`
	_, err := k.db.ExecContext(ctx,
		`UPDATE posts SET metadata = $1 WHERE id = $2`, meta, post.ID)
	if err != nil {
		t.Fatal(err)
	}

	fetched, _ := k.getPostByID(ctx, post.ID)
	if fetched.Metadata["key"] != "value" {
		t.Fatalf("metadata key = %v, want 'value'", fetched.Metadata["key"])
	}
	if n, ok := fetched.Metadata["number"].(float64); !ok || n != 42 {
		t.Fatalf("metadata number = %v, want 42", fetched.Metadata["number"])
	}
}

func TestConcurrency(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func() {
			_, err := k.CreateThread(ctx, "general", "user", "T", "B", "general:rw")
			if err != nil {
				t.Log(err)
			}
			done <- struct{}{}
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}

	threads, _ := k.ListThreads(ctx, "general", "general:r", ListOptions{Limit: 100})
	if len(threads) != 10 {
		t.Fatalf("expected 10 threads from concurrent creation, got %d", len(threads))
	}
}

func TestConcurrentReadWrite(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	root, _ := k.CreateThread(ctx, "general", "u1", "T", "B", "general:rwd")

	done := make(chan bool, 20)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := k.CreatePost(ctx, root.ID, "reader", "Reply", "general:rw")
			if err != nil && !errors.Is(err, ErrThreadLocked) {
				t.Log(err)
			}
			done <- true
		}()
	}
	for i := 0; i < 10; i++ {
		go func() {
			_, err := k.GetThread(ctx, root.ID, "general:r")
			if err != nil && !errors.Is(err, ErrPostNotFound) {
				t.Log(err)
			}
			done <- true
		}()
	}
	for i := 0; i < 20; i++ {
		<-done
	}
}

func TestClose(t *testing.T) {
	k := setupDB(t)
	err := k.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func TestDBMethod(t *testing.T) {
	k := setupDB(t)
	db := k.DB()
	if db == nil {
		t.Fatal("DB() returned nil")
	}
	db.Stats()
}
