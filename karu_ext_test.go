package karu

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPermissionDuplicateCodes(t *testing.T) {
	p, err := ParsePermissions("a:rwrw") // duplicate chars
	if err != nil {
		t.Fatal(err)
	}
	if !p.Has("a", 'r') {
		t.Error("expected 'r'")
	}
	if !p.Has("a", 'w') {
		t.Error("expected 'w'")
	}
}

func TestPermissionSingleLevelPath(t *testing.T) {
	p, err := ParsePermissions("a:r")
	if err != nil {
		t.Fatal(err)
	}
	if !p.Has("a", 'r') {
		t.Error("expected 'r' on single-level path 'a'")
	}
	if p.Has("b", 'r') {
		t.Error("expected no access on different path")
	}
}

func TestPermissionEmptyCodes(t *testing.T) {
	p := Permissions{"general": ""}
	if p.Has("general", 'r') {
		t.Error("empty codes should deny all")
	}
	if codes := p.Codes("general"); codes != "" {
		t.Errorf("expected empty codes, got %q", codes)
	}
}

func TestPermissionWhitespace(t *testing.T) {
	p, err := ParsePermissions("  general:rwd , moderators:D  ")
	if err != nil {
		t.Fatal(err)
	}
	if !p.Has("general", 'r') {
		t.Error("expected 'r' on 'general'")
	}
	if !p.Has("moderators", 'D') {
		t.Error("expected 'D' on 'moderators'")
	}
}

func TestPermissionSubpathOnly(t *testing.T) {
	p, err := ParsePermissions("a/b:r")
	if err != nil {
		t.Fatal(err)
	}
	// no permissions on parent
	if codes := p.Codes("a"); codes != "" {
		t.Errorf("expected empty on parent, got %q", codes)
	}
	if !p.Has("a/b", 'r') {
		t.Error("expected 'r' on exact subpath")
	}
	// subpath should cascade down (no intersection at parent, so only subpath applies)
	if !p.Has("a/b/c", 'r') {
		t.Error("expected 'r' on nested subpath via inheritance")
	}
}

func TestPermissionMultipleGroups(t *testing.T) {
	p, err := ParsePermissions("a:rwDtlm,b:rwd,a/b:rD")
	if err != nil {
		t.Fatal(err)
	}
	if !p.Has("a", 'r') {
		t.Error("expected 'r' on 'a'")
	}
	if !p.Has("b", 'r') {
		t.Error("expected 'r' on 'b'")
	}
	// intersection of "a:rwDtlm" and "a/b:rD" on "a/b" = {r}
	if !p.Has("a/b", 'r') {
		t.Error("expected 'r' on 'a/b' from intersection")
	}
	if !p.Has("a/b", 'D') {
		t.Error("expected 'D' on 'a/b' - in intersection of rwDtlm and rD")
	}
	if p.Has("a/b", 'w') {
		t.Error("expected no 'w' on 'a/b' - not in intersection")
	}
}

func TestPermissionLargeCodes(t *testing.T) {
	p, err := ParsePermissions("a:rwdDtlm")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range "rwdDtlm" {
		if !p.Has("a", byte(c)) {
			t.Errorf("expected code '%c'", c)
		}
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

func TestGetNonExistentThread(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	_, err := k.GetThread(ctx, "no-such-id", "general:r")
	if !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("expected ErrPostNotFound, got %v", err)
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

func TestDeleteNonExistentPost(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	err := k.DeletePost(ctx, "no-such-id", "user1", "general:rwd")
	if !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("expected ErrPostNotFound, got %v", err)
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

	// verify pages don't overlap
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

	// without subpaths
	threads, _ := k.ListThreads(ctx, "general", "general:r", ListOptions{Limit: 10})
	if len(threads) != 1 {
		t.Fatalf("without subpaths: expected 1, got %d", len(threads))
	}

	// with subpaths - user has general:r which gives read access to general/*
	threads, _ = k.ListThreads(ctx, "general", "general:r", ListOptions{Limit: 10, IncludeSubpaths: true})
	if len(threads) != 2 {
		t.Fatalf("with subpaths: expected 2, got %d", len(threads))
	}
}

func TestListThreadsDefaults(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	// test default limit clamping
	threads, err := k.ListThreads(ctx, "general", "general:r", ListOptions{Offset: -1, Limit: 0})
	if err != nil {
		t.Fatal(err)
	}
	_ = threads // should not panic, defaults applied
}

func TestNonAuthorCannotUpdateWithoutD(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	post, _ := k.CreateThread(ctx, "general", "user1", "T", "B", "general:rwd")

	// user2 tries to update user1's post without 'd' or 'D'
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

func TestAuthorCannotDeleteWithoutDPermission(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	post, _ := k.CreateThread(ctx, "general", "user1", "T", "B", "general:rw")

	err := k.DeletePost(ctx, post.ID, "user1", "general:rw")
	if err == nil {
		t.Fatal("expected error: user has 'rw' but no 'd'")
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

	// Verify all posts are gone
	_, err = k.GetThread(ctx, root.ID, "general:r")
	if !errors.Is(err, ErrPostNotFound) {
		t.Fatal("expected thread root to be deleted")
	}
	_, err = k.getPostByID(ctx, r1.ID)
	if err == nil {
		t.Fatal("expected reply to be deleted")
	}
}

func TestLockThreadWithoutLPermission(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	root, _ := k.CreateThread(ctx, "general", "user1", "T", "B", "general:rw")

	err := k.LockThread(ctx, root.ID, "user1", "general:rw")
	if err == nil {
		t.Fatal("expected error for locking without 'l'")
	}
}

func TestUnlockThreadWithoutLPermission(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	root, _ := k.CreateThread(ctx, "general", "user1", "T", "B", "general:rwdDtl")

	k.LockThread(ctx, root.ID, "user1", "general:rwdDtl")

	err := k.UnlockThread(ctx, root.ID, "user1", "general:rwd")
	if err == nil {
		t.Fatal("expected error for unlocking without 'l' in perms")
	}
}

func TestLockNonExistentThread(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	err := k.LockThread(ctx, "no-such-id", "mod1", "general:rwdDtl")
	if !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("expected ErrPostNotFound, got %v", err)
	}
}

func TestMoveThreadWithoutPermission(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	root, _ := k.CreateThread(ctx, "general", "user1", "T", "B", "general:rwd")
	err := k.MoveThread(ctx, root.ID, "other", "user1", "general:rw")
	if err == nil {
		t.Fatal("expected error for moving without 'm'")
	}
}

func TestMoveThreadNonRoot(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	root, _ := k.CreateThread(ctx, "general", "user1", "T", "B", "general:rwd")
	reply, _ := k.CreatePost(ctx, root.ID, "user2", "R", "general:rwd")

	// moving a non-thread-root via MoveThread should work (it moves the post
	// and its descendants, just like a root - the move just changes path)
	err := k.MoveThread(ctx, reply.ID, "other", "user1", "general:m")
	if err != nil {
		t.Fatal(err)
	}

	moved, _ := k.getPostByID(ctx, reply.ID)
	if moved.Path != "other" {
		t.Fatalf("reply path = %q, want %q", moved.Path, "other")
	}
}

func TestMovePostWithoutPermission(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	root1, _ := k.CreateThread(ctx, "general", "user1", "T1", "B1", "general:rwd")
	root2, _ := k.CreateThread(ctx, "general", "user1", "T2", "B2", "general:rwd")
	reply, _ := k.CreatePost(ctx, root1.ID, "user2", "Moving", "general:rwd")

	err := k.MovePost(ctx, reply.ID, root2.ID, "user1", "general:rw")
	if err == nil {
		t.Fatal("expected error for moving without 'm'")
	}
}

func TestMovePostToNonExistentParent(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	root, _ := k.CreateThread(ctx, "general", "user1", "T", "B", "general:rwd")
	reply, _ := k.CreatePost(ctx, root.ID, "user2", "Moving", "general:rwd")

	err := k.MovePost(ctx, reply.ID, "no-such-id", "mod1", "general:mw")
	if !errors.Is(err, ErrInvalidParent) && !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("expected ErrInvalidParent or ErrPostNotFound, got %v", err)
	}
}

func TestSearchNoQuery(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	k.CreateThread(ctx, "general", "u", "Title", "Body", "general:rwd")

	results, err := k.Search(ctx, "", "", "general:r", SearchOptions{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("empty query should match everything")
	}
}

func TestSearchNoMatch(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	k.CreateThread(ctx, "general", "u", "Hello", "World", "general:rwd")

	results, err := k.Search(ctx, "ZZZZNOSUCH", "", "general:r", SearchOptions{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestSearchExcludePaths(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	k.CreateThread(ctx, "general", "u1", "Discussion", "Interesting", "general:rwd")
	k.CreateThread(ctx, "other", "u2", "Discussion", "Interesting", "other:rwd")

	results, err := k.Search(ctx, "Discussion", "general", "general:r", SearchOptions{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result in 'general', got %d", len(results))
	}
	if results[0].Path != "general" {
		t.Fatalf("expected path 'general', got %q", results[0].Path)
	}
}

func TestPreferencesOverwrite(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	k.SetPreference(ctx, "user1", "key", "value1")
	k.SetPreference(ctx, "user1", "key", "value2")

	val, _ := k.GetPreference(ctx, "user1", "key")
	if val != "value2" {
		t.Fatalf("expected 'value2', got %q", val)
	}
}

func TestPreferencesMultipleUsers(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	k.SetPreference(ctx, "user1", "theme", "dark")
	k.SetPreference(ctx, "user2", "theme", "light")

	v1, _ := k.GetPreference(ctx, "user1", "theme")
	v2, _ := k.GetPreference(ctx, "user2", "theme")

	if v1 != "dark" {
		t.Fatalf("user1: expected 'dark', got %q", v1)
	}
	if v2 != "light" {
		t.Fatalf("user2: expected 'light', got %q", v2)
	}
}

func TestPreferenceNonexistent(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	val, err := k.GetPreference(ctx, "nosuchuser", "nosuchkey")
	if err != nil {
		t.Fatal(err)
	}
	if val != "" {
		t.Fatalf("expected empty string for nonexistent preference, got %q", val)
	}
}

func TestPreferenceDeleteNonexistent(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	// should not error
	err := k.DeletePreference(ctx, "nosuchuser", "nosuchkey")
	if err != nil {
		t.Fatal(err)
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

	// user with no blog permission shouldn't see blog posts
	blogThreads, _ := k.ListThreads(ctx, "blog/post-1", "other:r", ListOptions{})
	if len(blogThreads) != 0 {
		t.Fatal("expected 0 blog threads without blog permission")
	}

	// user with blog permission should see them
	blogThreads, _ = k.ListThreads(ctx, "blog/post-1", "blog:r", ListOptions{})
	if len(blogThreads) != 1 {
		t.Fatalf("expected 1 blog thread, got %d", len(blogThreads))
	}
}

func TestDeepNestedReplies(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	root, _ := k.CreateThread(ctx, "general", "user1", "Root", "B", "general:rwd")
	// create a chain: root -> a -> b -> c -> d
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

func TestMetadataPersistence(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	post, _ := k.CreateThread(ctx, "general", "user1", "T", "B", "general:rwd")

	// update metadata via raw query
	meta := `{"key":"value","number":42}`
	_, err := k.db.ExecContext(ctx,
		`UPDATE posts SET metadata = $1 WHERE id = $2`, meta, post.ID)
	if err != nil {
		t.Fatal(err)
	}

	// verify metadata is persisted
	fetched, _ := k.getPostByID(ctx, post.ID)
	if fetched.Metadata["key"] != "value" {
		t.Fatalf("metadata key = %v, want 'value'", fetched.Metadata["key"])
	}
	if n, ok := fetched.Metadata["number"].(float64); !ok || n != 42 {
		t.Fatalf("metadata number = %v, want 42", fetched.Metadata["number"])
	}
}

func TestEmptyStringPermissions(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	// calling with empty permission string for a read operation
	threads, err := k.ListThreads(ctx, "general", "", ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if threads != nil && len(threads) != 0 {
		t.Log("expected 0 threads with empty permissions")
	}
}

func TestListThreadsOrder(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	now := k.nowMilli()
	// insert posts with staggered timestamps
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

	// most recent first
	if threads[0].CreatedAt.Before(threads[1].CreatedAt) {
		t.Fatal("threads should be ordered by created_at DESC")
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

func TestGetThreadOtherPathAccess(t *testing.T) {
	k := setupDB(t)
	ctx := context.Background()

	root, _ := k.CreateThread(ctx, "general", "u1", "T", "B", "general:rwd")

	// user with 'a:r' (not 'general:r') should get silent deny
	_, err := k.GetThread(ctx, root.ID, "a:r")
	if !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("expected ErrPostNotFound, got %v", err)
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

func TestMultiplePermissionGroupsSamePath(t *testing.T) {
	p, err := ParsePermissions("general:rw,general:rwd")
	if err != nil {
		t.Fatal(err)
	}
	// the last value for 'general' overwrites the first in the map
	if !p.Has("general", 'd') {
		t.Error("expected 'd' from later group")
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
	db.Stats() // should not panic
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

func TestZeroValuePermissions(t *testing.T) {
	var p Permissions
	if p == nil {
		t.Log("zero value Permissions is nil, should not panic on Has")
	}
	// calling Has on nil map should not panic
	result := p.Has("any", 'r')
	if result {
		t.Error("nil permissions should give no access")
	}
}

func TestIntersectEmpty(t *testing.T) {
	r := intersect("abc", "")
	if r != "" {
		t.Fatalf("expected empty, got %q", r)
	}
	r = intersect("", "abc")
	if r != "" {
		t.Fatalf("expected empty, got %q", r)
	}
}

func TestIntersectNoOverlap(t *testing.T) {
	r := intersect("abc", "xyz")
	if r != "" {
		t.Fatalf("expected empty, got %q", r)
	}
}

func TestIntersectPartial(t *testing.T) {
	r := intersect("abcd", "cdef")
	if r != "cd" {
		t.Fatalf("expected 'cd', got %q", r)
	}
}

func TestIntersectFull(t *testing.T) {
	r := intersect("abc", "abc")
	if r != "abc" {
		t.Fatalf("expected 'abc', got %q", r)
	}
}
