package main

import (
	"fmt"
	"os"

	"modbridge/pkg/openapi"
)

func main() {
	spec := openapi.Generate()
	b, err := spec.ToJSON()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to generate OpenAPI spec: %v\n", err)
		os.Exit(1)
	}

	if _, err := os.Stdout.Write(b); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write OpenAPI spec: %v\n", err)
		os.Exit(1)
	}
}
