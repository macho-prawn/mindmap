package main

import (
	"context"
	"fmt"
	"os"

	"mindmap/internal/app"
	"mindmap/internal/provider"
)

func main() {
	discovery, err := provider.NewComputeProvider(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	cli, err := app.New(
		app.RealFileStore{},
		discovery,
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := cli.Run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
