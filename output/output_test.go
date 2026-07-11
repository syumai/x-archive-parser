package output_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/syumai/x-archive-parser/output"
	"github.com/syumai/x-archive-parser/post"
)

// jst is a fixed timezone used throughout the tests so expectations don't
// depend on the tzdata database being available or on the test machine's
// local timezone.
var jst = time.FixedZone("Asia/Tokyo", 9*3600)

func fixturePosts() []*post.Post {
	return []*post.Post{
		{
			ID:            "1",
			URL:           "https://twitter.com/example/status/1",
			CreatedAt:     time.Date(2026, 3, 1, 10, 23, 0, 0, jst),
			Type:          post.TypePost,
			Text:          "Go の WebAssembly サポートについて調べている。 https://example.com/article",
			Lang:          "ja",
			FavoriteCount: 10,
			RetweetCount:  2,
		},
		{
			ID:        "2",
			URL:       "https://twitter.com/example/status/2",
			CreatedAt: time.Date(2026, 3, 1, 12, 45, 0, 0, jst),
			Type:      post.TypeRepost,
			Text:      "wasmtime の新リリースが出た…",
			RepostOf:  "@someone",
			Lang:      "ja",
		},
		{
			ID:         "3",
			URL:        "https://twitter.com/example/status/3",
			CreatedAt:  time.Date(2026, 3, 1, 18, 2, 0, 0, jst),
			Type:       post.TypeReply,
			Text:       "それは syscall/js の制約ですね",
			ReplyTo:    "@friend",
			ReplyToURL: "https://twitter.com/friend/status/999",
			Lang:       "ja",
		},
		{
			// Second calendar day within the same month.
			ID:        "4",
			URL:       "https://twitter.com/example/status/4",
			CreatedAt: time.Date(2026, 3, 2, 9, 0, 0, 0, jst),
			Type:      post.TypePost,
			Text:      "multi-line post:\nline two\nline three",
			Lang:      "en",
		},
		{
			// A different month.
			ID:        "5",
			URL:       "https://twitter.com/example/status/5",
			CreatedAt: time.Date(2026, 4, 5, 8, 0, 0, 0, jst),
			Type:      post.TypePost,
			Text:      "April post",
			Lang:      "en",
		},
	}
}

func testMeta() output.Meta {
	return output.Meta{Username: "example", TZLabel: "Asia/Tokyo"}
}

const wantIndexMD = `# X post archive of @example

- Range: 2026-03-01 to 2026-04-05 (Asia/Tokyo)
- Total: 5 posts (post 3 / repost 1 / reply 1)

## How to use this archive

- All timestamps and month boundaries use Asia/Tokyo.
- Each month is available as ` + "`YYYY/YYYY-MM.md`" + ` (for reading) and ` + "`YYYY/YYYY-MM.jsonl`" + ` (for grep/jq).
- Search keywords against the jsonl files: each hit is a single line containing both metadata and text.
  Example: ` + "`grep -il wasm */[0-9]*.jsonl`" + ` to locate months, then read those months' .md files.
- ` + "`type`" + ` is one of: post (own post) / repost (RT; text may be truncated to ~140 chars) / reply.

## Posts per month

| Month | Total | post | repost | reply |
|---|---|---|---|---|
| [2026-03](2026/2026-03.md) | 4 | 2 | 1 | 1 |
| [2026-04](2026/2026-04.md) | 1 | 1 | 0 | 0 |
`

const wantMarch2026MD = `# 2026-03 (@example)

4 posts (post 2 / repost 1 / reply 1)

## 2026-03-01 (Sun)

- **10:23 post** https://twitter.com/example/status/1
  Go の WebAssembly サポートについて調べている。 https://example.com/article
- **12:45 repost** (RT of @someone) https://twitter.com/example/status/2
  wasmtime の新リリースが出た…
- **18:02 reply** (to @friend) https://twitter.com/example/status/3
  それは syscall/js の制約ですね

## 2026-03-02 (Mon)

- **09:00 post** https://twitter.com/example/status/4
  multi-line post:
  line two
  line three
`

func TestWrite_EndToEnd(t *testing.T) {
	dir := t.TempDir()
	posts := fixturePosts()

	summary, err := output.Write(dir, testMeta(), posts)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	// --- Summary ---
	if summary.Total != 5 {
		t.Errorf("Total = %d, want 5", summary.Total)
	}
	if summary.Posts != 3 {
		t.Errorf("Posts = %d, want 3", summary.Posts)
	}
	if summary.Reposts != 1 {
		t.Errorf("Reposts = %d, want 1", summary.Reposts)
	}
	if summary.Replies != 1 {
		t.Errorf("Replies = %d, want 1", summary.Replies)
	}
	if !summary.From.Equal(posts[0].CreatedAt) {
		t.Errorf("From = %v, want %v", summary.From, posts[0].CreatedAt)
	}
	if !summary.To.Equal(posts[4].CreatedAt) {
		t.Errorf("To = %v, want %v", summary.To, posts[4].CreatedAt)
	}

	// --- index.md ---
	gotIndex := readFile(t, filepath.Join(dir, "index.md"))
	if gotIndex != wantIndexMD {
		t.Errorf("index.md mismatch:\n--- got ---\n%s\n--- want ---\n%s", gotIndex, wantIndexMD)
	}

	// --- 2026-03.md (also covers multi-line text indentation) ---
	gotMarchMD := readFile(t, filepath.Join(dir, "2026", "2026-03.md"))
	if gotMarchMD != wantMarch2026MD {
		t.Errorf("2026-03.md mismatch:\n--- got ---\n%s\n--- want ---\n%s", gotMarchMD, wantMarch2026MD)
	}

	// --- 2026-03.jsonl ---
	wantJSONL := marshalJSONLLines(t, posts[0], posts[1], posts[2], posts[3])
	gotJSONL := readFile(t, filepath.Join(dir, "2026", "2026-03.jsonl"))
	if gotJSONL != wantJSONL {
		t.Errorf("2026-03.jsonl mismatch:\n--- got ---\n%s\n--- want ---\n%s", gotJSONL, wantJSONL)
	}

	// --- 2026-04.jsonl (single-post month) ---
	wantAprilJSONL := marshalJSONLLines(t, posts[4])
	gotAprilJSONL := readFile(t, filepath.Join(dir, "2026", "2026-04.jsonl"))
	if gotAprilJSONL != wantAprilJSONL {
		t.Errorf("2026-04.jsonl mismatch:\n--- got ---\n%s\n--- want ---\n%s", gotAprilJSONL, wantAprilJSONL)
	}
}

func TestWrite_SafetyCheck(t *testing.T) {
	t.Run("refuses non-empty directory without index.md", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "some-other-file.txt"), []byte("not ours"), 0o644); err != nil {
			t.Fatalf("setup: %v", err)
		}

		_, err := output.Write(dir, testMeta(), fixturePosts())
		if err == nil {
			t.Fatal("Write into unrelated non-empty directory: got nil error, want error")
		}
	})

	t.Run("re-running over a previous output succeeds and overwrites", func(t *testing.T) {
		dir := t.TempDir()
		posts := fixturePosts()

		if _, err := output.Write(dir, testMeta(), posts); err != nil {
			t.Fatalf("first Write: %v", err)
		}
		// Second run over the same directory (produced by this tool) must
		// succeed and simply overwrite the files.
		summary, err := output.Write(dir, testMeta(), posts)
		if err != nil {
			t.Fatalf("second Write: %v", err)
		}
		if summary.Total != 5 {
			t.Errorf("Total = %d, want 5", summary.Total)
		}

		got := readFile(t, filepath.Join(dir, "index.md"))
		if got != wantIndexMD {
			t.Errorf("index.md mismatch after re-run:\n--- got ---\n%s\n--- want ---\n%s", got, wantIndexMD)
		}
	})

	t.Run("accepts an empty existing directory", func(t *testing.T) {
		dir := t.TempDir()
		if _, err := output.Write(dir, testMeta(), fixturePosts()); err != nil {
			t.Fatalf("Write into empty existing directory: %v", err)
		}
	})

	t.Run("creates the directory if it does not exist", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "nested", "posts")
		if _, err := output.Write(dir, testMeta(), fixturePosts()); err != nil {
			t.Fatalf("Write into non-existent directory: %v", err)
		}
		if _, err := os.Stat(filepath.Join(dir, "index.md")); err != nil {
			t.Errorf("index.md not created: %v", err)
		}
	})
}

func TestWrite_NoPosts(t *testing.T) {
	dir := t.TempDir()
	summary, err := output.Write(dir, testMeta(), nil)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if summary.Total != 0 {
		t.Errorf("Total = %d, want 0", summary.Total)
	}
	if !summary.From.IsZero() || !summary.To.IsZero() {
		t.Errorf("From/To should be zero, got From=%v To=%v", summary.From, summary.To)
	}

	const want = `# X post archive of @example

- Total: 0 posts

## How to use this archive

- All timestamps and month boundaries use Asia/Tokyo.
- Each month is available as ` + "`YYYY/YYYY-MM.md`" + ` (for reading) and ` + "`YYYY/YYYY-MM.jsonl`" + ` (for grep/jq).
- Search keywords against the jsonl files: each hit is a single line containing both metadata and text.
  Example: ` + "`grep -il wasm */[0-9]*.jsonl`" + ` to locate months, then read those months' .md files.
- ` + "`type`" + ` is one of: post (own post) / repost (RT; text may be truncated to ~140 chars) / reply.

## Posts per month

| Month | Total | post | repost | reply |
|---|---|---|---|---|
`

	got := readFile(t, filepath.Join(dir, "index.md"))
	if got != want {
		t.Errorf("index.md mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestWrite_SortsAscendingByCreatedAtThenID(t *testing.T) {
	dir := t.TempDir()
	// Deliberately out of order, including a same-timestamp tie broken by ID.
	tie := time.Date(2026, 5, 1, 0, 0, 0, 0, jst)
	posts := []*post.Post{
		{ID: "20", URL: "https://twitter.com/example/status/20", CreatedAt: time.Date(2026, 5, 2, 0, 0, 0, 0, jst), Type: post.TypePost, Text: "second"},
		{ID: "2", URL: "https://twitter.com/example/status/2", CreatedAt: tie, Type: post.TypePost, Text: "tie-b"},
		{ID: "1", URL: "https://twitter.com/example/status/1", CreatedAt: tie, Type: post.TypePost, Text: "tie-a"},
	}

	if _, err := output.Write(dir, testMeta(), posts); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got := readFile(t, filepath.Join(dir, "2026", "2026-05.jsonl"))
	var gotIDs []string
	for _, line := range splitNonEmptyLines(got) {
		var p post.Post
		if err := json.Unmarshal([]byte(line), &p); err != nil {
			t.Fatalf("unmarshal line %q: %v", line, err)
		}
		gotIDs = append(gotIDs, p.ID)
	}
	wantIDs := []string{"1", "2", "20"}
	if len(gotIDs) != len(wantIDs) {
		t.Fatalf("got %d ids, want %d: %v", len(gotIDs), len(wantIDs), gotIDs)
	}
	for i := range wantIDs {
		if gotIDs[i] != wantIDs[i] {
			t.Errorf("id[%d] = %q, want %q (got order %v)", i, gotIDs[i], wantIDs[i], gotIDs)
		}
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %q: %v", path, err)
	}
	return string(b)
}

func marshalJSONLLines(t *testing.T, posts ...*post.Post) string {
	t.Helper()
	var out string
	for _, p := range posts {
		b, err := json.Marshal(p)
		if err != nil {
			t.Fatalf("marshaling post %s: %v", p.ID, err)
		}
		out += string(b) + "\n"
	}
	return out
}

func splitNonEmptyLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			if line := s[start:i]; line != "" {
				lines = append(lines, line)
			}
			start = i + 1
		}
	}
	return lines
}
