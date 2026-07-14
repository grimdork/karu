package karu

import (
	"context"
	"errors"
	"testing"
)

func TestPermissionParse(t *testing.T) {
	tests := []struct {
		input string
		ok    bool
		check map[string]string
	}{
		{"general:rwd", true, map[string]string{"general": "rwd"}},
		{"general:rwd,moderators:Dtlm", true, map[string]string{"general": "rwd", "moderators": "Dtlm"}},
		{"", true, map[string]string{}},
		{"general:", false, nil},
		{":rwd", false, nil},
		{"general:rwd,", true, map[string]string{"general": "rwd"}},
	}
	for _, tt := range tests {
		p, err := ParsePermissions(tt.input)
		if tt.ok && err != nil {
			t.Errorf("ParsePermissions(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if !tt.ok && err == nil {
			t.Errorf("ParsePermissions(%q) expected error, got nil", tt.input)
			continue
		}
		if !tt.ok {
			continue
		}
		for k, v := range tt.check {
			if p[k] != v {
				t.Errorf("ParsePermissions(%q)[%q] = %q, want %q", tt.input, k, p[k], v)
			}
		}
	}
}

func TestPermissionHierarchy(t *testing.T) {
	p, err := ParsePermissions("general:rwd,general/off-topic:r")
	if err != nil {
		t.Fatal(err)
	}

	if !p.Has("general", 'r') {
		t.Error("expected 'r' on 'general'")
	}
	if !p.Has("general", 'w') {
		t.Error("expected 'w' on 'general'")
	}
	if !p.Has("general", 'd') {
		t.Error("expected 'd' on 'general'")
	}
	if !p.Has("general/off-topic", 'r') {
		t.Error("expected 'r' on 'general/off-topic'")
	}
	if p.Has("general/off-topic", 'w') {
		t.Error("expected no 'w' on 'general/off-topic' (strictest)")
	}
	if p.Has("general/off-topic", 'd') {
		t.Error("expected no 'd' on 'general/off-topic' (strictest)")
	}
	if p.Has("general/off-topic/deep", 'w') {
		t.Error("expected no 'w' on subpath (inherits strictest from parent)")
	}
	if !p.Has("general/off-topic/deep", 'r') {
		t.Error("expected 'r' on subpath (inherits from parent)")
	}
}

func TestPermissionNoMatch(t *testing.T) {
	p, err := ParsePermissions("moderators:rwd")
	if err != nil {
		t.Fatal(err)
	}
	if p.Has("general", 'r') {
		t.Error("expected no access on unmatched path")
	}
	if codes := p.Codes("general"); codes != "" {
		t.Errorf("expected empty codes, got %q", codes)
	}
}

func TestPermissionIntersection(t *testing.T) {
	p, err := ParsePermissions("a:rw,a/b:r,a/b/c:rw")
	if err != nil {
		t.Fatal(err)
	}
	codes := p.Codes("a/b/c")
	if codes != "r" {
		t.Errorf("expected 'r' (intersection of rw∩r∩rw), got %q", codes)
	}
}

func setupDB(t testing.TB) *Karu {
	t.Helper()
	k, err := New("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { k.Close() })
	if err := k.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	return k
}

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

func TestCreateReplyLockedThread(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	root, err := k.CreateThread(ctx, "general", "user1", "Thread", "Content", "general:rwDtlm")
	if err != nil {
		t.Fatal(err)
	}

	err = k.LockThread(ctx, root.ID, "user1", "general:rwDtlm")
	if err != nil {
		t.Fatal(err)
	}

	_, err = k.CreatePost(ctx, root.ID, "user2", "Reply", "general:rw")
	if err != ErrThreadLocked {
		t.Fatalf("expected ErrThreadLocked, got %v", err)
	}

	err = k.UnlockThread(ctx, root.ID, "user1", "general:rwDtlm")
	if err != nil {
		t.Fatal(err)
	}

	_, err = k.CreatePost(ctx, root.ID, "user2", "Reply after unlock", "general:rw")
	if err != nil {
		t.Fatal(err)
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

func TestLockUnlock(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	root, _ := k.CreateThread(ctx, "general", "user1", "T", "B", "general:rwd")

	err := k.LockThread(ctx, root.ID, "user1", "general:rwd")
	if err == nil {
		t.Fatal("expected permission error without 'l'")
	}

	err = k.LockThread(ctx, root.ID, "mod1", "general:rwdDtl")
	if err != nil {
		t.Fatal(err)
	}

	thread, _ := k.GetThread(ctx, root.ID, "general:r")
	if !thread.IsLocked {
		t.Fatal("expected thread to be locked")
	}

	err = k.UnlockThread(ctx, root.ID, "mod1", "general:rwdDtl")
	if err != nil {
		t.Fatal(err)
	}

	thread, _ = k.GetThread(ctx, root.ID, "general:r")
	if thread.IsLocked {
		t.Fatal("expected thread to be unlocked")
	}
}

func TestMoveThread(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	root, _ := k.CreateThread(ctx, "general", "user1", "T", "B", "general:rwd")
	reply, _ := k.CreatePost(ctx, root.ID, "user2", "R", "general:rwd")

	err := k.MoveThread(ctx, root.ID, "other", "user1", "general:m")
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

func TestMovePost(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	root1, _ := k.CreateThread(ctx, "general", "user1", "T1", "B1", "general:rwd")
	root2, _ := k.CreateThread(ctx, "general", "user1", "T2", "B2", "general:rwd")

	reply, _ := k.CreatePost(ctx, root1.ID, "user2", "Moving this", "general:rwd")

	err := k.MovePost(ctx, reply.ID, root2.ID, "mod1", "general:mw")
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

func TestPreferences(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	err := k.SetPreference(ctx, "user1", "theme", "dark")
	if err != nil {
		t.Fatal(err)
	}

	val, err := k.GetPreference(ctx, "user1", "theme")
	if err != nil {
		t.Fatal(err)
	}
	if val != "dark" {
		t.Fatalf("got %q, want %q", val, "dark")
	}

	k.SetPreference(ctx, "user1", "hidden_paths", "nsfw,spoilers")

	prefs, err := k.ListPreferences(ctx, "user1")
	if err != nil {
		t.Fatal(err)
	}
	if len(prefs) != 2 {
		t.Fatalf("expected 2 prefs, got %d", len(prefs))
	}

	err = k.DeletePreference(ctx, "user1", "theme")
	if err != nil {
		t.Fatal(err)
	}
	val, _ = k.GetPreference(ctx, "user1", "theme")
	if val != "" {
		t.Fatalf("expected empty after delete, got %q", val)
	}
}

func TestSearch(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	k.CreateThread(ctx, "general", "user1", "Golang Discussion", "I love Go", "general:rwd")
	k.CreateThread(ctx, "general", "user2", "Rust vs Go", "Comparing languages", "general:rwd")
	k.CreateThread(ctx, "other", "user1", "Off Topic", "Just chatting", "other:rwd")

	results, err := k.Search(ctx, "Go", "", "general:r", SearchOptions{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("expected search results")
	}

	results, err = k.Search(ctx, "Go", "other", "general:r", SearchOptions{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results from path 'other' without permission, got %d", len(results))
	}
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

func TestStickyThread(t *testing.T) {
	// sticky is stored but not exposed via CreateThread yet
	// test the field is persisted correctly via raw insert
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
