package main

import (
	"fmt"
	"os"

	"lazyconfigs/internal/app"
	"lazyconfigs/internal/version"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Println("lazyconfigs " + version.Version)
		return
	}

	a := app.New()
	if err := a.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
