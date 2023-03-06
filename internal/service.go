package internal

import (
	"errors"
	"fmt"
	"net/http"
	"openai/internal/entity"
	"openai/internal/repository"
	"openai/pkg"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	gogpt "github.com/sashabaranov/go-gpt3"
	"golang.org/x/exp/slog"
)

type Repository interface {
	GetConversation(id int64) (*entity.Conversation, error)
	CreateConversation(c *entity.Conversation) error
	DeleteMessages(cid int64) error
	CreateMessage(m *entity.Message) error
}

type Service struct {
	cfg    *Config
	log    *slog.Logger
	bot    *tgbotapi.BotAPI
	repo   Repository
	openai *pkg.OpenAI
}

func NewService(cfg *Config, log *slog.Logger, bot *tgbotapi.BotAPI, repo Repository, openai *pkg.OpenAI) *Service {
	return &Service{cfg, log, bot, repo, openai}
}

// HandleTelegramMessage handles a message from a user
func (s *Service) HandleTelegramMessage(update *tgbotapi.Update) error {
	var (
		id    int64
		title string
	)

	// check if the message is from a group or a private chat
	if update.Message.Chat.IsGroup() {
		id = update.Message.Chat.ID
		title = update.Message.Chat.Title
	} else {
		id = update.Message.From.ID
		title = update.Message.From.UserName
	}

	s.log.Debug("received message from user", slog.String("title", title), slog.Int64("id", id))
	// send typing action
	// since typing actions are only valid for 3 seconds, we send a new one every 3 seconds in a goroutine until the message is complete
	_, err := s.bot.Request(tgbotapi.NewChatAction(id, tgbotapi.ChatTyping))
	if err != nil {
		s.log.Error("failed to send typing action", err)
		sentry.CaptureException(fmt.Errorf("failed to send typing action: %w", err))
	}
	ticker := time.NewTicker(3 * time.Second)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				ticker.Stop()
				return
			case <-ticker.C:
				_, err := s.bot.Request(tgbotapi.NewChatAction(id, tgbotapi.ChatTyping))
				if err != nil {
					s.log.Error("failed to send typing action", err)
					sentry.CaptureException(fmt.Errorf("failed to send typing action: %w", err))
				}
			}
		}
	}()

	// get conversation from datastore
	// if conversation is not found, create a new one
	c, err := s.repo.GetConversation(id)
	if err != nil {
		if errors.Is(err, repository.ErrConversationNotFound) {
			if err := s.repo.CreateConversation(&entity.Conversation{
				ChatID:    id,
				Title:     title,
				CreatedAt: time.Now(),
				Messages:  []*entity.Message{},
			}); err != nil {
				return fmt.Errorf("failed to create conversation: %w", err)
			}
		} else {
			return fmt.Errorf("failed to get conversation: %w", err)
		}
	}

	// create message object and add it to the conversation
	m := &entity.Message{
		ChatID:     id,
		CreatedAt:  time.Now(),
		IsResponse: false,
		Text:       update.Message.Text,
	}
	c.Messages = append(c.Messages, m)

	// create completion request from conversation messages
	prompt := c.ToGPTMessages()

	// generate GPT response
	r, err := s.openai.CreateCompletionRequest(title, prompt)
	if err != nil {
		// if the error is "Please reduce the length of the messages", delete all messages from the conversation and try again
		// this is a workaround for the 4096 token limit on the prompt. Read more here: https://platform.openai.com/docs/models/gpt-3-5
		// Currently we are deleting all messages from the conversation, but this can be improved by only deleting the few old messages
		// Should be fixed in the future by integrating the BPE tokeniser like this one: https://github.com/openai/tiktoken
		if strings.Contains(err.Error(), "Please reduce the length of the messages") {
			if err := s.repo.DeleteMessages(id); err != nil {
				return fmt.Errorf("failed to delete messages: %w", err)
			}

			// generate GPT response again without the old messages history
			r, err = s.openai.CreateCompletionRequest(title, []gogpt.ChatCompletionMessage{m.ToGPTMessage()})
			if err != nil {
				return fmt.Errorf("failed to create completion request: %w", err)
			}
		} else {
			return fmt.Errorf("failed to generate response: %w", err)
		}
	}

	// stop typing action goroutine
	done <- struct{}{}

	// This code splits the message into pieces no longer than 4096 characters if the message is longer than this threshold.
	// Each part is then sent to the telegram in markdown mode.
	// However, there is a pitfall in this logic: if the message you are trying to send using this code contains characters
	// that can generate Markdown markup, then they may violate the formatting of the message when it is split into parts.
	// It will also cause an API error when sending telegram message in markdown mode:
	// failed to send message: Bad Request: can't parse entities: Can't find end of the entity starting at byte offset 1110
	if len(r) > 4096 {
		s.log.Debug("message is too long, splitting it into multiple messages", slog.Int("length", len(r)))
		for i := 0; i < len(r); i += 4096 {
			for i := 0; i < len(r); i += 4096 {
				end := i + 4096
				if end > len(r) {
					end = len(r)
				}
				msg := tgbotapi.NewMessage(id, r[i:end])
				if _, err := s.bot.Send(msg); err != nil {
					return fmt.Errorf("failed to send message: %w", err)
				}
			}
		}
	} else {
		msg := tgbotapi.NewMessage(id, r)
		msg.ParseMode = tgbotapi.ModeMarkdown
		if _, err := s.bot.Send(msg); err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}
	}

	// save user question in datastore
	if err := s.repo.CreateMessage(m); err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}

	// save gpt response in datastore
	if err := s.repo.CreateMessage(&entity.Message{
		ChatID:     id,
		CreatedAt:  time.Now(),
		IsResponse: true,
		Text:       r,
	}); err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}

	return nil
}

// Run starts the telegram bot
// It sets up the webhook and listens for updates
func (s *Service) Run() error {
	// start listening for updates
	var updates tgbotapi.UpdatesChannel
	var err error

	// if webhook is enabled, start the webhook server, otherwise start the long polling mode.
	if s.cfg.Telegram.UseWebhook {
		updates, err = s.RunWebhook()
		if err != nil {
			return fmt.Errorf("failed to run webhook: %w", err)
		}
	} else {
		updates = s.RunLongPolling()
	}

	for update := range updates {
		if update.Message == nil { // ignore any non-Message updates
			s.log.Debug("received non-message update", slog.Any("update", update))
			continue
		}

		if err := s.HandleTelegramMessage(&update); err != nil {
			s.log.Error("failed to handle message", err)
			sentry.CaptureException(fmt.Errorf("failed to handle message: %w", err))
		}
	}
	return nil
}

// RunLongPolling starts the telegram bot in long polling mode
func (s *Service) RunLongPolling() tgbotapi.UpdatesChannel {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	return s.bot.GetUpdatesChan(u)
}

// RunWebhook starts the telegram bot in webhook mode
func (s *Service) RunWebhook() (tgbotapi.UpdatesChannel, error) {
	updates := s.bot.ListenForWebhook("/" + s.bot.Token)

	go http.ListenAndServe(":"+s.cfg.HTTPPort, nil)

	// sleep for 5 seconds to make sure the server is set up before listening for updates
	time.Sleep(5)

	// setup webhook
	wh, _ := tgbotapi.NewWebhook(s.cfg.Telegram.WebhookURL + "/" + s.bot.Token)
	_, err := s.bot.Request(wh)
	if err != nil {
		return nil, fmt.Errorf("failed to set webhook: %w", err)
	}

	s.log.Debug("start listening for updates", slog.String("port", s.cfg.HTTPPort))
	return updates, nil
}
