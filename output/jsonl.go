package output

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/syumai/x-archive-parser/post"
)

// writeMonthJSONL renders the JSONL file for a single month: one
// json.Marshal'ed post per line, in the given (ascending) order.
func writeMonthJSONL(path string, posts []*post.Post) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating %q: %w", path, err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	// Encoder.Encode appends the newline. HTML escaping is disabled so that
	// "&", "<" and ">" stay literal and remain greppable.
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	for _, p := range posts {
		if err := enc.Encode(p); err != nil {
			return fmt.Errorf("writing post %s to %q: %w", p.ID, path, err)
		}
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("writing %q: %w", path, err)
	}
	return nil
}
