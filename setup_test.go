package karu

import (
	"context"
	"testing"
)

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
