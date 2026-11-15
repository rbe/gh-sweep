// Package project is one GitHub repository and the artifacts and caches in it.
package project

import (
	"fmt"
	"strings"
)

// Project is one GitHub repository the user can wipe or open.
type Project struct {
	Owner   string
	Name    string
	Host    string
	APIHost string
	NWO     string

	On         bool
	Scanned    bool
	Err        string
	ArtN       int
	ArtBytes   int64
	ExpN       int
	ExpBytes   int64
	CacheN     int
	CacheBytes int64

	Items       []Item
	Groups      []Group
	ArtGroups   []int
	CacheGroups []int
}

// parseRepo accepts owner/repo or host/owner/repo. github.com is treated as
// the public host. A trailing .git is ignored.
func Parse(raw string) (Project, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimSuffix(raw, ".git")
	parts := strings.Split(raw, "/")
	var host, owner, name string
	switch len(parts) {
	case 2:
		owner, name = parts[0], parts[1]
	case 3:
		host, owner, name = parts[0], parts[1], parts[2]
	default:
		return Project{}, fmt.Errorf("not a repository name: %s", raw)
	}
	name = strings.TrimSuffix(name, ".git")
	if !validOwner(owner) || !validName(name) {
		return Project{}, fmt.Errorf("not a repository name: %s", raw)
	}
	if host != "" {
		if !strings.Contains(host, ".") || strings.Contains(host, "..") || !validHost(host) {
			return Project{}, fmt.Errorf("not a repository name: %s", raw)
		}
		if host == "github.com" || host == "www.github.com" {
			host = ""
		}
	}
	p := Project{Owner: owner, Name: name, Host: host}
	if host != "" {
		p.APIHost = host
		p.NWO = host + "/" + owner + "/" + name
	} else {
		p.NWO = owner + "/" + name
	}
	return p, nil
}

func validOwner(s string) bool {
	if s == "." || s == ".." || s == "" {
		return false
	}
	for i, r := range s {
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_':
		case i > 0 && (r == '.' || r == '-'):
		default:
			return false
		}
	}
	return true
}

func validName(s string) bool {
	if s == "." || s == ".." || s == "" {
		return false
	}
	for i, r := range s {
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_', r == '.':
		case i > 0 && r == '-':
		default:
			return false
		}
	}
	return true
}

func validHost(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '.', r == '-':
		default:
			return false
		}
	}
	return true
}
