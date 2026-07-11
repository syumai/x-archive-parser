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
	for _, p := range posts {
		b, err := json.Marshal(p)
		if err != nil {
			return fmt.Errorf("marshaling post %s: %w", p.ID, err)
		}
		if _, err := w.Write(b); err != nil {
			return fmt.Errorf("writing %q: %w", path, err)
		}
		if err := w.WriteByte('\n'); err != nil {
			return fmt.Errorf("writing %q: %w", path, err)
		}
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("writing %q: %w", path, err)
	}
	return nil
}
