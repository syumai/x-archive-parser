package output

import (
	"fmt"
	"os"
	"strings"

	"github.com/syumai/x-archive-parser/post"
)

// writeMonthMarkdown renders the Markdown file for a single month.
func writeMonthMarkdown(path string, meta Meta, g monthGroup) error {
	content := buildMonthMarkdown(meta, g)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing %q: %w", path, err)
	}
	return nil
}

// buildMonthMarkdown builds the Markdown content for a single month: a
// title, a summary line, and one day section per calendar day, each
// containing one bullet per post. Post text is emitted verbatim (not
// Markdown-escaped) and indented under its bullet.
func buildMonthMarkdown(meta Meta, g monthGroup) string {
	ms := summarizeGroup(g)

	var lines []string
	lines = append(lines, fmt.Sprintf("# %04d-%02d (@%s)", g.Year, int(g.Month), meta.Username))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("%s posts (post %s / repost %s / reply %s)",
		formatThousands(ms.Total), formatThousands(ms.Posts), formatThousands(ms.Reposts), formatThousands(ms.Replies)))

	curDay := -1
	for _, p := range g.Posts {
		day := p.CreatedAt.Day()
		if day != curDay {
			lines = append(lines, "")
			lines = append(lines, "## "+p.CreatedAt.Format("2006-01-02 (Mon)"))
			lines = append(lines, "")
			curDay = day
		}
		lines = append(lines, bulletLine(p))
		for _, textLine := range strings.Split(p.Text, "\n") {
			lines = append(lines, "  "+textLine)
		}
	}

	return strings.Join(lines, "\n") + "\n"
}

// bulletLine renders the "- **HH:MM type** [(RT of @x)|(to @x)] URL" line
// for a single post.
func bulletLine(p *post.Post) string {
	var annotation string
	switch p.Type {
	case post.TypeRepost:
		if p.RepostOf != "" {
			annotation = fmt.Sprintf(" (RT of %s)", p.RepostOf)
		}
	case post.TypeReply:
		if p.ReplyTo != "" {
			annotation = fmt.Sprintf(" (to %s)", p.ReplyTo)
		}
	}
	return fmt.Sprintf("- **%s %s**%s %s", p.CreatedAt.Format("15:04"), string(p.Type), annotation, p.URL)
}
