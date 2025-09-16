package openai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
)

// Client wraps the OpenAI client with additional functionality
type Client struct {
	client         *openai.Client
	embeddingModel string
	chatModel      string
	maxTokens      int
	temperature    float32
	requestTimeout time.Duration
	rateLimiter    *RateLimiter
}

// RateLimiter handles OpenAI API rate limiting
type RateLimiter struct {
	rpmLimit int
	tpmLimit int
	// In production, you'd use a proper rate limiter like redis-based
}

// EmbeddingRequest represents a request for text embedding
type EmbeddingRequest struct {
	Text  string
	Model string
}

// EmbeddingResponse represents the response from embedding API
type EmbeddingResponse struct {
	Embedding []float32
	Model     string
	Usage     EmbeddingUsage
}

type EmbeddingUsage struct {
	PromptTokens int
	TotalTokens  int
}

// ChatRequest represents a request for chat completion
type ChatRequest struct {
	Messages    []ChatMessage
	Model       string
	MaxTokens   int
	Temperature float32
	Context     string
}

type ChatMessage struct {
	Role    string
	Content string
}

// ChatResponse represents the response from chat completion API
type ChatResponse struct {
	Content      string
	Model        string
	Usage        ChatUsage
	FinishReason string
}

type ChatUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// NewClient creates a new OpenAI client wrapper
func NewClient(apiKey, embeddingModel, chatModel string, maxTokens int, temperature float64, requestTimeout time.Duration) *Client {
	client := openai.NewClient(apiKey)

	return &Client{
		client:         client,
		embeddingModel: embeddingModel,
		chatModel:      chatModel,
		maxTokens:      maxTokens,
		temperature:    float32(temperature),
		requestTimeout: requestTimeout,
		rateLimiter:    &RateLimiter{}, // Initialize with proper rate limiter in production
	}
}

// CreateEmbedding generates embeddings for the given text
func (c *Client) CreateEmbedding(ctx context.Context, req EmbeddingRequest) (*EmbeddingResponse, error) {
	// Use default model if not specified
	model := req.Model
	if model == "" {
		model = c.embeddingModel
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, c.requestTimeout)
	defer cancel()

	// Make the API request
	resp, err := c.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: []string{req.Text},
		Model: openai.EmbeddingModel(model),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create embedding: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embeddings returned from API")
	}

	return &EmbeddingResponse{
		Embedding: resp.Data[0].Embedding,
		Model:     model,
		Usage: EmbeddingUsage{
			PromptTokens: resp.Usage.PromptTokens,
			TotalTokens:  resp.Usage.TotalTokens,
		},
	}, nil
}

// CreateChatCompletion generates a chat completion response
func (c *Client) CreateChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// Use default model if not specified
	model := req.Model
	if model == "" {
		model = c.chatModel
	}

	// Use default values if not specified
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = c.maxTokens
	}

	temperature := req.Temperature
	if temperature == 0 {
		temperature = c.temperature
	}

	// Convert messages
	var messages []openai.ChatCompletionMessage
	for _, msg := range req.Messages {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, c.requestTimeout)
	defer cancel()

	// Make the API request
	resp, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create chat completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned from API")
	}

	choice := resp.Choices[0]
	return &ChatResponse{
		Content:      choice.Message.Content,
		Model:        model,
		FinishReason: string(choice.FinishReason),
		Usage: ChatUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}, nil
}

// CreateRAGResponse generates a response based on query and retrieved documents
func (c *Client) CreateRAGResponse(ctx context.Context, query string, documents []string, maxDocs int) (*ChatResponse, error) {
	// Limit the number of documents to avoid token limits
	if maxDocs > 0 && len(documents) > maxDocs {
		documents = documents[:maxDocs]
	}

	// Build context from documents
	context := strings.Join(documents, "\n\n")

	// Create system message with RAG instructions
	systemMessage := `You are a helpful assistant that answers questions based on the provided context. 
Use only the information from the context to answer questions. If the context doesn't contain enough 
information to answer the question, say so clearly. Be concise but comprehensive in your responses.`

	// Create user message with context and query
	userMessage := fmt.Sprintf(`Context:
%s

Question: %s

Please answer the question based on the provided context.`, context, query)

	req := ChatRequest{
		Messages: []ChatMessage{
			{Role: "system", Content: systemMessage},
			{Role: "user", Content: userMessage},
		},
		Context: context,
	}

	return c.CreateChatCompletion(ctx, req)
}

// BatchCreateEmbeddings creates embeddings for multiple texts efficiently
func (c *Client) BatchCreateEmbeddings(ctx context.Context, texts []string, model string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("no texts provided")
	}

	// Use default model if not specified
	if model == "" {
		model = c.embeddingModel
	}

	// OpenAI supports batch embedding requests
	ctx, cancel := context.WithTimeout(ctx, c.requestTimeout)
	defer cancel()

	resp, err := c.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: texts,
		Model: openai.EmbeddingModel(model),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create batch embeddings: %w", err)
	}

	if len(resp.Data) != len(texts) {
		return nil, fmt.Errorf("mismatch between input texts and returned embeddings")
	}

	embeddings := make([][]float32, len(texts))
	for i, data := range resp.Data {
		embeddings[i] = data.Embedding
	}

	return embeddings, nil
}

// EstimateTokens provides a rough estimate of token count for text
func (c *Client) EstimateTokens(text string) int {
	// Rough estimation: ~4 characters per token for English text
	// In production, use tiktoken library for accurate token counting
	return len(text) / 4
}

// TruncateToTokenLimit truncates text to fit within token limits
func (c *Client) TruncateToTokenLimit(text string, maxTokens int) string {
	estimatedTokens := c.EstimateTokens(text)
	if estimatedTokens <= maxTokens {
		return text
	}

	// Rough truncation - in production, use proper tokenization
	ratio := float64(maxTokens) / float64(estimatedTokens)
	truncateAt := int(float64(len(text)) * ratio)

	// Try to truncate at word boundaries
	if truncateAt < len(text) {
		for i := truncateAt; i > 0; i-- {
			if text[i] == ' ' {
				return text[:i] + "..."
			}
		}
	}

	return text[:truncateAt] + "..."
}

// HealthCheck verifies the OpenAI API connection
func (c *Client) HealthCheck(ctx context.Context) error {
	// Simple health check using a minimal embedding request
	_, err := c.CreateEmbedding(ctx, EmbeddingRequest{
		Text: "health check",
	})
	return err
}
