package karu

import (
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

func TestPermissionDuplicateCodes(t *testing.T) {
	p, err := ParsePermissions("a:rwrw")
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
	if codes := p.Codes("a"); codes != "" {
		t.Errorf("expected empty on parent, got %q", codes)
	}
	if !p.Has("a/b", 'r') {
		t.Error("expected 'r' on exact subpath")
	}
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

func TestMultiplePermissionGroupsSamePath(t *testing.T) {
	p, err := ParsePermissions("general:rw,general:rwd")
	if err != nil {
		t.Fatal(err)
	}
	if !p.Has("general", 'd') {
		t.Error("expected 'd' from later group")
	}
}

func TestZeroValuePermissions(t *testing.T) {
	var p Permissions
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
