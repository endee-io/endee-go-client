// Package main demonstrates basic usage of the endee-go-client SDK.
package main

import (
	"fmt"
	"log"

	endee "github.com/endee-io/endee-go-client"
)

func main() {
	// Connect to a local Endee instance (no auth required).
	client := endee.NewClient()

	// Connect to a cloud instance:
	// client := endee.NewClient(endee.WithToken("prefix:secret:region"))

	// Create an index.
	err := client.CreateIndex(
		"my_index",
		384,
		endee.Cosine,
		endee.DefaultM,
		endee.DefaultEfConstruction,
		endee.PrecisionInt16,
		nil,
		0,
	)
	if err != nil {
		log.Fatalf("CreateIndex: %v", err)
	}
	fmt.Println("Index created.")

	// Get a handle to the index.
	index, err := client.GetIndex("my_index")
	if err != nil {
		log.Fatalf("GetIndex: %v", err)
	}

	// Upsert vectors.
	vectors := []endee.VectorItem{
		{
			ID:     "doc-1",
			Vector: make([]float32, 384), // replace with real embeddings
			Meta:   map[string]interface{}{"title": "Example document"},
			Filter: map[string]interface{}{"category": "example"},
		},
	}
	if err := index.Upsert(vectors); err != nil {
		log.Fatalf("Upsert: %v", err)
	}
	fmt.Println("Vectors upserted.")

	// Query for similar vectors.
	queryVec := make([]float32, 384) // replace with real query embedding
	results, err := index.Query(queryVec, nil, nil, 10, nil, endee.DefaultEfSearch, false, nil)
	if err != nil {
		log.Fatalf("Query: %v", err)
	}

	for _, r := range results {
		fmt.Printf("ID: %s  Similarity: %.4f\n", r.ID, r.Similarity)
	}
}
