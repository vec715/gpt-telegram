package repository

import (
	"context"
	"fmt"
	"openai/internal/entity"

	"cloud.google.com/go/datastore"
)

type DatastoreRepository struct {
	*datastore.Client
}

func NewDatastore(ds *datastore.Client) *DatastoreRepository {
	return &DatastoreRepository{ds}
}

// GetConversation returns a conversation by chat id.
func (ds *DatastoreRepository) GetConversation(id int64) (*entity.Conversation, error) {
	q := datastore.NewQuery("conversation").Filter("chat_id =", id).Limit(1)

	var c []entity.Conversation
	if _, err := ds.GetAll(context.Background(), q, &c); err != nil {
		return nil, err
	}

	if len(c) == 0 {
		return nil, ErrConversationNotFound
	}

	// get messages
	messages, err := ds.getConversationMessages(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation messages: %w", err)
	}
	c[0].Messages = messages

	return &c[0], nil
}

// GetConversationMessages returns all messages for a given conversation.
func (ds *DatastoreRepository) getConversationMessages(cid int64) ([]*entity.Message, error) {
	q := datastore.NewQuery("messages").
		Filter("chat_id =", cid).
		Order("-created_at")

	var messages []*entity.Message
	if _, err := ds.GetAll(context.Background(), q, &messages); err != nil {
		return nil, err
	}

	return messages, nil
}

// DeleteMessages deletes all messages for a given conversation.
func (ds *DatastoreRepository) DeleteMessages(cid int64) error {
	q := datastore.NewQuery("messages").
		Filter("chat_id =", cid).
		KeysOnly()

	keys, err := ds.GetAll(context.Background(), q, nil)
	if err != nil {
		return err
	}

	if err := ds.DeleteMulti(context.Background(), keys); err != nil {
		return err
	}

	return nil
}

// CreateConversation creates a new message for a given conversation.
func (ds *DatastoreRepository) CreateMessage(m *entity.Message) error {
	key := datastore.IncompleteKey("messages", nil)
	if _, err := ds.Put(context.Background(), key, m); err != nil {
		return fmt.Errorf("failed to add conversation message: %w", err)
	}

	return nil
}

// CreateConversation creates a new conversation.
func (ds *DatastoreRepository) CreateConversation(c *entity.Conversation) error {
	key := datastore.IncompleteKey("conversation", nil)
	if _, err := ds.Put(context.Background(), key, c); err != nil {
		return fmt.Errorf("failed to put conversation: %w", err)
	}

	return nil
}
