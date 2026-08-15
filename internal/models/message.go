package models

import (
	"time"
)

type Message struct {
	ChatID          string
	MessageID       string
	ThreadID        *string
	Content         string
	MessageTime     time.Time
	ChatName        string
	ChatDescription *string
	SenderID        string
	SenderName      string
	AccountID       string
	CreatedAt       time.Time
	Domain          []string
	Entities        []string
	Intent          []string
	IsAIHandled     bool
}

type Semantic struct {
	Domain   []string
	Entities []string
	Intent   []string
}

type CreateMessage struct {
	ChatID          string
	MessageID       string
	ThreadID        string
	Content         string
	MessageTime     time.Time
	ChatName        string
	ChatDescription *string
	SenderID        string
	SenderName      string
	AccountID       string
}
