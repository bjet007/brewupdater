// Command brewupdater lists outdated Homebrew packages, favorites first, and upgrades the ones you pick.
package main

import (
	"os"

	"github.com/bjet007/brewupdater/internal/cmd/brewupdater"
)

func main() {
	os.Exit(brewupdater.Execute())
}
