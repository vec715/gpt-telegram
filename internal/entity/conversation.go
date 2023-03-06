package entity

import (
	"encoding/json"
	"time"

	"cloud.google.com/go/datastore"
	gogpt "github.com/sashabaranov/go-gpt3"
)

// Message represents a single message in a conversation with a user
type Message struct {
	K          *datastore.Key `datastore:"__key__" json:"-"`       // datastore key
	RedisK     *string        `datastore:"-" json:"-"`             // redis key
	ChatID     int64          `datastore:"chat_id" json:"chat_id"` // telegram chat id
	CreatedAt  time.Time      `datastore:"created_at" json:"created_at"`
	IsResponse bool           `datastore:"is_response" json:"is_response"` // true if the message is a response from the bot
	Text       string         `datastore:"text,noindex" json:"text"`
}

func (m *Message) MarshalBinary() ([]byte, error) { return json.Marshal(m) }

// ToGPTMessage converts a Message to a gogpt.ChatCompletionMessage
func (m *Message) ToGPTMessage() gogpt.ChatCompletionMessage {
	var role string
	if m.IsResponse {
		role = "assistant"
	} else {
		role = "user"
	}

	return gogpt.ChatCompletionMessage{
		Role:    role,
		Content: m.Text,
	}
}

// Conversation represents a whole conversation with a user.
// This can be a single private chat with the bot or a group chat.
type Conversation struct {
	K         *datastore.Key `datastore:"__key__" json:"-"`
	ChatID    int64          `datastore:"chat_id" json:"chat_id"` // telegram chat id
	Title     string         `datastore:"title,noindex" json:"title"`
	CreatedAt time.Time      `datastore:"created_at" json:"created_at"`
	Messages  []*Message     `datastore:"-" json:"messages"`
}

func (c *Conversation) MarshalBinary() ([]byte, error) { return json.Marshal(c) }

// ToGPTMessages converts the conversation messages to a slice of gogpt.ChatCompletionMessage
func (c *Conversation) ToGPTMessages() []gogpt.ChatCompletionMessage {
	prompt := []gogpt.ChatCompletionMessage{}
	for _, m := range c.Messages {
		var role string
		if m.IsResponse {
			role = "assistant"
		} else {
			role = "user"
		}

		prompt = append(prompt, gogpt.ChatCompletionMessage{
			Role:    role,
			Content: m.Text,
		})
	}
	return prompt
}
