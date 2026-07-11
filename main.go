// Command x-archive-parser parses an X (Twitter) archive and expands all
// posts (including reposts) into per-month Markdown and JSONL files,
// optimized for investigation by coding agents.
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/syumai/x-archive-parser/archive"
	"github.com/syumai/x-archive-parser/output"
	"github.com/syumai/x-archive-parser/post"
)

func main() {
	os.Exit(run())
}

func run() int {
	var (
		outDir = flag.String("o", "./posts", "output directory")
		tz     = flag.String("tz", "", "IANA timezone name (e.g. Asia/Tokyo, UTC) used for month boundaries and displayed times; defaults to the system local timezone")
	)
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "x-archive-parser parses an X (Twitter) archive into per-month Markdown and JSONL files for coding agents to investigate.")
		fmt.Fprintf(os.Stderr, "\nUsage:\n  %s [flags] [archive]\n\n  archive is the path to the archive zip file or extracted directory;\n  it defaults to the current directory if omitted. Flags must be given\n  before the archive argument.\n\nFlags:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	var archivePath string
	switch flag.NArg() {
	case 0:
		archivePath = "."
	case 1:
		archivePath = flag.Arg(0)
	default:
		flag.Usage()
		return 2
	}

	loc, tzLabel, err := resolveLocation(*tz)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	if err := convert(archivePath, *outDir, loc, tzLabel); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	return 0
}

// resolveLocation resolves the timezone to use, along with a human-readable
// label for it. An empty tz selects the system local timezone.
func resolveLocation(tz string) (*time.Location, string, error) {
	if tz != "" {
		loc, err := time.LoadLocation(tz)
		if err != nil {
			return nil, "", fmt.Errorf("loading timezone %q: %w", tz, err)
		}
		return loc, tz, nil
	}

	loc := time.Local
	now := time.Now().In(loc)
	abbr := now.Format("MST")
	offset := now.Format("-07:00")
	return loc, fmt.Sprintf("%s (UTC%s, system local)", abbr, offset), nil
}

// convert opens the archive, normalizes every tweet into a post.Post, and
// writes the per-month output files.
func convert(archivePath, outDir string, loc *time.Location, tzLabel string) error {
	a, err := archive.Open(archivePath)
	if err != nil {
		return err
	}
	defer a.Close()

	account, err := a.Account()
	if err != nil {
		return fmt.Errorf("reading account: %w", err)
	}

	var posts []*post.Post
	var skipped int
	for t, err := range a.Tweets() {
		if err != nil {
			return fmt.Errorf("reading tweets: %w", err)
		}
		p, err := post.FromTweet(t, account.Username, loc)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping unparsable tweet %s: %v\n", t.IDStr, err)
			skipped++
			continue
		}
		posts = append(posts, p)
	}

	notes, err := a.NoteTweets()
	if err != nil {
		return fmt.Errorf("reading note tweets: %w", err)
	}
	post.ApplyNoteTweets(posts, notes)

	summary, err := output.Write(outDir, output.Meta{Username: account.Username, TZLabel: tzLabel}, posts)
	if err != nil {
		return fmt.Errorf("writing output: %w", err)
	}

	printSummary(summary, outDir, tzLabel, skipped)
	return nil
}

// printSummary reports the resulting counts and date range to stderr.
func printSummary(s *output.Summary, outDir, tzLabel string, skipped int) {
	fmt.Fprintf(os.Stderr, "Wrote %s posts (post %s / repost %s / reply %s) to %s\n",
		formatThousands(s.Total), formatThousands(s.Posts), formatThousands(s.Reposts), formatThousands(s.Replies), outDir)
	if !s.From.IsZero() {
		fmt.Fprintf(os.Stderr, "Range: %s to %s (%s)\n",
			s.From.Format("2006-01-02"), s.To.Format("2006-01-02"), tzLabel)
	}
	if skipped > 0 {
		fmt.Fprintf(os.Stderr, "Skipped %s unparsable tweets\n", formatThousands(skipped))
	}
}

// formatThousands formats a non-negative integer with comma thousands
// separators, e.g. 1234567 -> "1,234,567".
func formatThousands(n int) string {
	s := strconv.Itoa(n)
	if len(s) <= 3 {
		return s
	}
	var out []byte
	rem := len(s) % 3
	if rem > 0 {
		out = append(out, s[:rem]...)
	}
	for i := rem; i < len(s); i += 3 {
		if len(out) > 0 {
			out = append(out, ',')
		}
		out = append(out, s[i:i+3]...)
	}
	return string(out)
}
