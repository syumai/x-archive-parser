package archive

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"strings"
)

// jsPrefixScanLimit bounds how many bytes are scanned for the opening '['
// of a "window.YTD.<name>.partN = [...]" data file. The prefix is always on
// the first line and only a few dozen bytes long; this limit just guards
// against reading a malformed file into memory.
const jsPrefixScanLimit = 4096

// openJSArray opens the archive data file at name (relative to fsys) and
// returns a JSON decoder positioned to stream-decode the JSON array it
// contains, after skipping the "window.YTD.<name>.partN = " prefix used by
// X archive .js files. The file is read incrementally, not loaded into
// memory in full.
//
// The caller must call the returned close function when done.
func openJSArray(fsys fs.FS, name string) (dec *json.Decoder, closeFn func() error, err error) {
	f, err := fsys.Open(name)
	if err != nil {
		return nil, nil, err
	}

	br := bufio.NewReader(f)
	var scanned []byte
	for {
		b, err := br.ReadByte()
		if err != nil {
			f.Close()
			return nil, nil, fmt.Errorf("archive: reading %s: no JSON array found: %w", name, err)
		}
		scanned = append(scanned, b)
		if b == '[' {
			break
		}
		if len(scanned) > jsPrefixScanLimit {
			f.Close()
			return nil, nil, fmt.Errorf("archive: reading %s: no '[' found within first %d bytes", name, jsPrefixScanLimit)
		}
	}

	r := io.MultiReader(strings.NewReader("["), br)
	return json.NewDecoder(r), f.Close, nil
}
