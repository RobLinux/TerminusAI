package providers

import "terminusai/internal/tokenizer"

type ChatMessage struct {
	Role    string `json:"role"` // "system", "user", or "assistant"
	Content string `json:"content"`
}

type ChatOptions struct {
	Model       string  `json:"model,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
}

type CompletionOptions struct {
	Language    string `json:"language,omitempty"`
	MaxTokens   int    `json:"max_tokens,omitempty"`
	Temperature int    `json:"temperature,omitempty"`
	Suffix      string `json:"suffix,omitempty"`
}

type LLMProvider interface {
	Name() string
	DefaultModel() string
	Chat(messages []ChatMessage, opts *ChatOptions) (string, error)
	GetTokenizer() tokenizer.Tokenizer
}

type CompletionProvider interface {
	LLMProvider
	Complete(prompt string, opts *CompletionOptions) (string, error)
}

// MessageSplitter handles splitting messages that exceed token limits
type MessageSplitter struct {
	tokenizer tokenizer.Tokenizer
	maxTokens int
	model     string
}

// NewMessageSplitter creates a new message splitter
func NewMessageSplitter(tok tokenizer.Tokenizer, maxTokens int, model string) *MessageSplitter {
	if maxTokens <= 0 {
		maxTokens = tok.GetMaxContextTokens(model)
	}
	return &MessageSplitter{
		tokenizer: tok,
		maxTokens: maxTokens,
		model:     model,
	}
}

// SplitMessages splits a conversation that exceeds token limits
func (ms *MessageSplitter) SplitMessages(messages []ChatMessage) ([][]ChatMessage, error) {
	totalTokens := ms.calculateTotalTokens(messages)

	if totalTokens <= ms.maxTokens {
		return [][]ChatMessage{messages}, nil
	}

	var result [][]ChatMessage
	var currentBatch []ChatMessage
	currentTokens := 0

	// Always preserve system messages in each batch
	var systemMessages []ChatMessage
	var otherMessages []ChatMessage

	for _, msg := range messages {
		if msg.Role == "system" {
			systemMessages = append(systemMessages, msg)
		} else {
			otherMessages = append(otherMessages, msg)
		}
	}

	systemTokens := ms.calculateTotalTokens(systemMessages)
	availableTokens := ms.maxTokens - systemTokens

	if availableTokens <= 0 {
		// System messages alone exceed limit, need to split them
		systemBatches, err := ms.splitSystemMessages(systemMessages)
		if err != nil {
			return nil, err
		}
		for _, batch := range systemBatches {
			result = append(result, batch)
		}

		// Process other messages separately
		if len(otherMessages) > 0 {
			otherBatches, err := ms.SplitMessages(otherMessages)
			if err != nil {
				return nil, err
			}
			result = append(result, otherBatches...)
		}
		return result, nil
	}

	// Add system messages to current batch
	currentBatch = append(currentBatch, systemMessages...)
	currentTokens = systemTokens

	for _, msg := range otherMessages {
		msgTokens := ms.tokenizer.EstimateMessageTokens(msg.Role, msg.Content)

		if currentTokens+msgTokens > ms.maxTokens {
			// Current batch is full, start a new one
			if len(currentBatch) > len(systemMessages) {
				result = append(result, currentBatch)
			}

			// Start new batch with system messages
			currentBatch = append([]ChatMessage{}, systemMessages...)
			currentTokens = systemTokens

			// If single message is too large, split it
			if msgTokens > availableTokens {
				splitContents, err := ms.tokenizer.SplitText(msg.Content, availableTokens-ms.tokenizer.EstimateMessageTokens(msg.Role, ""))
				if err != nil {
					return nil, err
				}

				for _, content := range splitContents {
					splitMsg := ChatMessage{Role: msg.Role, Content: content}
					splitMsgTokens := ms.tokenizer.EstimateMessageTokens(splitMsg.Role, splitMsg.Content)

					if currentTokens+splitMsgTokens > ms.maxTokens {
						if len(currentBatch) > len(systemMessages) {
							result = append(result, currentBatch)
						}
						currentBatch = append([]ChatMessage{}, systemMessages...)
						currentTokens = systemTokens
					}

					currentBatch = append(currentBatch, splitMsg)
					currentTokens += splitMsgTokens
				}
			} else {
				currentBatch = append(currentBatch, msg)
				currentTokens += msgTokens
			}
		} else {
			currentBatch = append(currentBatch, msg)
			currentTokens += msgTokens
		}
	}

	// Add the final batch if it has content beyond system messages
	if len(currentBatch) > len(systemMessages) {
		result = append(result, currentBatch)
	}

	return result, nil
}

// calculateTotalTokens calculates the total tokens for a slice of messages
func (ms *MessageSplitter) calculateTotalTokens(messages []ChatMessage) int {
	total := 0
	for _, msg := range messages {
		total += ms.tokenizer.EstimateMessageTokens(msg.Role, msg.Content)
	}
	return total
}

// splitSystemMessages splits system messages if they're too large
func (ms *MessageSplitter) splitSystemMessages(systemMessages []ChatMessage) ([][]ChatMessage, error) {
	var result [][]ChatMessage

	for _, msg := range systemMessages {
		msgTokens := ms.tokenizer.EstimateMessageTokens(msg.Role, msg.Content)

		if msgTokens <= ms.maxTokens {
			result = append(result, []ChatMessage{msg})
		} else {
			// Split the system message content
			splitContents, err := ms.tokenizer.SplitText(msg.Content, ms.maxTokens-ms.tokenizer.EstimateMessageTokens(msg.Role, ""))
			if err != nil {
				return nil, err
			}

			for _, content := range splitContents {
				splitMsg := ChatMessage{Role: msg.Role, Content: content}
				result = append(result, []ChatMessage{splitMsg})
			}
		}
	}

	return result, nil
}

// Ensure CopilotProvider implements both interfaces
var _ LLMProvider = (*CopilotProvider)(nil)
var _ CompletionProvider = (*CopilotProvider)(nil)
