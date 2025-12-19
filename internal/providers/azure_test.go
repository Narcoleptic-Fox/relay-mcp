package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Narcoleptic-Fox/relay-mcp/internal/config"
	"github.com/Narcoleptic-Fox/relay-mcp/internal/types"
)

func TestAzureProvider_URLFormat(t *testing.T) {
	var capturedURL string
	var capturedHeaders http.Header
	var capturedBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		capturedHeaders = r.Header

		// Parse request body
		json.NewDecoder(r.Body).Decode(&capturedBody)

		// Return valid response
		resp := openAIResponse{
			Choices: []openAIChoice{{
				Message:      openAIMessage{Role: "assistant", Content: "test response"},
				FinishReason: "stop",
			}},
			Usage: openAIUsage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.Config{
		AzureAPIKey:   "test-api-key",
		AzureEndpoint: server.URL,
		ModelRegistries: map[types.ProviderType][]types.ModelCapabilities{
			types.ProviderAzure: {
				{
					Provider:  types.ProviderAzure,
					ModelName: "gpt-4o",
				},
			},
		},
	}

	provider, err := NewAzureProvider(cfg)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	_, err = provider.GenerateContent(context.Background(), &GenerateRequest{
		Prompt: "test prompt",
		Model:  "gpt-4o",
	})
	if err != nil {
		t.Fatalf("failed to generate content: %v", err)
	}

	// Verify URL format includes deployment name and api-version
	if !strings.Contains(capturedURL, "/openai/deployments/gpt-4o/chat/completions") {
		t.Errorf("unexpected URL format: %s", capturedURL)
	}
	if !strings.Contains(capturedURL, "api-version=") {
		t.Errorf("missing api-version in URL: %s", capturedURL)
	}

	// Verify api-key header is used (NOT Authorization: Bearer)
	if capturedHeaders.Get("api-key") != "test-api-key" {
		t.Errorf("expected api-key header with value 'test-api-key', got: %v", capturedHeaders.Get("api-key"))
	}
	if capturedHeaders.Get("Authorization") != "" {
		t.Error("should NOT have Authorization header for Azure")
	}

	// Verify Content-Type
	if capturedHeaders.Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type application/json, got: %s", capturedHeaders.Get("Content-Type"))
	}

	// Verify request body does NOT include model (it's in the URL for Azure)
	if _, hasModel := capturedBody["model"]; hasModel {
		t.Error("Azure request body should not include 'model' field")
	}
}

func TestAzureProvider_EndpointNormalization(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		expected string
	}{
		{
			name:     "no trailing slash",
			endpoint: "https://example.openai.azure.com",
			expected: "/openai/deployments/gpt-4o/chat/completions",
		},
		{
			name:     "with trailing slash",
			endpoint: "https://example.openai.azure.com/",
			expected: "/openai/deployments/gpt-4o/chat/completions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedURL string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedURL = r.URL.Path

				resp := openAIResponse{
					Choices: []openAIChoice{{
						Message: openAIMessage{Content: "test"},
					}},
				}
				json.NewEncoder(w).Encode(resp)
			}))
			defer server.Close()

			cfg := &config.Config{
				AzureAPIKey:   "test-key",
				AzureEndpoint: server.URL + "/", // Add trailing slash to test normalization
			}

			provider, err := NewAzureProvider(cfg)
			if err != nil {
				t.Fatalf("failed to create provider: %v", err)
			}

			_, err = provider.GenerateContent(context.Background(), &GenerateRequest{
				Prompt: "test",
				Model:  "gpt-4o",
			})
			if err != nil {
				t.Fatalf("failed to generate: %v", err)
			}

			if capturedURL != tt.expected {
				t.Errorf("expected URL path %s, got %s", tt.expected, capturedURL)
			}
		})
	}
}

func TestAzureProvider_NotConfigured(t *testing.T) {
	// Missing API key
	cfg := &config.Config{
		AzureEndpoint: "https://example.openai.azure.com",
	}
	_, err := NewAzureProvider(cfg)
	if err == nil || !strings.Contains(err.Error(), "AZURE_OPENAI_API_KEY") {
		t.Errorf("expected error about missing API key, got: %v", err)
	}

	// Missing endpoint
	cfg = &config.Config{
		AzureAPIKey: "test-key",
	}
	_, err = NewAzureProvider(cfg)
	if err == nil || !strings.Contains(err.Error(), "AZURE_OPENAI_ENDPOINT") {
		t.Errorf("expected error about missing endpoint, got: %v", err)
	}
}

func TestAzureProvider_IsConfigured(t *testing.T) {
	cfg := &config.Config{
		AzureAPIKey:   "test-key",
		AzureEndpoint: "https://example.openai.azure.com",
	}

	provider, err := NewAzureProvider(cfg)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	if !provider.IsConfigured() {
		t.Error("expected provider to be configured")
	}
}
