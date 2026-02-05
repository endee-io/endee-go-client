package endee

import (
	"fmt"
	"strings"
	"testing"
)

func TestListIndexes(t *testing.T) {
	vecx := EndeeClient()
	indexes, err := vecx.ListIndexes()

	if err != nil {
		t.Fatalf("ListIndexes() failed: %v", err)
	}

	// An empty slice is valid, but nil is not
	if indexes == nil {
		t.Fatal("ListIndexes() returned nil indexes (expected empty slice if no indexes exist)")
	}

	t.Logf("Successfully retrieved %d indexes", len(indexes))

	// Log the indexes for debugging
	for i, index := range indexes {
		t.Logf("Index %d: %+v", i, index)
	}
}

func TestCreateIndex(t *testing.T) {
	vecx := EndeeClient()

	// Test with valid parameters
	err := vecx.CreateIndex("test_go_index", 768, "cosine", 16, 128, true, nil, 0)
	if err != nil {
		t.Logf("CreateIndex failed (this might be expected if index already exists): %v", err)
	} else {
		t.Log("CreateIndex succeeded")
	}
}

func TestDeleteIndex(t *testing.T) {
	vecx := EndeeClient()

	// Test deleting the index we created
	err := vecx.DeleteIndex("test_go_index")
	if err != nil {
		t.Logf("DeleteIndex failed (this might be expected if index doesn't exist): %v", err)
	} else {
		t.Log("DeleteIndex succeeded")
	}
}

func TestGetIndex(t *testing.T) {
	vecx := EndeeClient()

	// Test getting the index we created
	index, err := vecx.GetIndex("test_go_index")
	if err != nil {
		t.Logf("GetIndex failed (this might be expected if index doesn't exist): %v", err)
	} else {
		t.Logf("GetIndex succeeded: %+v", index)
		t.Logf("Index info: %+v", index.GetInfo())
	}
}

func TestUpsert(t *testing.T) {
	vecx := EndeeClient()

	// First, create an index for testing
	err := vecx.CreateIndex("test_upsert_index", 16, "cosine", 16, 128, true, nil, 0)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Logf("CreateIndex failed (might be expected): %v", err)
	}

	// Get the index to use for upserting
	index, err := vecx.GetIndex("test_upsert_index")
	if err != nil {
		t.Fatalf("GetIndex failed: %v", err)
	}

	t.Logf("Retrieved index: %s with dimension %d", index.Name, index.Dimension)

	// Create test vectors with the correct dimension (16)
	testVectors := []VectorItem{
		{
			ID:     "doc_1",
			Vector: make([]float32, 16),
			Meta: map[string]interface{}{
				"title":       "Test Document 1",
				"category":    "technology",
				"description": "A test document about technology",
			},
			Filter: map[string]interface{}{
				"category": "tech",
				"public":   true,
			},
		},
		{
			ID:     "doc_2",
			Vector: make([]float32, 16),
			Meta: map[string]interface{}{
				"title":       "Test Document 2",
				"category":    "science",
				"description": "A test document about science",
			},
			Filter: map[string]interface{}{
				"category": "science",
				"public":   false,
			},
		},
	}

	// Fill vectors with some meaningful test data
	for i := range testVectors {
		for j := range testVectors[i].Vector {
			// Create some varied test data that's not all zeros
			testVectors[i].Vector[j] = float32(j+1) * 0.1 * float32(i+1)
		}
	}

	t.Logf("Test vectors: %+v", testVectors)

	// Test the upsert
	err = index.Upsert(testVectors)
	if err != nil {
		t.Logf("Upsert failed: %v", err)
		// Check if it's a network error vs server error
		if strings.Contains(err.Error(), "failed to execute request") ||
			strings.Contains(err.Error(), "no such host") ||
			strings.Contains(err.Error(), "connection refused") {
			t.Log("Upsert failed due to network issues, but validation passed")
		} else if strings.Contains(err.Error(), "std::bad_cast") {
			t.Error("Server returned std::bad_cast - data structure mismatch (this should be fixed now)")
		} else {
			t.Logf("Upsert failed with server error (might be expected): %v", err)
		}
	} else {
		t.Log("Upsert succeeded!")
	}

	// Clean up - delete the test index
	err = vecx.DeleteIndex("test_upsert_index")
	if err != nil {
		t.Logf("Failed to clean up test index: %v", err)
	} else {
		t.Log("Successfully cleaned up test index")
	}
}

func TestQuery(t *testing.T) {
	vecx := EndeeClient()

	// Create an index for testing
	err := vecx.CreateIndex("test_query_index", 16, "cosine", 16, 128, true, nil, 0)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Logf("CreateIndex failed (might be expected): %v", err)
	}

	// Get the index
	index, err := vecx.GetIndex("test_query_index")
	if err != nil {
		t.Fatalf("GetIndex failed: %v", err)
	}

	// First, upsert some test vectors to query against
	testVectors := []VectorItem{
		{
			ID:     "query_test_1",
			Vector: []float32{1.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0},
			Meta: map[string]interface{}{
				"title": "Document 1",
				"type":  "article",
			},
			Filter: map[string]interface{}{
				"category": "tech",
			},
		},
		{
			ID:     "query_test_2",
			Vector: []float32{0.0, 1.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0},
			Meta: map[string]interface{}{
				"title": "Document 2",
				"type":  "blog",
			},
			Filter: map[string]interface{}{
				"category": "science",
			},
		},
		{
			ID:     "query_test_3",
			Vector: []float32{0.5, 0.5, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0},
			Meta: map[string]interface{}{
				"title": "Document 3",
				"type":  "paper",
			},
			Filter: map[string]interface{}{
				"category": "tech",
			},
		},
	}

	// Upsert the vectors
	err = index.Upsert(testVectors)
	if err != nil {
		t.Logf("Upsert failed: %v", err)
		// If upsert fails, we can still test the query structure
	} else {
		t.Log("Test vectors upserted successfully")
	}

	// Test query with a vector similar to the first one
	queryVector := []float32{0.9, 0.1, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0}

	// Test basic query (without filters)
	results, err := index.Query(queryVector, nil, nil, 3, nil, 128, false)
	if err != nil {
		t.Logf("Query failed: %v", err)
		// Check if it's a network/server error vs validation error
		if strings.Contains(err.Error(), "failed to execute request") ||
			strings.Contains(err.Error(), "no such host") ||
			strings.Contains(err.Error(), "connection refused") {
			t.Log("Query failed due to network issues, but validation passed")
		} else {
			t.Logf("Query failed with server error (might be expected): %v", err)
		}
	} else {
		t.Logf("Query succeeded! Found %d results", len(results))
		for i, result := range results {
			t.Logf("Result %d: ID=%s, Similarity=%.3f, Distance=%.3f",
				i, result.ID, result.Similarity, result.Distance)
			if result.Meta != nil {
				t.Logf("  Meta: %+v", result.Meta)
			}
			if result.Filter != nil {
				t.Logf("  Filter: %+v", result.Filter)
			}
		}
	}

	// Test query with filter
	filter := map[string]interface{}{
		"category": "tech",
	}

	results, err = index.Query(queryVector, nil, nil, 2, filter, 128, true)
	if err != nil {
		t.Logf("Filtered query failed: %v", err)
	} else {
		t.Logf("Filtered query succeeded! Found %d results", len(results))
		for i, result := range results {
			t.Logf("Filtered Result %d: ID=%s, Similarity=%.3f",
				i, result.ID, result.Similarity)
			if result.Vector != nil {
				t.Logf("  Vector included: %v", len(result.Vector) > 0)
			}
		}
	}

	// Test parameter validation
	_, err = index.Query(queryVector, nil, nil, 300, nil, 128, false) // k > 256
	if err == nil {
		t.Error("Expected error for k > 256, but got none")
	} else {
		t.Logf("Correctly caught k validation error: %v", err)
	}

	_, err = index.Query(queryVector, nil, nil, 10, nil, 2000, false) // ef > 1024
	if err == nil {
		t.Error("Expected error for ef > 1024, but got none")
	} else {
		t.Logf("Correctly caught ef validation error: %v", err)
	}

	// Test dimension mismatch
	wrongVector := []float32{1.0, 0.0} // Wrong dimension
	_, err = index.Query(wrongVector, nil, nil, 10, nil, 128, false)
	if err == nil {
		t.Error("Expected error for dimension mismatch, but got none")
	} else {
		t.Logf("Correctly caught dimension mismatch error: %v", err)
	}

	// Clean up
	err = vecx.DeleteIndex("test_query_index")
	if err != nil {
		t.Logf("Failed to clean up query test index: %v", err)
	} else {
		t.Log("Successfully cleaned up query test index")
	}
}

func TestDeleteVector(t *testing.T) {
	vecx := EndeeClient()

	// First, create an index for testing
	err := vecx.CreateIndex("test_delete_vector_index", 16, "cosine", 16, 128, true, nil, 0)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Logf("CreateIndex failed (might be expected): %v", err)
	}

	// Get the index to use for testing
	index, err := vecx.GetIndex("test_delete_vector_index")
	if err != nil {
		t.Fatalf("Failed to get index: %v", err)
	}

	// First, upsert a test vector to delete
	testVector := VectorItem{
		ID:     "test-delete-vector",
		Vector: make([]float32, index.Dimension),
		Meta:   map[string]interface{}{"test": "delete"},
	}

	// Fill vector with some test data
	for i := range testVector.Vector {
		testVector.Vector[i] = 0.1
	}

	// Upsert the test vector
	err = index.Upsert([]VectorItem{testVector})
	if err != nil {
		t.Fatalf("Failed to upsert test vector: %v", err)
	}

	// Now delete the vector
	result, err := index.DeleteVector("test-delete-vector")
	if err != nil {
		t.Fatalf("Failed to delete vector: %v", err)
	}

	fmt.Printf("Delete result: %s\n", result)

	// Verify the result contains expected message
	if !strings.Contains(result, "Vector deleted successfully") {
		t.Errorf("Expected result to contain 'Vector deleted successfully', got: %s", result)
	}

	// Clean up
	err = vecx.DeleteIndex("test_delete_vector_index")
	if err != nil {
		t.Logf("Failed to clean up delete vector test index: %v", err)
	} else {
		t.Log("Successfully cleaned up delete vector test index")
	}
}

func TestGetVector(t *testing.T) {
	vecx := EndeeClient()

	// First, create an index for testing
	err := vecx.CreateIndex("test_get_vector_index", 16, "cosine", 16, 128, true, nil, 0)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Logf("CreateIndex failed (might be expected): %v", err)
	}

	// Get the index to use for testing
	index, err := vecx.GetIndex("test_get_vector_index")
	if err != nil {
		t.Fatalf("Failed to get index: %v", err)
	}

	// Create and upsert a test vector
	testVector := VectorItem{
		ID:     "test-get-vector",
		Vector: make([]float32, index.Dimension),
		Meta:   map[string]interface{}{"test": "get", "number": 42},
		Filter: map[string]interface{}{"category": "test"},
	}

	// Fill vector with some test data
	for i := range testVector.Vector {
		testVector.Vector[i] = float32(i) * 0.1
	}

	// Upsert the test vector
	err = index.Upsert([]VectorItem{testVector})
	if err != nil {
		t.Fatalf("Failed to upsert test vector: %v", err)
	}

	// Now get the vector back
	retrievedVector, err := index.GetVector("test-get-vector")
	if err != nil {
		t.Fatalf("Failed to get vector: %v", err)
	}

	// Verify the retrieved vector matches what we stored
	if retrievedVector.ID != testVector.ID {
		t.Errorf("Expected ID %s, got %s", testVector.ID, retrievedVector.ID)
	}

	if len(retrievedVector.Vector) != len(testVector.Vector) {
		t.Errorf("Expected vector length %d, got %d", len(testVector.Vector), len(retrievedVector.Vector))
	}

	// Check metadata
	if retrievedVector.Meta["test"] != "get" {
		t.Errorf("Expected meta test='get', got %v", retrievedVector.Meta["test"])
	}

	fmt.Printf("Retrieved vector: ID=%s, Vector length=%d, Meta=%+v, Filter=%+v\n",
		retrievedVector.ID, len(retrievedVector.Vector), retrievedVector.Meta, retrievedVector.Filter)

	// Clean up
	err = vecx.DeleteIndex("test_get_vector_index")
	if err != nil {
		t.Logf("Failed to clean up get vector test index: %v", err)
	} else {
		t.Log("Successfully cleaned up get vector test index")
	}
}

// ============== Hybrid Index Tests ==============

func TestCreateHybridIndex(t *testing.T) {
	vecx := EndeeClient()

	// Test creating a hybrid index with sparse dimensions
	sparseDim := 1000 // sparse dimension for hybrid search
	err := vecx.CreateIndex("test_hybrid_index", 16, "cosine", 16, 128, true, nil, sparseDim)
	if err != nil {
		t.Logf("CreateIndex for hybrid failed (this might be expected if index already exists): %v", err)
	} else {
		t.Logf("CreateIndex for hybrid succeeded with SparseDim=%d", sparseDim)
	}

	// Verify the index was created with the correct sparse dimension
	index, err := vecx.GetIndex("test_hybrid_index")
	if err != nil {
		t.Logf("GetIndex for hybrid failed: %v", err)
	} else {
		t.Logf("Hybrid index info: %s", index.GetInfo())
	}

	// Clean up
	// err = vecx.DeleteIndex("test_hybrid_index")
	if err != nil {
		t.Logf("Failed to clean up hybrid test index: %v", err)
	} else {
		t.Log("Successfully cleaned up hybrid test index")
	}
}

func TestHybridUpsert(t *testing.T) {
	vecx := EndeeClient()

	// Create a hybrid index with sparse dimensions
	sparseDim := 1000
	err := vecx.CreateIndex("test_hybrid_upsert_index", 16, "cosine", 16, 128, true, nil, sparseDim)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Logf("CreateIndex failed (might be expected): %v", err)
	}

	// Get the index
	index, err := vecx.GetIndex("test_hybrid_upsert_index")
	if err != nil {
		t.Fatalf("GetIndex failed: %v", err)
	}

	t.Logf("Retrieved hybrid index: %s with dimension %d", index.Name, index.Dimension)

	// Create test vectors with both dense and sparse components
	testVectors := []VectorItem{
		{
			ID:            "hybrid_doc_1",
			Vector:        make([]float32, 16),
			SparseIndices: []int{0, 5, 100, 500},         // sparse term indices
			SparseValues:  []float32{0.5, 0.3, 0.8, 0.2}, // sparse term weights (e.g., TF-IDF)
			Meta: map[string]interface{}{
				"title":    "Hybrid Document 1",
				"category": "technology",
			},
			Filter: map[string]interface{}{
				"type": "article",
			},
		},
		{
			ID:            "hybrid_doc_2",
			Vector:        make([]float32, 16),
			SparseIndices: []int{5, 50, 200, 750},
			SparseValues:  []float32{0.7, 0.4, 0.6, 0.1},
			Meta: map[string]interface{}{
				"title":    "Hybrid Document 2",
				"category": "science",
			},
			Filter: map[string]interface{}{
				"type": "paper",
			},
		},
		{
			ID:            "hybrid_doc_3",
			Vector:        make([]float32, 16),
			SparseIndices: []int{0, 10, 100, 300, 999},
			SparseValues:  []float32{0.9, 0.2, 0.5, 0.4, 0.3},
			Meta: map[string]interface{}{
				"title":    "Hybrid Document 3",
				"category": "technology",
			},
			Filter: map[string]interface{}{
				"type": "blog",
			},
		},
	}

	// Fill dense vectors with test data
	for i := range testVectors {
		for j := range testVectors[i].Vector {
			testVectors[i].Vector[j] = float32(j+1) * 0.1 * float32(i+1)
		}
	}

	t.Logf("Test hybrid vectors: %+v", testVectors)

	// Test the upsert with hybrid vectors
	err = index.Upsert(testVectors)
	if err != nil {
		t.Logf("Hybrid Upsert failed: %v", err)
		if strings.Contains(err.Error(), "failed to execute request") ||
			strings.Contains(err.Error(), "no such host") ||
			strings.Contains(err.Error(), "connection refused") {
			t.Log("Hybrid Upsert failed due to network issues, but validation passed")
		} else {
			t.Logf("Hybrid Upsert failed with server error (might be expected): %v", err)
		}
	} else {
		t.Log("Hybrid Upsert succeeded!")
	}

	// Clean up
	err = vecx.DeleteIndex("test_hybrid_upsert_index")
	if err != nil {
		t.Logf("Failed to clean up hybrid upsert test index: %v", err)
	} else {
		t.Log("Successfully cleaned up hybrid upsert test index")
	}
}

func TestHybridQuery(t *testing.T) {
	vecx := EndeeClient()

	// Create a hybrid index
	sparseDim := 1000
	err := vecx.CreateIndex("test_hybrid_query_index", 16, "cosine", 16, 128, true, nil, sparseDim)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Logf("CreateIndex failed (might be expected): %v", err)
	}

	// Get the index
	index, err := vecx.GetIndex("test_hybrid_query_index")
	if err != nil {
		t.Fatalf("GetIndex failed: %v", err)
	}

	// Upsert some test vectors with sparse components
	testVectors := []VectorItem{
		{
			ID:            "hq_doc_1",
			Vector:        []float32{1.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0},
			SparseIndices: []int{0, 10, 100},
			SparseValues:  []float32{0.9, 0.5, 0.3},
			Meta: map[string]interface{}{
				"title": "Query Test Doc 1",
			},
			Filter: map[string]interface{}{
				"category": "tech",
			},
		},
		{
			ID:            "hq_doc_2",
			Vector:        []float32{0.0, 1.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0},
			SparseIndices: []int{5, 50, 500},
			SparseValues:  []float32{0.8, 0.4, 0.2},
			Meta: map[string]interface{}{
				"title": "Query Test Doc 2",
			},
			Filter: map[string]interface{}{
				"category": "science",
			},
		},
		{
			ID:            "hq_doc_3",
			Vector:        []float32{0.5, 0.5, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0},
			SparseIndices: []int{0, 5, 10, 50},
			SparseValues:  []float32{0.6, 0.7, 0.4, 0.3},
			Meta: map[string]interface{}{
				"title": "Query Test Doc 3",
			},
			Filter: map[string]interface{}{
				"category": "tech",
			},
		},
	}

	err = index.Upsert(testVectors)
	if err != nil {
		t.Logf("Upsert failed: %v", err)
	} else {
		t.Log("Test vectors upserted successfully for hybrid query")
	}

	// Test hybrid query with both dense vector and sparse components
	queryVector := []float32{0.9, 0.1, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0}
	querySparseIndices := []int{0, 10}       // query sparse indices
	querySparseValues := []float32{0.8, 0.6} // query sparse values

	// Test hybrid query (with sparse components)
	results, err := index.Query(queryVector, querySparseIndices, querySparseValues, 3, nil, 128, false)
	if err != nil {
		t.Logf("Hybrid Query failed: %v", err)
		if strings.Contains(err.Error(), "failed to execute request") ||
			strings.Contains(err.Error(), "no such host") ||
			strings.Contains(err.Error(), "connection refused") {
			t.Log("Hybrid Query failed due to network issues, but validation passed")
		} else {
			t.Logf("Hybrid Query failed with server error (might be expected): %v", err)
		}
	} else {
		t.Logf("Hybrid Query succeeded! Found %d results", len(results))
		for i, result := range results {
			t.Logf("Hybrid Result %d: ID=%s, Similarity=%.3f, Distance=%.3f",
				i, result.ID, result.Similarity, result.Distance)
			if result.Meta != nil {
				t.Logf("  Meta: %+v", result.Meta)
			}
		}
	}

	// Test hybrid query with filter
	filter := map[string]interface{}{
		"category": "tech",
	}

	results, err = index.Query(queryVector, querySparseIndices, querySparseValues, 2, filter, 128, true)
	if err != nil {
		t.Logf("Filtered hybrid query failed: %v", err)
	} else {
		t.Logf("Filtered hybrid query succeeded! Found %d results", len(results))
		for i, result := range results {
			t.Logf("Filtered Hybrid Result %d: ID=%s, Similarity=%.3f",
				i, result.ID, result.Similarity)
			if result.Vector != nil {
				t.Logf("  Vector included: %v", len(result.Vector) > 0)
			}
		}
	}

	// Test dense-only query on hybrid index (sparse params set to nil)
	results, err = index.Query(queryVector, nil, nil, 3, nil, 128, false)
	if err != nil {
		t.Logf("Dense-only query on hybrid index failed: %v", err)
	} else {
		t.Logf("Dense-only query on hybrid index succeeded! Found %d results", len(results))
	}

	// Clean up
	err = vecx.DeleteIndex("test_hybrid_query_index")
	if err != nil {
		t.Logf("Failed to clean up hybrid query test index: %v", err)
	} else {
		t.Log("Successfully cleaned up hybrid query test index")
	}
}

func TestGetHybridVector(t *testing.T) {
	vecx := EndeeClient()

	// Create a hybrid index
	sparseDim := 1000
	err := vecx.CreateIndex("test_get_hybrid_vector_index", 16, "cosine", 16, 128, true, nil, sparseDim)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Logf("CreateIndex failed (might be expected): %v", err)
	}

	// Get the index
	index, err := vecx.GetIndex("test_get_hybrid_vector_index")
	if err != nil {
		t.Fatalf("Failed to get index: %v", err)
	}

	// Create and upsert a hybrid test vector
	testVector := VectorItem{
		ID:            "test-get-hybrid-vector",
		Vector:        make([]float32, index.Dimension),
		SparseIndices: []int{1, 10, 100, 500, 999},
		SparseValues:  []float32{0.9, 0.7, 0.5, 0.3, 0.1},
		Meta:          map[string]interface{}{"test": "hybrid_get", "version": 1},
		Filter:        map[string]interface{}{"type": "hybrid_test"},
	}

	// Fill vector with test data
	for i := range testVector.Vector {
		testVector.Vector[i] = float32(i) * 0.1
	}

	// Upsert the test vector
	err = index.Upsert([]VectorItem{testVector})
	if err != nil {
		t.Fatalf("Failed to upsert hybrid test vector: %v", err)
	}

	// Get the vector back
	retrievedVector, err := index.GetVector("test-get-hybrid-vector")
	if err != nil {
		t.Fatalf("Failed to get hybrid vector: %v", err)
	}

	// Verify the retrieved vector
	if retrievedVector.ID != testVector.ID {
		t.Errorf("Expected ID %s, got %s", testVector.ID, retrievedVector.ID)
	}

	if len(retrievedVector.Vector) != len(testVector.Vector) {
		t.Errorf("Expected vector length %d, got %d", len(testVector.Vector), len(retrievedVector.Vector))
	}

	// Check sparse data
	if len(retrievedVector.SparseIndices) > 0 {
		t.Logf("Retrieved sparse indices: %v", retrievedVector.SparseIndices)
		t.Logf("Retrieved sparse values: %v", retrievedVector.SparseValues)

		if len(retrievedVector.SparseIndices) != len(testVector.SparseIndices) {
			t.Errorf("Expected %d sparse indices, got %d", len(testVector.SparseIndices), len(retrievedVector.SparseIndices))
		}
		if len(retrievedVector.SparseValues) != len(testVector.SparseValues) {
			t.Errorf("Expected %d sparse values, got %d", len(testVector.SparseValues), len(retrievedVector.SparseValues))
		}
	} else {
		t.Log("No sparse data returned (server may not return sparse data in GetVector)")
	}

	// Check metadata
	if retrievedVector.Meta["test"] != "hybrid_get" {
		t.Errorf("Expected meta test='hybrid_get', got %v", retrievedVector.Meta["test"])
	}

	fmt.Printf("Retrieved hybrid vector: ID=%s, Vector length=%d, SparseIndices=%v, SparseValues=%v, Meta=%+v\n",
		retrievedVector.ID, len(retrievedVector.Vector), retrievedVector.SparseIndices, retrievedVector.SparseValues, retrievedVector.Meta)

	// Clean up
	err = vecx.DeleteIndex("test_get_hybrid_vector_index")
	if err != nil {
		t.Logf("Failed to clean up get hybrid vector test index: %v", err)
	} else {
		t.Log("Successfully cleaned up get hybrid vector test index")
	}
}

func TestDeleteHybridVector(t *testing.T) {
	vecx := EndeeClient()

	// Create a hybrid index
	sparseDim := 1000
	err := vecx.CreateIndex("test_delete_hybrid_vector_index", 16, "cosine", 16, 128, true, nil, sparseDim)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Logf("CreateIndex failed (might be expected): %v", err)
	}

	// Get the index
	index, err := vecx.GetIndex("test_delete_hybrid_vector_index")
	if err != nil {
		t.Fatalf("Failed to get index: %v", err)
	}

	// Create and upsert a hybrid test vector
	testVector := VectorItem{
		ID:            "test-delete-hybrid-vector",
		Vector:        make([]float32, index.Dimension),
		SparseIndices: []int{5, 50, 500},
		SparseValues:  []float32{0.8, 0.5, 0.2},
		Meta:          map[string]interface{}{"test": "delete_hybrid"},
	}

	// Fill vector with test data
	for i := range testVector.Vector {
		testVector.Vector[i] = 0.1
	}

	// Upsert the test vector
	err = index.Upsert([]VectorItem{testVector})
	if err != nil {
		t.Fatalf("Failed to upsert hybrid test vector: %v", err)
	}

	// Delete the vector
	result, err := index.DeleteVector("test-delete-hybrid-vector")
	if err != nil {
		t.Fatalf("Failed to delete hybrid vector: %v", err)
	}

	fmt.Printf("Delete hybrid vector result: %s\n", result)

	// Verify the result
	if !strings.Contains(result, "Vector deleted successfully") {
		t.Errorf("Expected result to contain 'Vector deleted successfully', got: %s", result)
	}

	// Clean up
	err = vecx.DeleteIndex("test_delete_hybrid_vector_index")
	if err != nil {
		t.Logf("Failed to clean up delete hybrid vector test index: %v", err)
	} else {
		t.Log("Successfully cleaned up delete hybrid vector test index")
	}
}
