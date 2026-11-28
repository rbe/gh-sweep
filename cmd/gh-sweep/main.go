// Command gh-sweep deletes GitHub Actions artifacts and caches. The
// package internal/cli holds the command line.
package main

import (
	"os"

	"github.com/rbe/gh-sweep/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
