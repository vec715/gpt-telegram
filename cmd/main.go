package main

import (
	"context"
	"fmt"
	"openai/internal"
	"openai/internal/repository"
	"openai/pkg"
	"os"

	"cloud.google.com/go/datastore"
	"github.com/getsentry/sentry-go"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slog"
)

// initSentry initializes sentry if the DSN is set
func initSentry(dsn string) {
	err := sentry.Init(sentry.ClientOptions{
		Dsn: dsn,
		// Set TracesSampleRate to 1.0 to capture 100%
		// of transactions for performance monitoring.
		// We recommend adjusting this value in production,
		TracesSampleRate: 1.0,
	})
	if err != nil {
		slog.Error("sentry.Init", err)
		os.Exit(1)
	}
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout))

	cfg, err := internal.NewConfig()
	if err != nil {
		logger.Error("failed to load config", err)
		os.Exit(1)
	}

	// init sentry if DSN is set
	if cfg.Sentry.DSN != "" {
		initSentry(cfg.Sentry.DSN)
	}

	var repo internal.Repository

	// init google cloud project resources if GCP is enabled
	// otherwise use redis
	if cfg.UseGCP {
		ds, err := datastore.NewClient(context.Background(), cfg.GCP.ProjectID)
		if err != nil {
			logger.Error("failed to create datastore client", err)
			os.Exit(1)
		}
		repo = repository.NewDatastore(ds)
	} else {
		repo = repository.NewRedis(context.Background(), redis.NewClient(&redis.Options{
			Addr:     cfg.Redis.Addr,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		}))
	}

	// init openai client and telegram bot
	gpt := pkg.NewOpenAI(cfg.OpenAI.Token)
	bot, err := tgbotapi.NewBotAPI(cfg.Telegram.BotToken)
	bot.Debug = cfg.Telegram.Debug
	if err != nil {
		sentry.CaptureException(fmt.Errorf("failed to create bot: %w", err))
		logger.Error("failed to create bot", err)
		os.Exit(1)
	}

	// run service
	s := internal.NewService(cfg, logger, bot, repo, gpt)
	panic(s.Run())
}
