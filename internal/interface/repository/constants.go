package repository

import "time"

// Common constants for the repository package
const (
	// HTTP related constants
	MaxBackoffDuration  = 30 * time.Second
	DefaultBufferSize   = 1024
	DefaultIdleTimeout  = 180 * time.Second
	MaxIdleConnections  = 100
	MaxIdleConnsPerHost = 5

	// Token related constants
	TokenCacheTimeout = 60 * time.Minute
	DefaultKeySize    = 32 // AES-256

	// Retry related constants
	DefaultMaxRetries = 3
)
