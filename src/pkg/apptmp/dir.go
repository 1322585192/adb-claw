package apptmp

import (
	"os"
	"strings"
)

// Root is the host temp directory. Tests set TMPDIR; Windows os.TempDir
// otherwise ignores that variable and would leak fixtures into %TEMP%.
func Root() string {
	if dir := strings.TrimSpace(os.Getenv("TMPDIR")); dir != "" {
		return dir
	}
	return os.TempDir()
}
