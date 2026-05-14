package main

import (
	"fmt"
	"os"

	"reverb/internal/app"
)

var Version = "dev"

func main() {
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
