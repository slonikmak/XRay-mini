// Integration-only adapter: the shipped config builder, without process/service API.
package main

import (
	"fmt"
	"github.com/slonikmak/XRay-mini/internal/core"
	"io"
	"os"
)

func main() {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}
	output, err := core.PrepareConfig(input)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Stdout.Write(output)
}
