// Package fsx holds the one file-system rule every state file follows: replace atomically,
// and survive Windows refusing a rename while another reader has the target open.
package fsx

import (
	"os"
	"time"
)

// Retry budget for Replace. A reader (the UI refreshing the board) holds a file for a few
// milliseconds; antivirus scans can take longer.
const (
	replaceTries = 25
	replacePause = 20 * time.Millisecond
)

// Replace renames tmp over dst. On Windows a rename onto a file that is open elsewhere fails
// with "Access is denied"; Replace waits and retries for about half a second before giving up.
func Replace(tmp, dst string) error {
	var err error
	for i := 0; i < replaceTries; i++ {
		if err = os.Rename(tmp, dst); err == nil {
			return nil
		}
		time.Sleep(replacePause)
	}
	return err
}
