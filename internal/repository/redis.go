package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"openai/internal/entity"
	"sort"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type RedisRepository struct {
	*redis.Client
	ctx context.Context
}

func NewRedis(ctx context.Context, c *redis.Client) *RedisRepository {
	return &RedisRepository{c, ctx}
}

// GetConversation returns a conversation by chat id.
func (r *RedisRepository) GetConversation(id int64) (*entity.Conversation, error) {
	var c entity.Conversation

	val, err := r.Get(r.ctx, fmt.Sprintf("conversation:%d", id)).Bytes()
	switch {
	case err == redis.Nil:
		return nil, ErrConversationNotFound
	case err != nil:
		return nil, fmt.Errorf("failed to get conversation: %v", err)
	default:
		if err := json.Unmarshal(val, &c); err != nil {
			return nil, fmt.Errorf("failed to unmarshal conversation into entity.Conversation: %v", err)
		}

		// get messages for conversation
		messages, err := r.GetConversationMessages(id)
		if err != nil {
			return nil, fmt.Errorf("failed to get conversation messages: %v", err)
		}
		c.Messages = messages

		return &c, nil
	}
}

// GetConversationMessages returns all messages for a given conversation.
func (r *RedisRepository) GetConversationMessages(cid int64) ([]*entity.Message, error) {
	messages := []*entity.Message{}

	pattern := fmt.Sprintf("messages:%d:*", cid)
	keys, err := r.Keys(r.ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	for _, key := range keys {
		value, err := r.Get(r.ctx, key).Result()
		if err != nil {
			return nil, err
		}

		var message entity.Message
		err = json.Unmarshal([]byte(value), &message)
		if err != nil {
			return nil, err
		}
		messages = append(messages, &message)
	}

	// sort messages in ascending order by created_at
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].CreatedAt.Before(messages[j].CreatedAt)
	})

	return messages, nil
}

// DeleteMessages deletes all messages for a given conversation.
func (r *RedisRepository) DeleteMessages(cid int64) error {
	return r.Del(r.ctx, fmt.Sprintf("messages:%d:*", cid)).Err()
}

// CreateMessage saves a message.
func (r *RedisRepository) CreateMessage(m *entity.Message) error {
	// generate unique UUID v4 for message
	u := uuid.New().String()
	if u == "" {
		return fmt.Errorf("failed to generate UUID")
	}

	s := r.Set(r.ctx, fmt.Sprintf("messages:%d:%s", m.ChatID, u), m, 0)
	return s.Err()
}

// SaveConversation saves a conversation.
func (r *RedisRepository) CreateConversation(c *entity.Conversation) error {
	s := r.Set(r.ctx, fmt.Sprintf("conversation:%d", c.ChatID), c, 0)
	return s.Err()
}
