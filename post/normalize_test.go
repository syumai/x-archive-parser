package post

import (
	"testing"
	"time"

	"github.com/syumai/x-archive-parser/archive"
)

func TestFromTweet_CreatedAt(t *testing.T) {
	tokyo, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatalf("LoadLocation(Asia/Tokyo): %v", err)
	}

	tests := []struct {
		name      string
		createdAt string
		loc       *time.Location
		want      time.Time
	}{
		{
			name:      "UTC stays UTC",
			createdAt: "Fri Jul 10 15:56:55 +0000 2026",
			loc:       time.UTC,
			want:      time.Date(2026, time.July, 10, 15, 56, 55, 0, time.UTC),
		},
		{
			name:      "UTC converted to Asia/Tokyo crosses midnight",
			createdAt: "Tue Jun 30 23:30:00 +0000 2026",
			loc:       tokyo,
			want:      time.Date(2026, time.July, 1, 8, 30, 0, 0, tokyo),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tw := &archive.Tweet{
				IDStr:     "1",
				CreatedAt: tt.createdAt,
				FullText:  "hello",
			}
			p, err := FromTweet(tw, "someone", tt.loc)
			if err != nil {
				t.Fatalf("FromTweet: %v", err)
			}
			if !p.CreatedAt.Equal(tt.want) {
				t.Errorf("CreatedAt = %v, want %v", p.CreatedAt, tt.want)
			}
			if p.CreatedAt.Location().String() != tt.loc.String() {
				t.Errorf("CreatedAt location = %v, want %v", p.CreatedAt.Location(), tt.loc)
			}
		})
	}
}

func TestFromTweet_CreatedAtParseError(t *testing.T) {
	tw := &archive.Tweet{
		IDStr:     "12345",
		CreatedAt: "not a valid timestamp",
		FullText:  "hello",
	}
	_, err := FromTweet(tw, "someone", time.UTC)
	if err == nil {
		t.Fatal("expected an error for an unparseable created_at, got nil")
	}
	if got := err.Error(); !containsID(got, "12345") {
		t.Errorf("error %q does not mention the tweet ID", got)
	}
}

func containsID(s, id string) bool {
	for i := 0; i+len(id) <= len(s); i++ {
		if s[i:i+len(id)] == id {
			return true
		}
	}
	return false
}

func TestFromTweet_URLExpansion(t *testing.T) {
	tw := &archive.Tweet{
		IDStr:     "1",
		CreatedAt: "Fri Jul 10 15:56:55 +0000 2026",
		FullText:  "check https://t.co/aaa and https://t.co/bbb and https://t.co/ccc",
		Entities: archive.Entities{
			URLs: []archive.URLEntity{
				{URL: "https://t.co/aaa", ExpandedURL: "https://example.com/first"},
				{URL: "https://t.co/bbb", ExpandedURL: "https://example.com/second"},
				{URL: "https://t.co/ccc", ExpandedURL: ""}, // must be left untouched
			},
		},
	}
	p, err := FromTweet(tw, "someone", time.UTC)
	if err != nil {
		t.Fatalf("FromTweet: %v", err)
	}
	want := "check https://example.com/first and https://example.com/second and https://t.co/ccc"
	if p.Text != want {
		t.Errorf("Text = %q, want %q", p.Text, want)
	}
}

func TestFromTweet_Repost(t *testing.T) {
	tests := []struct {
		name         string
		fullText     string
		wantText     string
		wantRepostOf string
	}{
		{
			name:         "well-formed repost",
			fullText:     "RT @foo: hello",
			wantText:     "hello",
			wantRepostOf: "@foo",
		},
		{
			name:         "malformed repost keeps full text",
			fullText:     "RT @foo hello",
			wantText:     "RT @foo hello",
			wantRepostOf: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tw := &archive.Tweet{
				IDStr:     "1",
				CreatedAt: "Fri Jul 10 15:56:55 +0000 2026",
				FullText:  tt.fullText,
			}
			p, err := FromTweet(tw, "someone", time.UTC)
			if err != nil {
				t.Fatalf("FromTweet: %v", err)
			}
			if p.Type != TypeRepost {
				t.Errorf("Type = %v, want %v", p.Type, TypeRepost)
			}
			if p.Text != tt.wantText {
				t.Errorf("Text = %q, want %q", p.Text, tt.wantText)
			}
			if p.RepostOf != tt.wantRepostOf {
				t.Errorf("RepostOf = %q, want %q", p.RepostOf, tt.wantRepostOf)
			}
		})
	}
}

func TestFromTweet_Reply(t *testing.T) {
	tests := []struct {
		name              string
		inReplyToStatusID string
		inReplyToScreen   string
		wantReplyTo       string
		wantReplyToURL    string
	}{
		{
			name:              "with screen name",
			inReplyToStatusID: "999",
			inReplyToScreen:   "friend",
			wantReplyTo:       "@friend",
			wantReplyToURL:    "https://twitter.com/friend/status/999",
		},
		{
			name:              "without screen name",
			inReplyToStatusID: "999",
			inReplyToScreen:   "",
			wantReplyTo:       "",
			wantReplyToURL:    "https://twitter.com/i/web/status/999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tw := &archive.Tweet{
				IDStr:                "1",
				CreatedAt:            "Fri Jul 10 15:56:55 +0000 2026",
				FullText:             "some reply text",
				InReplyToStatusIDStr: tt.inReplyToStatusID,
				InReplyToScreenName:  tt.inReplyToScreen,
			}
			p, err := FromTweet(tw, "someone", time.UTC)
			if err != nil {
				t.Fatalf("FromTweet: %v", err)
			}
			if p.Type != TypeReply {
				t.Errorf("Type = %v, want %v", p.Type, TypeReply)
			}
			if p.ReplyTo != tt.wantReplyTo {
				t.Errorf("ReplyTo = %q, want %q", p.ReplyTo, tt.wantReplyTo)
			}
			if p.ReplyToURL != tt.wantReplyToURL {
				t.Errorf("ReplyToURL = %q, want %q", p.ReplyToURL, tt.wantReplyToURL)
			}
		})
	}
}

func TestFromTweet_PlainPost(t *testing.T) {
	tw := &archive.Tweet{
		IDStr:     "42",
		CreatedAt: "Fri Jul 10 15:56:55 +0000 2026",
		FullText:  "just a normal post",
	}
	p, err := FromTweet(tw, "someone", time.UTC)
	if err != nil {
		t.Fatalf("FromTweet: %v", err)
	}
	if p.Type != TypePost {
		t.Errorf("Type = %v, want %v", p.Type, TypePost)
	}
	wantURL := "https://twitter.com/someone/status/42"
	if p.URL != wantURL {
		t.Errorf("URL = %q, want %q", p.URL, wantURL)
	}
}

func TestFromTweet_Counts(t *testing.T) {
	tests := []struct {
		name          string
		favoriteCount string
		retweetCount  string
		wantFavorite  int
		wantRetweet   int
	}{
		{name: "numeric strings", favoriteCount: "42", retweetCount: "7", wantFavorite: 42, wantRetweet: 7},
		{name: "empty strings default to 0", favoriteCount: "", retweetCount: "", wantFavorite: 0, wantRetweet: 0},
		{name: "unparseable strings default to 0", favoriteCount: "abc", retweetCount: "n/a", wantFavorite: 0, wantRetweet: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tw := &archive.Tweet{
				IDStr:         "1",
				CreatedAt:     "Fri Jul 10 15:56:55 +0000 2026",
				FullText:      "text",
				FavoriteCount: tt.favoriteCount,
				RetweetCount:  tt.retweetCount,
			}
			p, err := FromTweet(tw, "someone", time.UTC)
			if err != nil {
				t.Fatalf("FromTweet: %v", err)
			}
			if p.FavoriteCount != tt.wantFavorite {
				t.Errorf("FavoriteCount = %d, want %d", p.FavoriteCount, tt.wantFavorite)
			}
			if p.RetweetCount != tt.wantRetweet {
				t.Errorf("RetweetCount = %d, want %d", p.RetweetCount, tt.wantRetweet)
			}
		})
	}
}

func TestFromTweet_Lang(t *testing.T) {
	tw := &archive.Tweet{
		IDStr:     "1",
		CreatedAt: "Fri Jul 10 15:56:55 +0000 2026",
		FullText:  "text",
		Lang:      "ja",
	}
	p, err := FromTweet(tw, "someone", time.UTC)
	if err != nil {
		t.Fatalf("FromTweet: %v", err)
	}
	if p.Lang != "ja" {
		t.Errorf("Lang = %q, want %q", p.Lang, "ja")
	}
}

func TestApplyNoteTweets(t *testing.T) {
	sameInstant := time.Date(2026, time.July, 10, 15, 56, 55, 0, time.UTC)
	otherInstant := time.Date(2026, time.July, 10, 16, 0, 0, 0, time.UTC)

	truncatedPost := &Post{
		ID:        "1",
		Type:      TypePost,
		CreatedAt: sameInstant,
		Text:      "This is a very long post that got truncated by the archive export tool…",
	}
	repostAtSameInstant := &Post{
		ID:        "2",
		Type:      TypeRepost,
		CreatedAt: sameInstant,
		Text:      "RT text that also got truncated…",
	}
	untouchedPost := &Post{
		ID:        "3",
		Type:      TypePost,
		CreatedAt: otherInstant,
		Text:      "a short post",
	}

	posts := []*Post{truncatedPost, repostAtSameInstant, untouchedPost}

	notes := []archive.NoteTweet{
		{
			NoteTweetID: "n1",
			CreatedAt:   sameInstant.Format(time.RFC3339),
			Text:        "This is a very long post that got truncated by the archive export tool but here is the rest of it, in full.",
		},
	}

	ApplyNoteTweets(posts, notes)

	wantText := "This is a very long post that got truncated by the archive export tool but here is the rest of it, in full."
	if truncatedPost.Text != wantText {
		t.Errorf("truncatedPost.Text = %q, want %q", truncatedPost.Text, wantText)
	}
	if repostAtSameInstant.Text != "RT text that also got truncated…" {
		t.Errorf("repostAtSameInstant.Text was modified: %q", repostAtSameInstant.Text)
	}
	if untouchedPost.Text != "a short post" {
		t.Errorf("untouchedPost.Text was modified: %q", untouchedPost.Text)
	}
}

func TestApplyNoteTweets_EmptyNotes(t *testing.T) {
	p := &Post{
		ID:        "1",
		Type:      TypePost,
		CreatedAt: time.Now(),
		Text:      "unchanged…",
	}
	posts := []*Post{p}
	ApplyNoteTweets(posts, nil)
	if p.Text != "unchanged…" {
		t.Errorf("Text was modified with an empty notes slice: %q", p.Text)
	}
}

func TestApplyNoteTweets_InvalidCreatedAtSkipped(t *testing.T) {
	p := &Post{
		ID:        "1",
		Type:      TypePost,
		CreatedAt: time.Date(2026, time.July, 10, 15, 56, 55, 0, time.UTC),
		Text:      "unchanged…",
	}
	notes := []archive.NoteTweet{
		{NoteTweetID: "n1", CreatedAt: "not-a-valid-timestamp", Text: "full text"},
	}
	ApplyNoteTweets([]*Post{p}, notes)
	if p.Text != "unchanged…" {
		t.Errorf("Text was modified despite an unparseable note created_at: %q", p.Text)
	}
}

func TestFromTweet_HTMLUnescape(t *testing.T) {
	tw := &archive.Tweet{
		IDStr:     "1",
		CreatedAt: "Fri Jul 03 15:56:55 +0000 2026",
		FullText:  "Tom &amp; Jerry &lt;3 &gt; &amp;lt; https://t.co/a",
		Entities:  archive.Entities{URLs: []archive.URLEntity{{URL: "https://t.co/a", ExpandedURL: "https://e.com/?a=1&b=2"}}},
	}
	p, err := FromTweet(tw, "u", time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	want := "Tom & Jerry <3 > &lt; https://e.com/?a=1&b=2"
	if p.Text != want {
		t.Errorf("Text = %q, want %q", p.Text, want)
	}
}
