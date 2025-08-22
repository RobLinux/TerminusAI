package tokenizer

import (
	"errors"
	"strings"
)

// Tokenizer interface defines methods for token counting and text splitting
type Tokenizer interface {
	// CountTokens estimates the number of tokens in the given text
	CountTokens(text string) int

	// GetMaxContextTokens returns the maximum context window for the model
	GetMaxContextTokens(model string) int

	// SplitText splits text into chunks that fit within the token limit
	SplitText(text string, maxTokens int) ([]string, error)

	// EstimateMessageTokens estimates tokens for a complete message including role overhead
	EstimateMessageTokens(role, content string) int
}

// BaseTokenizer provides common functionality for all tokenizers
type BaseTokenizer struct {
	tokensPerWord float64
}

// CountTokens implements a basic word-based token counting
func (t *BaseTokenizer) CountTokens(text string) int {
	return t.estimateTokensByWords(text)
}

// Simple word-based token estimation (fallback method)
func (t *BaseTokenizer) estimateTokensByWords(text string) int {
	if text == "" {
		return 0
	}

	// Basic estimation: split by whitespace and multiply by average tokens per word
	words := strings.Fields(text)
	return int(float64(len(words)) * t.tokensPerWord)
}

// SplitText splits text into chunks that fit within the token limit
func (t *BaseTokenizer) SplitText(text string, maxTokens int) ([]string, error) {
	if maxTokens <= 0 {
		return nil, errors.New("maxTokens must be greater than 0")
	}

	// If the entire text fits, return it as a single chunk
	if t.CountTokens(text) <= maxTokens {
		return []string{text}, nil
	}

	var chunks []string
	lines := strings.Split(text, "\n")
	currentChunk := ""

	for _, line := range lines {
		testChunk := currentChunk
		if testChunk != "" {
			testChunk += "\n"
		}
		testChunk += line

		// If adding this line would exceed the limit
		if t.CountTokens(testChunk) > maxTokens {
			// If we have a current chunk, save it
			if currentChunk != "" {
				chunks = append(chunks, currentChunk)
				currentChunk = ""
			}

			// If a single line is too long, split it by sentences or words
			if t.CountTokens(line) > maxTokens {
				lineParts, err := t.splitLongLine(line, maxTokens)
				if err != nil {
					return nil, err
				}
				chunks = append(chunks, lineParts...)
			} else {
				currentChunk = line
			}
		} else {
			currentChunk = testChunk
		}
	}

	// Add the final chunk if it exists
	if currentChunk != "" {
		chunks = append(chunks, currentChunk)
	}

	return chunks, nil
}

// splitLongLine splits a very long line into smaller parts
func (t *BaseTokenizer) splitLongLine(line string, maxTokens int) ([]string, error) {
	var parts []string

	// Try splitting by sentences first
	sentences := strings.Split(line, ". ")
	currentPart := ""

	for i, sentence := range sentences {
		testPart := currentPart
		if testPart != "" {
			testPart += ". "
		}
		testPart += sentence
		if i < len(sentences)-1 {
			testPart += "."
		}

		if t.CountTokens(testPart) > maxTokens {
			if currentPart != "" {
				parts = append(parts, currentPart)
				currentPart = ""
			}

			// If a single sentence is still too long, split by words
			if t.CountTokens(sentence) > maxTokens {
				wordParts, err := t.splitBySentenceWords(sentence, maxTokens)
				if err != nil {
					return nil, err
				}
				parts = append(parts, wordParts...)
			} else {
				currentPart = sentence
				if i < len(sentences)-1 {
					currentPart += "."
				}
			}
		} else {
			currentPart = testPart
		}
	}

	if currentPart != "" {
		parts = append(parts, currentPart)
	}

	return parts, nil
}

// splitBySentenceWords splits a sentence by words when it's too long
func (t *BaseTokenizer) splitBySentenceWords(sentence string, maxTokens int) ([]string, error) {
	words := strings.Fields(sentence)
	var parts []string
	currentPart := ""

	for _, word := range words {
		testPart := currentPart
		if testPart != "" {
			testPart += " "
		}
		testPart += word

		if t.CountTokens(testPart) > maxTokens {
			if currentPart != "" {
				parts = append(parts, currentPart)
				currentPart = word
			} else {
				// Single word is too long - this shouldn't happen in practice
				// but we'll include it anyway
				parts = append(parts, word)
			}
		} else {
			currentPart = testPart
		}
	}

	if currentPart != "" {
		parts = append(parts, currentPart)
	}

	return parts, nil
}

// EstimateMessageTokens provides a base implementation for message token estimation
func (t *BaseTokenizer) EstimateMessageTokens(role, content string) int {
	// Add overhead for role and message structure
	roleOverhead := 4 // Approximate tokens for role formatting
	return t.CountTokens(content) + roleOverhead
}
