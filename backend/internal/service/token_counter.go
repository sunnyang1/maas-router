package service

import (
	"context"
	"strings"
	"unicode/utf8"
)

// TokenCounter provides local token counting capabilities
type TokenCounter interface {
	// CountTokens counts tokens in a text string
	// Uses a fast estimation algorithm (chars / 4 for English, chars / 2 for CJK)
	CountTokens(text string) int

	// CountMessagesTokens counts tokens for a list of messages
	CountMessagesTokens(messages []Message) int

	// CountExactTokens attempts exact counting using tiktoken if available
	// Falls back to estimation if tiktoken is not available
	CountExactTokens(text string, model string) (int, error)
}

// Message represents a chat message for token counting
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type tokenCounter struct{}

// NewTokenCounter creates a new TokenCounter instance
func NewTokenCounter() TokenCounter {
	return &tokenCounter{}
}

// CountTokens counts tokens in a text string using a fast estimation algorithm.
// CJK characters typically use 1-2 tokens per character.
// Latin characters typically use 4 characters per token.
func (tc *tokenCounter) CountTokens(text string) int {
	if text == "" {
		return 0
	}
	// CJK characters typically use 1-2 tokens per character
	// Latin characters typically use 4 characters per token
	cjkCount := 0
	latinCount := 0
	for _, r := range text {
		if isCJK(r) {
			cjkCount++
		} else {
			latinCount++
		}
	}
	// Rough estimation: CJK ~1.5 tokens/char, Latin ~4 chars/token
	return cjkCount*3/2 + latinCount/4 + 1
}

// isCJK checks if a rune is a CJK character
func isCJK(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) || // CJK Unified Ideographs
		(r >= 0x3400 && r <= 0x4DBF) || // CJK Extension A
		(r >= 0x3000 && r <= 0x303F) || // CJK Symbols
		(r >= 0xFF00 && r <= 0xFFEF) || // Fullwidth Forms
		(r >= 0xAC00 && r <= 0xD7AF) || // Korean Hangul
		(r >= 0x3040 && r <= 0x309F) || // Japanese Hiragana
		(r >= 0x30A0 && r <= 0x30FF) // Japanese Katakana
}

// CountMessagesTokens counts tokens for a list of messages.
// Each message has ~4 overhead tokens for role and delimiters,
// plus 3 priming tokens for the conversation format.
func (tc *tokenCounter) CountMessagesTokens(messages []Message) int {
	total := 0
	for _, msg := range messages {
		total += 4 // overhead per message (~role, delimiter)
		total += tc.CountTokens(msg.Content)
	}
	total += 3 // priming tokens
	return total
}

// CountExactTokens attempts exact counting using tiktoken if available.
// Falls back to estimation if tiktoken is not available.
func (tc *tokenCounter) CountExactTokens(text string, model string) (int, error) {
	// For now, fall back to estimation
	// TODO: integrate tiktoken-go when available
	_ = context.Background()
	_ = strings.TrimSpace
	_ = utf8.RuneCountInString
	return tc.CountTokens(text), nil
}
