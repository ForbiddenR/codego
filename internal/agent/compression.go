package agent

import (
	"strings"

	"github.com/nice-code/codego/internal/types"
)

// CompressionConfig controls context compression behavior.
type CompressionConfig struct {
	// Threshold is the ratio of estimated tokens to max context at which to trigger compression.
	Threshold float64
	// TargetRatio is how much of the context to keep after compression.
	TargetRatio float64
	// MaxMessagesToKeep is minimum messages to preserve (never compress below this).
	MaxMessagesToKeep int
}

// DefaultCompressionConfig returns sensible defaults.
func DefaultCompressionConfig() CompressionConfig {
	return CompressionConfig{
		Threshold:         0.70,
		TargetRatio:       0.30,
		MaxMessagesToKeep: 4,
	}
}

// EstimateTokens returns an approximate token count for a string.
// Uses the rough heuristic: ~4 characters per token for English text.
func EstimateTokens(text string) int {
	return (len(text) + 3) / 4
}

// EstimateMessageTokens estimates tokens for a single message.
func EstimateMessageTokens(msg types.Message) int {
	tokens := 4 // role overhead
	for _, block := range msg.Content {
		switch {
		case block.IsText():
			tokens += EstimateTokens(block.Text)
		case block.IsToolUse():
			tokens += EstimateTokens(block.Name)
			tokens += 20 // ID + structure overhead
			for k, v := range block.Input {
				tokens += EstimateTokens(k)
				tokens += EstimateTokens(toString(v))
			}
		case block.IsToolResult():
			tokens += EstimateTokens(block.Content)
			tokens += 10 // ID + structure
		case block.IsThinking():
			tokens += EstimateTokens(block.Text)
		}
	}
	return tokens
}

// EstimateConversationTokens estimates total tokens for a conversation.
func EstimateConversationTokens(msgs []types.Message) int {
	total := 0
	for _, msg := range msgs {
		total += EstimateMessageTokens(msg)
	}
	return total
}

// ShouldCompress returns true if the conversation needs compression.
func ShouldCompress(msgs []types.Message, maxTokens int, cfg CompressionConfig) bool {
	if len(msgs) <= cfg.MaxMessagesToKeep {
		return false
	}
	estimated := EstimateConversationTokens(msgs)
	threshold := int(float64(maxTokens) * cfg.Threshold)
	return estimated > threshold
}

// CompressMessages reduces the conversation size by summarizing older messages.
// It preserves the most recent messages and replaces older ones with a summary.
//
// This is a simple version that keeps system messages and the last N messages.
// A more sophisticated version would use the API to generate summaries.
func CompressMessages(msgs []types.Message, cfg CompressionConfig) []types.Message {
	if len(msgs) <= cfg.MaxMessagesToKeep {
		return msgs
	}

	// Count how many messages to keep at the end
	keepLast := cfg.MaxMessagesToKeep
	if keepLast > len(msgs) {
		keepLast = len(msgs)
	}

	// Find the cutoff point
	cutoff := len(msgs) - keepLast

	// Collect the messages we're removing
	removed := msgs[:cutoff]
	kept := msgs[cutoff:]

	// Build a summary of removed messages
	summary := summarizeMessages(removed)

	// Create a summary message
	result := make([]types.Message, 0, len(kept)+1)
	if summary != "" {
		result = append(result, types.NewSystemMessage("Earlier conversation summary: "+summary))
	}
	result = append(result, kept...)

	return result
}

// summarizeMessages creates a brief summary of messages.
func summarizeMessages(msgs []types.Message) string {
	var userCount, assistantCount, toolCount int
	var topics []string

	for _, msg := range msgs {
		switch msg.Role {
		case types.RoleUser:
			userCount++
			// Extract first few words as topic
			text := msg.TextContent()
			if text != "" && len(topics) < 3 {
				words := strings.Fields(text)
				if len(words) > 5 {
					words = words[:5]
				}
				topics = append(topics, strings.Join(words, " ")+"...")
			}
		case types.RoleAssistant:
			assistantCount++
			toolCount += len(msg.ToolCalls())
		}
	}

	parts := []string{}
	if userCount > 0 {
		parts = append(parts, itoa(userCount)+" user messages")
	}
	if assistantCount > 0 {
		parts = append(parts, itoa(assistantCount)+" assistant responses")
	}
	if toolCount > 0 {
		parts = append(parts, itoa(toolCount)+" tool calls")
	}

	summary := strings.Join(parts, ", ")
	if len(topics) > 0 {
		summary += ". Topics discussed: " + strings.Join(topics, "; ")
	}
	return summary
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return "" // don't use fmt.Sprintf for performance
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
