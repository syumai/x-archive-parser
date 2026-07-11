package archive

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// TestIntegration_RealArchive exercises the archive package against a real
// X archive directory, if one is present at ../example/twitter-*. This
// directory contains personal data and is never committed (see
// .gitignore), so this test is skipped unless it exists locally.
//
// Two optional environment variables let a caller assert exact values
// against their own archive without hardcoding personal data in this file:
//
//	X_ARCHIVE_TEST_USERNAME    expected Account().Username
//	X_ARCHIVE_TEST_TWEET_COUNT expected number of tweets (integer)
//
// When unset, the test only checks that Account() returns a non-empty
// username and that Tweets() yields more than zero tweets with no error.
func TestIntegration_RealArchive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	matches, err := filepath.Glob("../example/twitter-*")
	if err != nil {
		t.Fatalf("Glob: %v", err)
	}
	if len(matches) == 0 {
		t.Skip("no ../example/twitter-* archive found; skipping integration test")
	}

	a, err := Open(matches[0])
	if err != nil {
		t.Fatalf("Open(%s): %v", matches[0], err)
	}
	defer a.Close()

	account, err := a.Account()
	if err != nil {
		t.Fatalf("Account: %v", err)
	}

	wantUsername := os.Getenv("X_ARCHIVE_TEST_USERNAME")
	if wantUsername != "" {
		if account.Username != wantUsername {
			t.Errorf("Account().Username = %q, want %q", account.Username, wantUsername)
		}
	} else if account.Username == "" {
		t.Error("Account().Username is empty, want non-empty")
	}

	count := 0
	for _, err := range a.Tweets() {
		if err != nil {
			t.Fatalf("Tweets: %v", err)
		}
		count++
	}

	if wantCountStr := os.Getenv("X_ARCHIVE_TEST_TWEET_COUNT"); wantCountStr != "" {
		wantCount, err := strconv.Atoi(wantCountStr)
		if err != nil {
			t.Fatalf("parsing X_ARCHIVE_TEST_TWEET_COUNT=%q: %v", wantCountStr, err)
		}
		if count != wantCount {
			t.Errorf("got %d tweets, want %d", count, wantCount)
		}
	} else if count == 0 {
		t.Error("got 0 tweets, want more than 0")
	}
}
