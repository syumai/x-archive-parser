package archive

import (
	"errors"
	"fmt"
	"io/fs"
)

// NoteTweets returns every long-form "Note Tweet" stored in
// data/note-tweet.js. If the archive has no note tweets — either because
// the file is missing or its array is empty — it returns an empty slice
// and a nil error.
func (a *Archive) NoteTweets() ([]NoteTweet, error) {
	dec, closeFn, err := openJSArray(a.fsys, "data/note-tweet.js")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("archive: reading note tweets: %w", err)
	}
	defer closeFn()

	if _, err := dec.Token(); err != nil { // consume the opening '['
		return nil, fmt.Errorf("archive: reading data/note-tweet.js: %w", err)
	}

	var noteTweets []NoteTweet
	for dec.More() {
		var entry struct {
			NoteTweet struct {
				NoteTweetID string `json:"noteTweetId"`
				CreatedAt   string `json:"createdAt"`
				Core        struct {
					Text string `json:"text"`
				} `json:"core"`
			} `json:"noteTweet"`
		}
		if err := dec.Decode(&entry); err != nil {
			return nil, fmt.Errorf("archive: decoding data/note-tweet.js: %w", err)
		}
		noteTweets = append(noteTweets, NoteTweet{
			NoteTweetID: entry.NoteTweet.NoteTweetID,
			CreatedAt:   entry.NoteTweet.CreatedAt,
			Text:        entry.NoteTweet.Core.Text,
		})
	}

	return noteTweets, nil
}
