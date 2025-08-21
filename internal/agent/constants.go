package agent

import "time"

const (
	// maxAPIRetries defines the maximum number of retries for API errors
	maxAPIRetries = 3
	// retryDelay is the base delay between retries (exponential backoff)
	retryDelay = 2 * time.Second
	// maxFileSize is the maximum file size to process during searches (16MB)
	maxFileSize = 16 * 1024 * 1024
)
