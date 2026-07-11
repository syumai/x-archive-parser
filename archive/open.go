package archive

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
)

// Archive provides access to the contents of an X (Twitter) archive, which
// may be a zip file downloaded from X or a directory that such a zip was
// extracted to.
type Archive struct {
	fsys   fs.FS
	closer io.Closer
}

// Open opens the X archive located at path. If path is a directory, its
// contents are read directly; if path is a file, it is opened as a zip
// archive.
//
// The archive root (the directory containing data/account.js) is detected
// automatically: it may be the root of fsys, or nested one level down (for
// example when a previously extracted archive directory was zipped up
// again, producing a single wrapper directory inside the zip).
func Open(path string) (*Archive, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("archive: %w", err)
	}

	var fsys fs.FS
	var closer io.Closer
	if info.IsDir() {
		fsys = os.DirFS(path)
	} else {
		rc, err := zip.OpenReader(path)
		if err != nil {
			return nil, fmt.Errorf("archive: opening zip %s: %w", path, err)
		}
		fsys = rc
		closer = rc
	}

	root, err := findRoot(fsys, path)
	if err != nil {
		if closer != nil {
			closer.Close()
		}
		return nil, err
	}

	return &Archive{fsys: root, closer: closer}, nil
}

// findRoot locates the fs.FS subtree that contains data/account.js, the
// anchor file used to recognize an X archive. origin is only used to
// produce a helpful error message.
func findRoot(fsys fs.FS, origin string) (fs.FS, error) {
	if _, err := fs.Stat(fsys, "data/account.js"); err == nil {
		return fsys, nil
	}

	matches, err := fs.Glob(fsys, "*/data/account.js")
	if err != nil {
		return nil, fmt.Errorf("archive: searching for account.js: %w", err)
	}
	if len(matches) > 0 {
		root := strings.TrimSuffix(matches[0], "/data/account.js")
		sub, err := fs.Sub(fsys, root)
		if err != nil {
			return nil, fmt.Errorf("archive: rooting archive at %s: %w", root, err)
		}
		return sub, nil
	}

	return nil, fmt.Errorf("account.js not found under %s: not an X archive?", origin)
}

// Close releases any resources held by the archive (such as an open zip
// file). It is a no-op if the archive was opened from a directory.
func (a *Archive) Close() error {
	if a.closer == nil {
		return nil
	}
	return a.closer.Close()
}
