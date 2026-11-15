package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const artifactJQ = `def clean:
  if type == "string" then
    gsub("\t"; " ") | gsub("\r"; " ") | gsub("\n"; " ") | gsub("\u001f"; " ")
  else
    .
  end;
"T \(.total_count | tostring)",
(
  .artifacts[]?
  | [
      (.id | tostring),
      ((.name // "") | clean),
      ((.size_in_bytes // 0) | tostring),
      (.created_at // ""),
      (if .expired then "1" else "0" end),
      (((.workflow_run // {}) | .head_branch // "") | clean),
      (((.workflow_run // {}) | .id // 0) | tostring)
    ]
  | join("\u001f")
)`

const cacheJQ = `def clean:
  if type == "string" then
    gsub("\t"; " ") | gsub("\r"; " ") | gsub("\n"; " ") | gsub("\u001f"; " ")
  else
    .
  end;
.[]?
| [
    (.id | tostring),
    ((.key // "") | clean),
    ((.sizeInBytes // 0) | tostring),
    (.createdAt // ""),
    ((.ref // "") | clean),
    (.lastAccessedAt // "")
  ]
| join("\u001f")`

const projectListCap = 200
const cacheListCap = 10000

var runGHFunc = runGH

func runGH(args ...string) (string, error) {
	cmd := exec.Command("gh", args...)
	cmd.Stdin = bytes.NewReader(nil)
	cmd.Env = replaceEnv(os.Environ(),
		"GH_PROMPT_DISABLED=1",
		"GH_NO_UPDATE_NOTIFIER=1",
		"GH_PAGER=cat",
		"PAGER=cat",
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		msg = strings.TrimPrefix(msg, "gh: ")
		if msg == "" {
			msg = err.Error()
		}
		return stdout.String(), fmt.Errorf("%s", firstLine(msg))
	}
	return stdout.String(), nil
}

func replaceEnv(base []string, kv ...string) []string {
	skip := map[string]bool{}
	for _, p := range kv {
		k, _, _ := strings.Cut(p, "=")
		skip[k] = true
	}
	out := make([]string, 0, len(base)+len(kv))
	for _, e := range base {
		k, _, _ := strings.Cut(e, "=")
		if skip[k] {
			continue
		}
		out = append(out, e)
	}
	return append(out, kv...)
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}

func ghAuth() error {
	_, err := runGHFunc("auth", "status")
	if err != nil {
		return fmt.Errorf("gh is not logged in. Run: gh auth login")
	}
	return nil
}

func listRepoNames(who string) ([]string, error) {
	args := []string{"repo", "list"}
	if who != "" {
		args = append(args, who)
	}
	args = append(args, "--limit", fmt.Sprint(projectListCap), "--json", "nameWithOwner", "--jq", ".[].nameWithOwner")
	out, err := runGHFunc(args...)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			names = append(names, line)
		}
	}
	return names, nil
}

func detectCheckout() (string, error) {
	out, err := runGHFunc("repo", "view", "--json", "nameWithOwner,url", "-q", `[.nameWithOwner, .url] | @tsv`)
	if err != nil {
		return "", err
	}
	nwo, url, _ := strings.Cut(strings.TrimSpace(out), "\t")
	if nwo == "" {
		return "", fmt.Errorf("could not detect a repository")
	}
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		rest := url
		if i := strings.Index(rest, "://"); i >= 0 {
			rest = rest[i+3:]
		}
		host, _, _ := strings.Cut(rest, "/")
		if host != "" && host != "github.com" && host != "www.github.com" && host != "api.github.com" {
			return host + "/" + nwo, nil
		}
	}
	return nwo, nil
}

func fetchArtifacts(p Project) (string, error) {
	args := []string{"api"}
	if p.APIHost != "" {
		args = append(args, "--hostname", p.APIHost)
	}
	args = append(args,
		"--paginate",
		"-H", "Accept: application/vnd.github+json",
		fmt.Sprintf("repos/%s/%s/actions/artifacts?per_page=100", p.Owner, p.Name),
		"--jq", artifactJQ,
	)
	return runGHFunc(args...)
}

func fetchCaches(p Project) (string, error) {
	return runGHFunc(
		"cache", "list",
		"-R", p.NWO,
		"--limit", fmt.Sprint(cacheListCap),
		"--sort", "size_in_bytes",
		"--order", "desc",
		"--json", "id,key,ref,sizeInBytes,createdAt,lastAccessedAt",
		"--jq", cacheJQ,
	)
}

func parseArtifacts(raw string) (items []Item, total int) {
	seen := map[string]bool{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "T ") {
			total = int(atoi64(strings.TrimPrefix(line, "T ")))
			continue
		}
		f := strings.Split(line, "\x1f")
		if len(f) < 7 {
			continue
		}
		id := stripControls(f[0])
		if id == "" || seen["a:"+id] {
			continue
		}
		seen["a:"+id] = true
		name := stripControls(f[1])
		if name == "" {
			name = "(unnamed)"
		}
		if len([]rune(name)) > 300 {
			name = string([]rune(name)[:300])
		}
		items = append(items, Item{
			Kind:    "artifact",
			ID:      id,
			Name:    name,
			Size:    atoi64(f[2]),
			When:    parseGitHubTime(f[3]),
			Expired: strings.TrimSpace(f[4]) == "1",
			Meta:    shortenRef(stripControls(f[5])),
			Run:     stripControls(f[6]),
			Group:   -1,
		})
	}
	return items, total
}

func parseCaches(raw string) []Item {
	seen := map[string]bool{}
	var items []Item
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		f := strings.Split(line, "\x1f")
		if len(f) < 6 {
			continue
		}
		id := stripControls(f[0])
		if id == "" || seen["c:"+id] {
			continue
		}
		seen["c:"+id] = true
		key := stripControls(f[1])
		if key == "" {
			key = "(unnamed)"
		}
		when := parseGitHubTime(f[5])
		if when.IsZero() {
			when = parseGitHubTime(f[3])
		}
		ref := stripControls(f[4])
		items = append(items, Item{
			Kind:  "cache",
			ID:    id,
			Name:  key,
			Size:  atoi64(f[2]),
			When:  when,
			Meta:  shortenRef(ref),
			Run:   ref,
			Group: -1,
		})
	}
	return items
}

func loadProject(base Project, wantArt, wantCache bool) (Project, error) {
	on := base.On
	var items []Item
	var artErr, cacheErr error
	var total int
	if wantArt {
		raw, err := fetchArtifacts(base)
		if err != nil {
			artErr = err
		} else {
			var arts []Item
			arts, total = parseArtifacts(raw)
			items = append(items, arts...)
		}
	}
	if wantCache {
		raw, err := fetchCaches(base)
		if err != nil {
			cacheErr = err
		} else {
			items = append(items, parseCaches(raw)...)
		}
	}
	if wantArt && artErr != nil && (!wantCache || cacheErr != nil) {
		return base, artErr
	}
	if wantCache && !wantArt && cacheErr != nil {
		return base, cacheErr
	}
	base.Items = items
	base.On = on
	base.Err = ""
	if artErr != nil {
		base.Err = artErr.Error()
	} else if cacheErr != nil {
		base.Err = cacheErr.Error()
	}
	base.rebuildGroups()
	if wantArt && total > base.ArtN {
		note := fmt.Sprintf("GitHub reports %d artifacts; the API returned %d.", total, base.ArtN)
		if base.Err != "" {
			base.Err = note + " " + base.Err
		} else {
			base.Err = note
		}
	}
	return base, nil
}

func rateLimited(msg string) bool {
	l := strings.ToLower(msg)
	return strings.Contains(l, "rate limit") ||
		strings.Contains(l, "http 429") ||
		strings.Contains(l, "http 502") ||
		strings.Contains(l, "http 503") ||
		strings.Contains(l, "http 504")
}

func deleteArtifact(p Project, id string) error {
	args := []string{"api"}
	if p.APIHost != "" {
		args = append(args, "--hostname", p.APIHost)
	}
	args = append(args, "-X", "DELETE", fmt.Sprintf("repos/%s/%s/actions/artifacts/%s", p.Owner, p.Name, id))
	return retryGH(args...)
}

func deleteCache(p Project, id string) error {
	return retryGH("cache", "delete", "-R", p.NWO, id)
}

func retryGH(args ...string) error {
	wait := 2 * time.Second
	var last error
	for attempt := 0; attempt < 6; attempt++ {
		_, err := runGHFunc(args...)
		if err == nil {
			return nil
		}
		last = err
		if !rateLimited(err.Error()) || attempt == 5 {
			return err
		}
		time.Sleep(wait)
		if wait < 16*time.Second {
			wait *= 2
		}
	}
	return last
}
