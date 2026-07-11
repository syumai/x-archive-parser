package archive

// Tweet is the subset of a raw archive tweet object that this tool uses.
// The archive stores numeric counts as strings (e.g. "favorite_count": "0").
type Tweet struct {
	IDStr                string   `json:"id_str"`
	CreatedAt            string   `json:"created_at"` // e.g. "Fri Jul 10 15:56:55 +0000 2026"
	FullText             string   `json:"full_text"`
	Lang                 string   `json:"lang"`
	FavoriteCount        string   `json:"favorite_count"`
	RetweetCount         string   `json:"retweet_count"`
	InReplyToStatusIDStr string   `json:"in_reply_to_status_id_str"`
	InReplyToScreenName  string   `json:"in_reply_to_screen_name"`
	Entities             Entities `json:"entities"`
}

// Entities holds the entity lists attached to a tweet.
type Entities struct {
	URLs []URLEntity `json:"urls"`
}

// URLEntity maps a t.co short URL in the tweet text to its expanded form.
type URLEntity struct {
	URL         string `json:"url"`
	ExpandedURL string `json:"expanded_url"`
}

// Account is the account information stored in data/account.js.
type Account struct {
	Username           string `json:"username"`
	AccountID          string `json:"accountId"`
	AccountDisplayName string `json:"accountDisplayName"`
	CreatedAt          string `json:"createdAt"` // RFC 3339, e.g. "2013-08-26T20:14:03.579Z"
}

// NoteTweet is a long-form post stored in data/note-tweet.js.
// Note tweets carry their own ID (not the tweet's id_str), so they are
// matched back to tweets by creation time and text prefix, best-effort.
type NoteTweet struct {
	NoteTweetID string `json:"noteTweetId"`
	CreatedAt   string `json:"createdAt"` // RFC 3339
	Text        string // core.text
}
