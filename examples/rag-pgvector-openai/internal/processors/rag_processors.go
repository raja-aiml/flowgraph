package processors

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"

	"rag-pgvector-openai/internal/database"
	"rag-pgvector-openai/internal/openai"
)

// ProductionEmbedder handles text embedding using OpenAI
type ProductionEmbedder struct {
	client *openai.Client
	model  string
}

// NewProductionEmbedder creates a new production embedder
func NewProductionEmbedder(client *openai.Client, model string) *ProductionEmbedder {
	return &ProductionEmbedder{
		client: client,
		model:  model,
	}
}

func (p *ProductionEmbedder) Name() string {
	return "production_embedder"
}

func (p *ProductionEmbedder) Process(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	text, exists := input["text"]
	if !exists {
		return nil, fmt.Errorf("text field required for embedding")
	}

	textStr, ok := text.(string)
	if !ok {
		return nil, fmt.Errorf("text must be a string")
	}

	// Create embedding using OpenAI
	resp, err := p.client.CreateEmbedding(ctx, openai.EmbeddingRequest{
		Text:  textStr,
		Model: p.model,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding: %w", err)
	}

	log.Printf("Created embedding for text (length: %d), tokens used: %d", len(textStr), resp.Usage.TotalTokens)

	return map[string]interface{}{
		"embedding":   resp.Embedding,
		"text":        textStr,
		"model":       resp.Model,
		"provider":    "openai",
		"tokens_used": resp.Usage.TotalTokens,
		"dimensions":  len(resp.Embedding),
	}, nil
}

// ProductionVectorSearch handles vector similarity search using pgvector
type ProductionVectorSearch struct {
	vectorDB            *database.VectorDB
	maxResults          int
	similarityThreshold float64
}

// NewProductionVectorSearch creates a new production vector search processor
func NewProductionVectorSearch(vectorDB *database.VectorDB, maxResults int, similarityThreshold float64) *ProductionVectorSearch {
	return &ProductionVectorSearch{
		vectorDB:            vectorDB,
		maxResults:          maxResults,
		similarityThreshold: similarityThreshold,
	}
}

func (p *ProductionVectorSearch) Name() string {
	return "production_vector_search"
}

func (p *ProductionVectorSearch) Process(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	embedding, exists := input["embedding"]
	if !exists {
		return nil, fmt.Errorf("embedding field required for search")
	}

	// Convert embedding to []float32
	var embeddingSlice []float32
	switch e := embedding.(type) {
	case []float32:
		embeddingSlice = e
	case []float64:
		embeddingSlice = make([]float32, len(e))
		for i, v := range e {
			embeddingSlice[i] = float32(v)
		}
	case []interface{}:
		embeddingSlice = make([]float32, len(e))
		for i, v := range e {
			if f, ok := v.(float64); ok {
				embeddingSlice[i] = float32(f)
			} else if f, ok := v.(float32); ok {
				embeddingSlice[i] = f
			} else {
				return nil, fmt.Errorf("invalid embedding format at index %d", i)
			}
		}
	default:
		return nil, fmt.Errorf("embedding must be a slice of numbers")
	}

	// Perform similarity search
	results, err := p.vectorDB.SearchSimilarDocuments(ctx, embeddingSlice, p.maxResults, p.similarityThreshold)
	if err != nil {
		return nil, fmt.Errorf("failed to perform vector search: %w", err)
	}

	log.Printf("Vector search found %d results with similarity > %.2f", len(results), p.similarityThreshold)

	// Convert results to the expected format
	searchResults := make([]map[string]interface{}, len(results))
	for i, result := range results {
		searchResults[i] = map[string]interface{}{
			"id":         result.Document.ID,
			"content":    result.Document.Content,
			"metadata":   result.Document.Metadata,
			"similarity": result.Similarity,
			"created_at": result.Document.CreatedAt,
		}
	}

	return map[string]interface{}{
		"results":              searchResults,
		"query_embedding":      embeddingSlice,
		"provider":             "pgvector",
		"total_results":        len(results),
		"similarity_threshold": p.similarityThreshold,
	}, nil
}

// ProductionRAGGenerator handles response generation using OpenAI
type ProductionRAGGenerator struct {
	client      *openai.Client
	model       string
	maxTokens   int
	temperature float32
	maxDocs     int
}

// NewProductionRAGGenerator creates a new production RAG generator
func NewProductionRAGGenerator(client *openai.Client, model string, maxTokens int, temperature float64, maxDocs int) *ProductionRAGGenerator {
	return &ProductionRAGGenerator{
		client:      client,
		model:       model,
		maxTokens:   maxTokens,
		temperature: float32(temperature),
		maxDocs:     maxDocs,
	}
}

func (p *ProductionRAGGenerator) Name() string {
	return "production_rag_generator"
}

func (p *ProductionRAGGenerator) Process(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	query, queryExists := input["query"]
	if !queryExists {
		// Try to get query from original input or text
		if originalQuery, exists := input["text"]; exists {
			query = originalQuery
		} else {
			return nil, fmt.Errorf("query or text field required for generation")
		}
	}

	queryStr, ok := query.(string)
	if !ok {
		return nil, fmt.Errorf("query must be a string")
	}

	// Extract document contents from search results
	var documents []string
	if results, exists := input["results"]; exists {
		if resultsList, ok := results.([]map[string]interface{}); ok {
			for _, result := range resultsList {
				if content, exists := result["content"]; exists {
					if contentStr, ok := content.(string); ok {
						documents = append(documents, contentStr)
					}
				}
			}
		}
	}

	log.Printf("Generating RAG response for query with %d retrieved documents", len(documents))

	// Generate response using OpenAI with RAG context
	resp, err := p.client.CreateRAGResponse(ctx, queryStr, documents, p.maxDocs)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RAG response: %w", err)
	}

	log.Printf("Generated response (length: %d), tokens used: %d", len(resp.Content), resp.Usage.TotalTokens)

	return map[string]interface{}{
		"response":          resp.Content,
		"query":             queryStr,
		"context_docs_used": len(documents),
		"model":             resp.Model,
		"provider":          "openai",
		"finish_reason":     resp.FinishReason,
		"tokens_used":       resp.Usage.TotalTokens,
		"temperature":       p.temperature,
	}, nil
}

// ResultRanker re-ranks search results based on various criteria
type ResultRanker struct {
	diversityWeight float64
	recencyWeight   float64
}

// NewResultRanker creates a new result ranker
func NewResultRanker(diversityWeight, recencyWeight float64) *ResultRanker {
	return &ResultRanker{
		diversityWeight: diversityWeight,
		recencyWeight:   recencyWeight,
	}
}

func (r *ResultRanker) Name() string {
	return "result_ranker"
}

func (r *ResultRanker) Process(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	results, exists := input["results"]
	if !exists {
		return input, nil // No results to rank
	}

	resultsList, ok := results.([]map[string]interface{})
	if !ok {
		return input, nil // Invalid results format
	}

	if len(resultsList) <= 1 {
		return input, nil // No need to rank single result
	}

	log.Printf("Re-ranking %d search results", len(resultsList))

	// Apply re-ranking algorithm
	rankedResults := r.rankResults(resultsList)

	// Update the input with ranked results
	output := make(map[string]interface{})
	for k, v := range input {
		output[k] = v
	}
	output["results"] = rankedResults
	output["reranked"] = true
	output["rerank_method"] = "production_ranking"

	return output, nil
}

func (r *ResultRanker) rankResults(results []map[string]interface{}) []map[string]interface{} {
	// Create a copy of results with ranking scores
	type rankedResult struct {
		result map[string]interface{}
		score  float64
	}

	rankedResults := make([]rankedResult, len(results))

	for i, result := range results {
		// Base score from similarity
		baseScore := 0.0
		if similarity, exists := result["similarity"]; exists {
			if simFloat, ok := similarity.(float64); ok {
				baseScore = simFloat
			}
		}

		// Content length bonus (longer content might be more informative)
		contentBonus := 0.0
		if content, exists := result["content"]; exists {
			if contentStr, ok := content.(string); ok {
				// Normalize content length bonus (0.0 to 0.1)
				contentBonus = float64(len(contentStr)) / 10000.0
				if contentBonus > 0.1 {
					contentBonus = 0.1
				}
			}
		}

		// Metadata quality bonus
		metadataBonus := 0.0
		if metadata, exists := result["metadata"]; exists {
			if metaMap, ok := metadata.(map[string]interface{}); ok {
				// More metadata fields = higher quality bonus
				metadataBonus = float64(len(metaMap)) * 0.01
				if metadataBonus > 0.05 {
					metadataBonus = 0.05
				}
			}
		}

		// Calculate final score
		finalScore := baseScore + contentBonus + metadataBonus

		rankedResults[i] = rankedResult{
			result: result,
			score:  finalScore,
		}
	}

	// Sort by final score (descending)
	sort.Slice(rankedResults, func(i, j int) bool {
		return rankedResults[i].score > rankedResults[j].score
	})

	// Extract sorted results and add ranking information
	sortedResults := make([]map[string]interface{}, len(rankedResults))
	for i, rr := range rankedResults {
		result := make(map[string]interface{})
		for k, v := range rr.result {
			result[k] = v
		}
		result["rank_score"] = rr.score
		result["rank_position"] = i + 1
		sortedResults[i] = result
	}

	return sortedResults
}

// QueryProcessor handles initial query processing and validation
type QueryProcessor struct{}

// NewQueryProcessor creates a new query processor
func NewQueryProcessor() *QueryProcessor {
	return &QueryProcessor{}
}

func (q *QueryProcessor) Name() string {
	return "production_query_processor"
}

func (q *QueryProcessor) Process(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	query, exists := input["query"]
	if !exists {
		return nil, fmt.Errorf("query field required")
	}

	queryStr, ok := query.(string)
	if !ok {
		return nil, fmt.Errorf("query must be a string")
	}

	// Clean and validate query
	queryStr = strings.TrimSpace(queryStr)
	if queryStr == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	log.Printf("Processing query: %s (length: %d)", queryStr, len(queryStr))

	// Basic query preprocessing
	processedQuery := q.preprocessQuery(queryStr)

	return map[string]interface{}{
		"query":            queryStr,
		"processed_query":  processedQuery,
		"text":             processedQuery, // For embedding
		"query_length":     len(queryStr),
		"processed_length": len(processedQuery),
		"timestamp":        ctx.Value("timestamp"),
	}, nil
}

func (q *QueryProcessor) preprocessQuery(query string) string {
	// Basic query preprocessing
	// In production, you might want more sophisticated preprocessing
	processed := strings.TrimSpace(query)

	// Remove extra whitespace
	processed = strings.Join(strings.Fields(processed), " ")

	// Convert to lowercase for consistency (optional, depending on use case)
	// processed = strings.ToLower(processed)

	return processed
}
