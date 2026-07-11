package archive

import (
	"fmt"
	"io/fs"
	"iter"
	"sort"
)

// tweetFilePatterns lists the glob patterns (relative to the archive root)
// that may contain tweet entries. Recent archives store everything in
// data/tweets.js, but older archives may use the singular data/tweet.js and
// may split large exports into numbered parts.
var tweetFilePatterns = []string{
	"data/tweets.js",
	"data/tweet.js",
	"data/tweets-part*.js",
	"data/tweet-part*.js",
}

// tweetFiles returns the deduplicated, lexically sorted list of data files
// in the archive that contain tweet entries.
func (a *Archive) tweetFiles() ([]string, error) {
	seen := make(map[string]bool)
	var files []string
	for _, pattern := range tweetFilePatterns {
		matches, err := fs.Glob(a.fsys, pattern)
		if err != nil {
			return nil, fmt.Errorf("archive: matching %s: %w", pattern, err)
		}
		for _, m := range matches {
			if !seen[m] {
				seen[m] = true
				files = append(files, m)
			}
		}
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("archive: no tweet data file found (expected data/tweets.js or similar)")
	}
	sort.Strings(files)
	return files, nil
}

// Tweets streams every tweet stored in the archive, across all tweet data
// files (see tweetFilePatterns), in lexical file order and array order
// within each file. Iteration stops after the first error, which is
// delivered as the final (nil, err) pair.
func (a *Archive) Tweets() iter.Seq2[*Tweet, error] {
	return func(yield func(*Tweet, error) bool) {
		files, err := a.tweetFiles()
		if err != nil {
			yield(nil, err)
			return
		}

		for _, name := range files {
			if !a.yieldTweetsFromFile(name, yield) {
				return
			}
		}
	}
}

// yieldTweetsFromFile streams the tweets in a single data file, calling
// yield for each one. It returns false if iteration should stop, matching
// the return value of yield.
func (a *Archive) yieldTweetsFromFile(name string, yield func(*Tweet, error) bool) bool {
	dec, closeFn, err := openJSArray(a.fsys, name)
	if err != nil {
		return yield(nil, fmt.Errorf("archive: reading %s: %w", name, err))
	}
	defer closeFn()

	if _, err := dec.Token(); err != nil { // consume the opening '['
		return yield(nil, fmt.Errorf("archive: reading %s: %w", name, err))
	}

	for dec.More() {
		var entry struct {
			Tweet Tweet `json:"tweet"`
		}
		if err := dec.Decode(&entry); err != nil {
			return yield(nil, fmt.Errorf("archive: decoding tweet in %s: %w", name, err))
		}
		if !yield(&entry.Tweet, nil) {
			return false
		}
	}
	return true
}
