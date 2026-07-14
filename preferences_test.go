package karu

import (
	"context"
	"testing"
)

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

	err := k.DeletePreference(ctx, "nosuchuser", "nosuchkey")
	if err != nil {
		t.Fatal(err)
	}
}
