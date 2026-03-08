// Package endee provides a Go client for the Endee vector database.
//
// Endee is a high-performance vector database built for approximate nearest
// neighbor (ANN) search using the HNSW algorithm. This client supports dense
// vector search, hybrid (dense + sparse) search, metadata filtering, and
// multiple precision levels.
//
// # Connecting
//
// For a local instance (no auth):
//
//	client := endee.NewClient()
//
// For a cloud instance:
//
//	client := endee.NewClient(endee.WithToken("prefix:secret:region"))
//
// # Creating an Index
//
//	err := client.CreateIndex("my_index", 384, endee.Cosine,
//	    endee.DefaultM, endee.DefaultEfConstruction,
//	    endee.PrecisionInt16, nil, 0)
//
// # Upserting Vectors
//
//	index, _ := client.GetIndex("my_index")
//	index.Upsert([]endee.VectorItem{
//	    {ID: "doc-1", Vector: embedding, Meta: map[string]interface{}{"title": "Hello"}},
//	})
//
// # Querying
//
//	results, _ := index.Query(queryVec, nil, nil, 10, nil, endee.DefaultEfSearch, false, nil)
//	for _, r := range results {
//	    fmt.Printf("%s: %.4f\n", r.ID, r.Similarity)
//	}
package endee
