package archive

import "fmt"

// Account returns the account information stored in data/account.js.
func (a *Archive) Account() (*Account, error) {
	dec, closeFn, err := openJSArray(a.fsys, "data/account.js")
	if err != nil {
		return nil, fmt.Errorf("archive: reading account: %w", err)
	}
	defer closeFn()

	if _, err := dec.Token(); err != nil { // consume the opening '['
		return nil, fmt.Errorf("archive: reading data/account.js: %w", err)
	}

	if !dec.More() {
		return nil, fmt.Errorf("archive: data/account.js contains no account entry")
	}

	var entry struct {
		Account Account `json:"account"`
	}
	if err := dec.Decode(&entry); err != nil {
		return nil, fmt.Errorf("archive: decoding data/account.js: %w", err)
	}

	return &entry.Account, nil
}
