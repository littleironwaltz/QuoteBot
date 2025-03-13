package repository

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/littleironwaltz/quotebot/config"
)

// HTTPError holds error information for HTTP requests
type HTTPError struct {
	StatusCode int
	Message    string
	Err        error
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP error (status %d): %s: %v", e.StatusCode, e.Message, e.Err)
}

// RetryPolicy defines the retry behavior for HTTP requests
type RetryPolicy struct {
	MaxRetries   int
	RetryBackoff time.Duration
}

// HTTPClient handles HTTP communication
type HTTPClient struct {
	client      *http.Client
	retryPolicy RetryPolicy
	bufferPool  *sync.Pool
}

// NewHTTPClient creates a new HTTPClient instance
func NewHTTPClient(cfg *config.Config) *HTTPClient {
	// Configure TLS
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
	}

	transport := &http.Transport{
		IdleConnTimeout:     DefaultIdleTimeout,
		MaxIdleConns:        MaxIdleConnections,
		MaxIdleConnsPerHost: MaxIdleConnsPerHost,
		TLSClientConfig:     tlsConfig,
	}

	return &HTTPClient{
		client: &http.Client{
			Timeout:   cfg.HTTPTimeout,
			Transport: transport,
		},
		retryPolicy: RetryPolicy{
			MaxRetries:   cfg.MaxRetries,
			RetryBackoff: cfg.RetryBackoff,
		},
		bufferPool: &sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
	}
}

// DoRequest sends an HTTP request with retry logic
func (c *HTTPClient) DoRequest(ctx context.Context, method string, url string, body interface{}, headers map[string]string) (*http.Response, error) {
	// Encode body if provided
	var buf *bytes.Buffer
	var bodyBytes []byte
	if body != nil {
		buf = c.bufferPool.Get().(*bytes.Buffer)
		buf.Reset()
		defer c.bufferPool.Put(buf)

		if err := json.NewEncoder(buf).Encode(body); err != nil {
			return nil, fmt.Errorf("failed to encode request body: %w", err)
		}

		// Save a copy for retries
		bodyBytes = make([]byte, buf.Len())
		copy(bodyBytes, buf.Bytes())
	}

	// Execute request with retries
	var resp *http.Response
	var err error

	for attempt := 0; attempt <= c.retryPolicy.MaxRetries; attempt++ {
		if attempt > 0 {
			// Apply backoff with a maximum limit
			backoff := c.calculateBackoff(attempt)

			select {
			case <-time.After(backoff):
				// Continue with retry
			case <-ctx.Done():
				return nil, fmt.Errorf("context cancelled during backoff: %w", ctx.Err())
			}

			// Reset buffer for retry if needed
			if buf != nil && len(bodyBytes) > 0 {
				buf.Reset()
				buf.Write(bodyBytes)
			}
		}

		// Make the actual request
		resp, err = c.sendRequest(ctx, method, url, buf, headers)
		if err == nil {
			// Request succeeded
			return resp, nil
		}

		// Determine if we should retry
		if !c.shouldRetry(err, attempt) {
			return nil, err
		}

		// Log retry attempt
		log.Printf("Request failed (attempt %d/%d): %v. Retrying...",
			attempt+1, c.retryPolicy.MaxRetries+1, sanitizeError(err))
	}

	// All retries failed
	return nil, fmt.Errorf("request failed after %d attempts: %w", c.retryPolicy.MaxRetries+1, err)
}

// calculateBackoff determines the backoff duration for a retry
func (c *HTTPClient) calculateBackoff(attempt int) time.Duration {
	backoff := c.retryPolicy.RetryBackoff * time.Duration(1<<uint(attempt-1))
	if backoff > MaxBackoffDuration {
		backoff = MaxBackoffDuration
	}
	return backoff
}

// shouldRetry determines if a request should be retried
func (c *HTTPClient) shouldRetry(err error, attempt int) bool {
	// Don't retry if we've reached the maximum
	if attempt >= c.retryPolicy.MaxRetries {
		return false
	}

	// Check HTTP errors specifically
	if httpErr, ok := err.(*HTTPError); ok {
		// Don't retry on client errors (except 429 Too Many Requests)
		if httpErr.StatusCode >= 400 && httpErr.StatusCode < 500 && httpErr.StatusCode != 429 {
			return false
		}

		// Log rate limiting specifically
		if httpErr.StatusCode == 429 {
			log.Printf("Rate limit exceeded (attempt %d/%d), backing off",
				attempt+1, c.retryPolicy.MaxRetries+1)
		}

		// Retry on server errors and rate limits
		return true
	}

	// Retry on network errors
	return true
}

// sendRequest sends a single HTTP request without retrying
func (c *HTTPClient) sendRequest(ctx context.Context, method string, url string, body *bytes.Buffer, headers map[string]string) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = body
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Handle error response
		var errorBody string
		if resp.Body != nil {
			// Read response body with limit
			limitReader := io.LimitReader(resp.Body, DefaultBufferSize)
			bodyBytes, readErr := io.ReadAll(limitReader)
			if readErr == nil {
				errorBody = string(bodyBytes)
				// Reset the body for further reading
				resp.Body.Close()
				resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			}
		}

		// Sanitize the error body
		errorBody = sanitizeErrorBody(errorBody)

		return resp, &HTTPError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("%s: %s", resp.Status, errorBody),
			Err:        err,
		}
	}

	return resp, nil
}

// DecodeJSONResponse decodes a JSON response into the provided target
func (c *HTTPClient) DecodeJSONResponse(resp *http.Response, target interface{}) error {
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}
	return nil
}

// EncodeJSONRequest encodes a request body as JSON and returns a buffer from the pool
func (c *HTTPClient) EncodeJSONRequest(body interface{}) (*bytes.Buffer, []byte, error) {
	buf := c.bufferPool.Get().(*bytes.Buffer)
	buf.Reset()

	if err := json.NewEncoder(buf).Encode(body); err != nil {
		c.bufferPool.Put(buf)
		return nil, nil, fmt.Errorf("failed to encode request body: %w", err)
	}

	// Make a copy of the buffer for potential retries
	bodyBytes := make([]byte, buf.Len())
	copy(bodyBytes, buf.Bytes())

	return buf, bodyBytes, nil
}

// sanitizeError removes sensitive information from error messages
func sanitizeError(err error) error {
	if err == nil {
		return nil
	}

	errMsg := err.Error()
	// Mask sensitive patterns
	sensitivePatterns := []string{"Bearer ", "accessJwt", "refreshJwt", "Authorization"}

	for _, pattern := range sensitivePatterns {
		if strings.Contains(strings.ToLower(errMsg), strings.ToLower(pattern)) {
			start := strings.Index(strings.ToLower(errMsg), strings.ToLower(pattern))
			end := start + len(pattern) + 30 // pattern + some extra chars for the token
			if end > len(errMsg) {
				end = len(errMsg)
			}

			errMsg = errMsg[:start] + "[REDACTED]" + errMsg[end:]
		}
	}

	if errMsg != err.Error() {
		return fmt.Errorf("%s", errMsg)
	}
	return err
}

// sanitizeErrorBody removes sensitive information from error response bodies
func sanitizeErrorBody(body string) string {
	if body == "" {
		return ""
	}

	// Sanitize JWT tokens and other sensitive information
	sensitivePatterns := []string{"eyJ", "jwt", "bearer", "auth", "token"}

	result := body
	for _, pattern := range sensitivePatterns {
		if strings.Contains(strings.ToLower(result), strings.ToLower(pattern)) {
			start := strings.Index(strings.ToLower(result), strings.ToLower(pattern))
			end := start + len(pattern) + 30 // pattern + some extra chars
			if end > len(result) {
				end = len(result)
			}

			result = result[:start] + "[REDACTED]" + result[end:]
		}
	}

	return result
}
