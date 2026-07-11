package archive

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// collectTweets drains a.Tweets(), failing the test on the first error.
func collectTweets(t *testing.T, a *Archive) []Tweet {
	t.Helper()

	var tweets []Tweet
	for tw, err := range a.Tweets() {
		if err != nil {
			t.Fatalf("Tweets: %v", err)
		}
		tweets = append(tweets, *tw)
	}
	return tweets
}

// buildZip zips the contents of srcDir into a new zip file under t.TempDir()
// and returns its path. Each entry's path is optionally nested under
// prefix, simulating a zip whose contents sit inside a single wrapper
// directory (as happens when a previously extracted archive is re-zipped).
func buildZip(t *testing.T, srcDir, prefix string) string {
	t.Helper()

	zipPath := filepath.Join(t.TempDir(), "archive.zip")
	zf, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("creating zip file: %v", err)
	}
	defer zf.Close()

	zw := zip.NewWriter(zf)

	err = filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(rel)
		if prefix != "" {
			name = prefix + "/" + name
		}

		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()

		dst, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = io.Copy(dst, src)
		return err
	})
	if err != nil {
		t.Fatalf("walking %s: %v", srcDir, err)
	}

	if err := zw.Close(); err != nil {
		t.Fatalf("closing zip writer: %v", err)
	}
	if err := zf.Close(); err != nil {
		t.Fatalf("closing zip file: %v", err)
	}

	return zipPath
}

func TestOpen_Directory(t *testing.T) {
	a, err := Open("testdata/simple")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer a.Close()

	account, err := a.Account()
	if err != nil {
		t.Fatalf("Account: %v", err)
	}
	if account.Username != "testuser" {
		t.Errorf("Account().Username = %q, want %q", account.Username, "testuser")
	}
	if account.AccountID != "1000000000" {
		t.Errorf("Account().AccountID = %q, want %q", account.AccountID, "1000000000")
	}

	tweets := collectTweets(t, a)
	if len(tweets) != 4 {
		t.Fatalf("got %d tweets, want 4", len(tweets))
	}
	if tweets[0].IDStr != "1000000000000000001" {
		t.Errorf("tweets[0].IDStr = %q, want %q", tweets[0].IDStr, "1000000000000000001")
	}
	if len(tweets[0].Entities.URLs) != 1 {
		t.Fatalf("tweets[0] has %d URL entities, want 1", len(tweets[0].Entities.URLs))
	}
	if got, want := tweets[0].Entities.URLs[0].ExpandedURL, "https://example.com/article"; got != want {
		t.Errorf("tweets[0] expanded URL = %q, want %q", got, want)
	}
	if got, want := tweets[2].InReplyToStatusIDStr, "1000000000000000001"; got != want {
		t.Errorf("tweets[2].InReplyToStatusIDStr = %q, want %q", got, want)
	}
	if got, want := tweets[2].InReplyToScreenName, "friend"; got != want {
		t.Errorf("tweets[2].InReplyToScreenName = %q, want %q", got, want)
	}

	noteTweets, err := a.NoteTweets()
	if err != nil {
		t.Fatalf("NoteTweets: %v", err)
	}
	if len(noteTweets) != 1 {
		t.Fatalf("got %d note tweets, want 1", len(noteTweets))
	}
	if noteTweets[0].NoteTweetID != "n_1000000000000000004" {
		t.Errorf("noteTweets[0].NoteTweetID = %q, want %q", noteTweets[0].NoteTweetID, "n_1000000000000000004")
	}
	if !strings.Contains(noteTweets[0].Text, "full long-form text") {
		t.Errorf("noteTweets[0].Text = %q, missing expected content", noteTweets[0].Text)
	}
}

func TestOpen_Zip(t *testing.T) {
	dirArchive, err := Open("testdata/simple")
	if err != nil {
		t.Fatalf("Open(directory): %v", err)
	}
	defer dirArchive.Close()
	dirTweets := collectTweets(t, dirArchive)

	zipPath := buildZip(t, "testdata/simple", "")
	zipArchive, err := Open(zipPath)
	if err != nil {
		t.Fatalf("Open(zip): %v", err)
	}
	defer zipArchive.Close()
	zipTweets := collectTweets(t, zipArchive)

	if !reflect.DeepEqual(dirTweets, zipTweets) {
		t.Fatalf("zip tweets differ from directory tweets:\ndir:  %+v\nzip:  %+v", dirTweets, zipTweets)
	}

	dirAccount, err := dirArchive.Account()
	if err != nil {
		t.Fatalf("Account (directory): %v", err)
	}
	zipAccount, err := zipArchive.Account()
	if err != nil {
		t.Fatalf("Account (zip): %v", err)
	}
	if !reflect.DeepEqual(dirAccount, zipAccount) {
		t.Fatalf("zip account differs from directory account: dir=%+v zip=%+v", dirAccount, zipAccount)
	}
}

func TestOpen_ZipWithWrapperDir(t *testing.T) {
	zipPath := buildZip(t, "testdata/simple", "twitter-2023-01-01-testarchive")

	a, err := Open(zipPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer a.Close()

	account, err := a.Account()
	if err != nil {
		t.Fatalf("Account: %v", err)
	}
	if account.Username != "testuser" {
		t.Errorf("Account().Username = %q, want %q", account.Username, "testuser")
	}

	tweets := collectTweets(t, a)
	if len(tweets) != 4 {
		t.Fatalf("got %d tweets, want 4", len(tweets))
	}
}

func TestOpen_NotAnArchive(t *testing.T) {
	dir := t.TempDir()
	if _, err := Open(dir); err == nil {
		t.Fatal("Open on a directory without data/account.js: got nil error, want an error")
	} else if !strings.Contains(err.Error(), "not an X archive") {
		t.Errorf("Open error = %v, want it to mention %q", err, "not an X archive")
	}
}

func TestTweets_MultiPart(t *testing.T) {
	a, err := Open("testdata/multipart")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer a.Close()

	tweets := collectTweets(t, a)
	if len(tweets) != 4 {
		t.Fatalf("got %d tweets, want 4", len(tweets))
	}

	var ids []string
	for _, tw := range tweets {
		ids = append(ids, tw.IDStr)
	}
	sort.Strings(ids)
	want := []string{
		"2000000000000000001",
		"2000000000000000002",
		"2000000000000000003",
		"2000000000000000004",
	}
	if !reflect.DeepEqual(ids, want) {
		t.Fatalf("tweet IDs = %v, want %v", ids, want)
	}
}

func TestNoteTweets_Empty(t *testing.T) {
	a, err := Open("testdata/multipart")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer a.Close()

	noteTweets, err := a.NoteTweets()
	if err != nil {
		t.Fatalf("NoteTweets: %v", err)
	}
	if len(noteTweets) != 0 {
		t.Fatalf("got %d note tweets, want 0", len(noteTweets))
	}
}

func TestNoteTweets_FileMissing(t *testing.T) {
	// testdata/simple has no data/tweet-headers or similar quirks, but to
	// exercise the "file missing entirely" path we build a minimal archive
	// without data/note-tweet.js at all.
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	accountJS := "window.YTD.account.part0 = [\n  {\"account\": {\"username\": \"nouser\", \"accountId\": \"1\", \"accountDisplayName\": \"No User\", \"createdAt\": \"2020-01-01T00:00:00.000Z\"}}\n]\n"
	if err := os.WriteFile(filepath.Join(dataDir, "account.js"), []byte(accountJS), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	a, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer a.Close()

	noteTweets, err := a.NoteTweets()
	if err != nil {
		t.Fatalf("NoteTweets: %v", err)
	}
	if len(noteTweets) != 0 {
		t.Fatalf("got %d note tweets, want 0", len(noteTweets))
	}
}
