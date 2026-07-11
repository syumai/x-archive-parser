// Package output writes normalized posts to a directory of per-month
// Markdown and JSONL files, designed to be navigated by coding agents:
// find the right month by file name, grep the JSONL for a keyword, then
// read the Markdown for context.
package output

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/syumai/x-archive-parser/post"
)

// Meta holds display metadata used when rendering output files.
type Meta struct {
	// Username is the account handle without a leading "@".
	Username string
	// TZLabel is a human-readable description of the timezone used for
	// all timestamps and month/day boundaries (e.g. "Asia/Tokyo" or
	// "JST (UTC+09:00, system local)").
	TZLabel string
}

// Summary reports aggregate counts and the date range of the posts written.
type Summary struct {
	Total   int
	Posts   int
	Reposts int
	Replies int
	// From and To are the CreatedAt of the first and last posts written,
	// in ascending order. Both are zero if there were no posts.
	From time.Time
	To   time.Time
}

// monthGroup collects the posts belonging to a single calendar month, in
// ascending order.
type monthGroup struct {
	Year  int
	Month time.Month
	Posts []*post.Post
}

// monthSummary is the per-month aggregate used both in the monthly
// Markdown header and in the index.md table.
type monthSummary struct {
	Year    int
	Month   time.Month
	Total   int
	Posts   int
	Reposts int
	Replies int
}

// Write renders all posts into dir: one index.md, plus one Markdown file
// and one JSONL file per calendar month (under a YYYY subdirectory).
//
// dir is created if it does not exist. If dir exists and is non-empty, it
// must already contain an index.md (i.e. it must be a previous output of
// this tool) or Write returns an error, to avoid clobbering an unrelated
// directory. Re-running Write over a previous output overwrites its files
// and is idempotent.
func Write(dir string, meta Meta, posts []*post.Post) (*Summary, error) {
	if err := checkOutputDir(dir); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating output directory %q: %w", dir, err)
	}

	sorted := sortedPosts(posts)
	groups := groupByMonth(sorted)

	summary := &Summary{}
	monthSummaries := make([]monthSummary, 0, len(groups))
	for _, g := range groups {
		ms := summarizeGroup(g)
		monthSummaries = append(monthSummaries, ms)
		summary.Total += ms.Total
		summary.Posts += ms.Posts
		summary.Reposts += ms.Reposts
		summary.Replies += ms.Replies

		yearDir := filepath.Join(dir, fmt.Sprintf("%04d", g.Year))
		if err := os.MkdirAll(yearDir, 0o755); err != nil {
			return nil, fmt.Errorf("creating year directory %q: %w", yearDir, err)
		}

		base := fmt.Sprintf("%04d-%02d", g.Year, int(g.Month))
		if err := writeMonthMarkdown(filepath.Join(yearDir, base+".md"), meta, g); err != nil {
			return nil, err
		}
		if err := writeMonthJSONL(filepath.Join(yearDir, base+".jsonl"), g.Posts); err != nil {
			return nil, err
		}
	}

	if len(sorted) > 0 {
		summary.From = sorted[0].CreatedAt
		summary.To = sorted[len(sorted)-1].CreatedAt
	}

	if err := writeIndex(dir, meta, summary, monthSummaries); err != nil {
		return nil, err
	}

	return summary, nil
}

// checkOutputDir refuses to write into a non-empty directory that does not
// look like a previous output of this tool.
func checkOutputDir(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("checking output directory %q: %w", dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("output path %q exists and is not a directory", dir)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading output directory %q: %w", dir, err)
	}
	if len(entries) == 0 {
		return nil
	}
	for _, e := range entries {
		if !e.IsDir() && e.Name() == "index.md" {
			return nil
		}
	}
	return fmt.Errorf("output directory %q is non-empty and does not contain index.md; refusing to write into a directory that was not produced by this tool", dir)
}

// sortedPosts returns a new slice of posts sorted ascending by CreatedAt,
// tie-broken by ID. The input slice is left untouched.
func sortedPosts(posts []*post.Post) []*post.Post {
	sorted := make([]*post.Post, len(posts))
	copy(sorted, posts)
	sort.SliceStable(sorted, func(i, j int) bool {
		ti, tj := sorted[i].CreatedAt, sorted[j].CreatedAt
		if !ti.Equal(tj) {
			return ti.Before(tj)
		}
		return sorted[i].ID < sorted[j].ID
	})
	return sorted
}

// groupByMonth splits an ascending-sorted slice of posts into ascending
// per-calendar-month groups, using each post's own CreatedAt location.
func groupByMonth(sorted []*post.Post) []monthGroup {
	var groups []monthGroup
	for _, p := range sorted {
		y, m := p.CreatedAt.Year(), p.CreatedAt.Month()
		if n := len(groups); n > 0 && groups[n-1].Year == y && groups[n-1].Month == m {
			groups[n-1].Posts = append(groups[n-1].Posts, p)
			continue
		}
		groups = append(groups, monthGroup{Year: y, Month: m, Posts: []*post.Post{p}})
	}
	return groups
}

func summarizeGroup(g monthGroup) monthSummary {
	ms := monthSummary{Year: g.Year, Month: g.Month, Total: len(g.Posts)}
	for _, p := range g.Posts {
		switch p.Type {
		case post.TypePost:
			ms.Posts++
		case post.TypeRepost:
			ms.Reposts++
		case post.TypeReply:
			ms.Replies++
		}
	}
	return ms
}
