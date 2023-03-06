package internal

import "github.com/kelseyhightower/envconfig"

type (
	Config struct {
		UseGCP   bool   `envconfig:"USE_GCP" default:"false"` // use GCP Datastore and Cloud Storage for persistent storage or Redis
		HTTPPort string `envconfig:"HTTP_PORT" default:"8080"`
		OpenAI
		Telegram
		Redis
		GCP
		Sentry
	}

	OpenAI struct {
		Token string `envconfig:"OPEN_AI_TOKEN" required:"true"`
	}

	Telegram struct {
		BotToken   string `envconfig:"TELEGRAM_BOT_TOKEN" required:"true"`
		UseWebhook bool   `envconfig:"TELEGRAM_USE_WEBHOOK" default:"false"` // use webhook or long polling
		WebhookURL string `envconfig:"TELEGRAM_WEBHOOK_URL"`
		Debug      bool   `envconfig:"TELEGRAM_DEBUG" default:"false"`
	}

	Redis struct {
		Addr     string `envconfig:"REDIS_ADDR" default:"redis:6379"`
		Password string `envconfig:"REDIS_PASSWORD" default:""`
		DB       int    `envconfig:"REDIS_DB" default:"0"`
	}

	GCP struct {
		ProjectID          string `envconfig:"GCP_PROJECT_ID" default:"openai-telegram"`
		CredetialsFilePath string `envconfig:"GCP_CREDENTIALS" default:"credentials.json"`
	}

	Sentry struct {
		DSN string `envconfig:"SENTRY_DSN"`
	}
)

func NewConfig() (*Config, error) {
	var c Config
	if err := envconfig.Process("", &c); err != nil {
		return nil, err
	}
	return &c, nil
}
