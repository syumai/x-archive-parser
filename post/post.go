// Package post defines the normalized post model produced from raw archive
// tweets, independent of the archive's on-disk representation.
package post

import "time"

// Type classifies a post.
type Type string

const (
	TypePost   Type = "post"   // an original post
	TypeRepost Type = "repost" // a retweet ("RT @user: ..." in the archive)
	TypeReply  Type = "reply"  // a reply to another post
)

// Post is a normalized post ready for output.
type Post struct {
	ID            string    `json:"id"`
	URL           string    `json:"url"`        // https://twitter.com/<username>/status/<id>
	CreatedAt     time.Time `json:"created_at"` // converted to the selected timezone
	Type          Type      `json:"type"`
	Text          string    `json:"text"`                // t.co URLs expanded; repost prefix removed
	RepostOf      string    `json:"repost_of,omitempty"` // "@handle" of the original author, for reposts
	ReplyTo       string    `json:"reply_to,omitempty"`  // "@handle" being replied to, for replies
	ReplyToURL    string    `json:"reply_to_url,omitempty"`
	Lang          string    `json:"lang,omitempty"`
	FavoriteCount int       `json:"favorite_count"`
	RetweetCount  int       `json:"retweet_count"`
}
