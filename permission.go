package karu

import "strings"

type Permissions map[string]string

func ParsePermissions(s string) (Permissions, error) {
	p := make(Permissions)
	s = strings.TrimSpace(s)
	if s == "" {
		return p, nil
	}
	for _, group := range strings.Split(s, ",") {
		group = strings.TrimSpace(group)
		if group == "" {
			continue
		}
		parts := strings.SplitN(group, ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, ErrInvalidPermission
		}
		p[parts[0]] = parts[1]
	}
	return p, nil
}

func (p Permissions) Codes(path string) string {
	parts := strings.Split(path, "/")
	var codes string
	init := false
	for i := len(parts); i > 0; i-- {
		check := strings.Join(parts[:i], "/")
		if c, ok := p[check]; ok {
			if !init {
				codes = c
				init = true
			} else {
				codes = intersect(codes, c)
			}
		}
	}
	return codes
}

func (p Permissions) Has(path string, code byte) bool {
	return strings.ContainsRune(p.Codes(path), rune(code))
}

func intersect(a, b string) string {
	var sb strings.Builder
	for i := 0; i < len(a); i++ {
		if strings.ContainsRune(b, rune(a[i])) {
			sb.WriteByte(a[i])
		}
	}
	return sb.String()
}
