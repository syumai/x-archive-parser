package post

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/syumai/x-archive-parser/archive"
)

// createdAtLayout matches the format used for Tweet.CreatedAt, e.g.
// "Fri Jul 10 15:56:55 +0000 2026".
const createdAtLayout = "Mon Jan 2 15:04:05 -0700 2006"

// truncatedSuffixRe matches the trailing ellipsis of a truncated repost
// text, optionally followed by a single trailing URL (e.g. "…" or
// "… https://example.com/x").
var truncatedSuffixRe = regexp.MustCompile(`…( https?://\S+)?$`)

// FromTweet normalizes a raw archive tweet into a Post.
//
// It expands t.co URLs found in entities.urls, classifies the post as a
// repost, reply, or plain post (in that priority order), and converts
// CreatedAt to loc.
func FromTweet(t *archive.Tweet, username string, loc *time.Location) (*Post, error) {
	createdAt, err := time.Parse(createdAtLayout, t.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("post: parsing created_at for tweet %s: %w", t.IDStr, err)
	}

	text := expandURLs(t.FullText, t.Entities.URLs)

	p := &Post{
		ID:            t.IDStr,
		URL:           fmt.Sprintf("https://twitter.com/%s/status/%s", username, t.IDStr),
		CreatedAt:     createdAt.In(loc),
		Text:          text,
		Lang:          t.Lang,
		FavoriteCount: parseCount(t.FavoriteCount),
		RetweetCount:  parseCount(t.RetweetCount),
	}

	switch {
	case strings.HasPrefix(text, "RT @"):
		// Pattern: "RT @<handle>: <rest>". If the ": " separator is
		// missing, the text is malformed; keep it as-is but still
		// classify as a repost.
		p.Type = TypeRepost
		rest := text[len("RT @"):]
		if idx := strings.Index(rest, ": "); idx >= 0 {
			p.RepostOf = "@" + rest[:idx]
			p.Text = rest[idx+len(": "):]
		}

	case t.InReplyToStatusIDStr != "":
		p.Type = TypeReply
		if t.InReplyToScreenName != "" {
			p.ReplyTo = "@" + t.InReplyToScreenName
			p.ReplyToURL = fmt.Sprintf("https://twitter.com/%s/status/%s", t.InReplyToScreenName, t.InReplyToStatusIDStr)
		} else {
			p.ReplyToURL = fmt.Sprintf("https://twitter.com/i/web/status/%s", t.InReplyToStatusIDStr)
		}

	default:
		p.Type = TypePost
	}

	return p, nil
}

// expandURLs replaces every t.co short URL in text with its expanded form,
// for each entity that has both a URL and a non-empty expanded URL.
func expandURLs(text string, urls []archive.URLEntity) string {
	for _, u := range urls {
		if u.URL == "" || u.ExpandedURL == "" {
			continue
		}
		text = strings.ReplaceAll(text, u.URL, u.ExpandedURL)
	}
	return text
}

// parseCount parses an archive count field (stored as a string). Empty or
// unparseable values are treated as 0; parsing never fails.
func parseCount(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

// ApplyNoteTweets merges long-form "note tweet" text back into posts,
// best-effort.
//
// Note tweets (data/note-tweet.js) carry the untruncated text of long
// posts, but they have their own ID rather than the tweet's ID, so they
// cannot be matched exactly. Instead, each note is matched to a post by
// creation instant (excluding reposts, since retweeted text is not
// authored by the account owner and note tweets don't apply to them).
// When more than one post shares that instant, the match is narrowed
// further by checking that the post's text looks truncated (ends with
// "…", optionally followed by a trailing URL) and that the note's text
// starts with a prefix of the post's text (with the truncation marker and
// trailing URL stripped). If the timestamp uniquely identifies a single,
// apparently-truncated post, that match is used without the prefix check.
//
// This is inherently heuristic: it can miss a match (post left truncated)
// or, in rare cases of multiple truncated posts at the same instant with
// similar prefixes, pick the wrong one. No error is returned.
func ApplyNoteTweets(posts []*Post, notes []archive.NoteTweet) {
	for _, note := range notes {
		t, err := time.Parse(time.RFC3339, note.CreatedAt)
		if err != nil {
			continue
		}

		var candidates []*Post
		for _, p := range posts {
			if p.Type == TypeRepost {
				continue
			}
			if p.CreatedAt.Equal(t) {
				candidates = append(candidates, p)
			}
		}

		if len(candidates) == 1 {
			if p := candidates[0]; looksTruncated(p.Text) {
				p.Text = note.Text
			}
			continue
		}

		for _, p := range candidates {
			if !looksTruncated(p.Text) {
				continue
			}
			prefix := truncatedPrefix(p.Text, 50)
			if prefix != "" && strings.HasPrefix(note.Text, prefix) {
				p.Text = note.Text
				break
			}
		}
	}
}

// looksTruncated reports whether text ends with the archive's truncation
// marker ("…"), optionally followed by a single trailing URL.
func looksTruncated(text string) bool {
	return truncatedSuffixRe.MatchString(text)
}

// truncatedPrefix strips the truncation marker (and any trailing URL) from
// text and returns up to the first n runes of what remains, for use as a
// best-effort match against a note tweet's full text.
func truncatedPrefix(text string, n int) string {
	stripped := truncatedSuffixRe.ReplaceAllString(text, "")
	r := []rune(stripped)
	if len(r) > n {
		r = r[:n]
	}
	return string(r)
}
