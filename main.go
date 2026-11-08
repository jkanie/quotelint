// Command quotelint checks every quoted passage in a document against
// its source files and lists each quote that does not match verbatim.
package main

import (
	"os"

	"github.com/jkanie/quotelint/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
