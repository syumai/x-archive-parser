package output

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// writeIndex renders index.md, the entry point for a coding agent
// navigating the archive: overall range/totals, usage notes, and a
// per-month table linking to each month's Markdown file.
func writeIndex(dir string, meta Meta, summary *Summary, months []monthSummary) error {
	content := buildIndex(meta, summary, months)
	path := filepath.Join(dir, "index.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing %q: %w", path, err)
	}
	return nil
}

func buildIndex(meta Meta, summary *Summary, months []monthSummary) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# X post archive of @%s\n\n", meta.Username)

	if summary.Total > 0 {
		fmt.Fprintf(&b, "- Range: %s to %s (%s)\n", summary.From.Format("2006-01-02"), summary.To.Format("2006-01-02"), meta.TZLabel)
		fmt.Fprintf(&b, "- Total: %s posts (post %s / repost %s / reply %s)\n\n",
			formatThousands(summary.Total), formatThousands(summary.Posts), formatThousands(summary.Reposts), formatThousands(summary.Replies))
	} else {
		b.WriteString("- Total: 0 posts\n\n")
	}

	b.WriteString("## How to use this archive\n\n")
	fmt.Fprintf(&b, "- All timestamps and month boundaries use %s.\n", meta.TZLabel)
	b.WriteString("- Each month is available as `YYYY/YYYY-MM.md` (for reading) and `YYYY/YYYY-MM.jsonl` (for grep/jq).\n")
	b.WriteString("- Search keywords against the jsonl files: each hit is a single line containing both metadata and text.\n")
	b.WriteString("  Example: `grep -il wasm */[0-9]*.jsonl` to locate months, then read those months' .md files.\n")
	b.WriteString("- `type` is one of: post (own post) / repost (RT; text may be truncated to ~140 chars) / reply.\n\n")

	b.WriteString("## Posts per month\n\n")
	b.WriteString("| Month | Total | post | repost | reply |\n")
	b.WriteString("|---|---|---|---|---|\n")
	for _, ms := range months {
		label := fmt.Sprintf("%04d-%02d", ms.Year, int(ms.Month))
		link := fmt.Sprintf("%04d/%s.md", ms.Year, label)
		fmt.Fprintf(&b, "| [%s](%s) | %s | %s | %s | %s |\n",
			label, link, formatThousands(ms.Total), formatThousands(ms.Posts), formatThousands(ms.Reposts), formatThousands(ms.Replies))
	}

	return b.String()
}

// formatThousands formats n with comma thousands separators, e.g. 1234567 ->
// "1,234,567".
func formatThousands(n int) string {
	s := strconv.Itoa(n)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}

	var out []byte
	l := len(s)
	for i := 0; i < l; i++ {
		if i > 0 && (l-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, s[i])
	}

	if neg {
		return "-" + string(out)
	}
	return string(out)
}
