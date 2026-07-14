package karu

import (
	"context"
	"testing"
)

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
