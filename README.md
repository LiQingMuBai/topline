# Topline

Topline is a Go-based Telegram bot focused on lightweight top-up workflows. The current streamlined version centers on four user-facing sections:

- mobile top-up
- data top-up
- profile center
- support contact

The project uses Telegram Bot API for interaction, GORM with MySQL for persistence, and an in-memory cache for lightweight runtime state such as pagination and temporary input flow management.

## Features

- Mobile top-up entry flow with country, plan, and mobile number selection
- Data top-up entry flow with country and plan routing
- Profile center with balance overview and deposit record pagination
- Support contact entry with working hours notice
- USDT deposit order creation and balance payment flow
- Environment-based configuration with a minimal `.env.example`

## Project Structure

```text
.
├── cmd/
│   └── main.go
├── internal/
│   ├── app/                # App bootstrap and lifecycle
│   ├── cache/              # Cache abstractions and memory cache
│   ├── config/             # .env configuration loading
│   ├── domain/             # Core domain models
│   ├── infrastructure/     # Repositories and utility helpers
│   ├── poly/               # Top-up related models and repositories
│   ├── service/
│   │   ├── order/          # Order lifecycle
│   │   ├── profile/        # Profile and support
│   │   └── topup/          # Top-up workflows
│   └── telegram/           # Router and dispatchers
├── static/                 # Static assets
└── translations/           # Translation resources
```

## Requirements

- Go 1.24+
- MySQL 8.x or compatible
- A Telegram bot token from BotFather

## Configuration

Copy the example file and fill in your own values:

```bash
cp .env.example .env
```

Required variables:

```env
TG_BOT_API=your_telegram_bot_token
MYSQL_DSN=root:password@tcp(127.0.0.1:3306)/polytopup?charset=utf8mb4&parseTime=True&loc=Local
AGENT=admin
```

Common optional variables:

```env
BOT_NAME=polytopup
TG_DEBUG=false
TOPUP_NOTIFY_CHAT_ID=0
DEFAULT_LANG=zh
TRANSLATIONS_DIR=translations
ORDER_IMAGE_PATH=./static/CCTV.png
```

## Run Locally

Install dependencies and start the bot:

```bash
go mod tidy
go run ./cmd
```

## Test

Run all tests:

```bash
go test ./...
```

The current test suite includes focused unit tests for:

- order balance payment
- top-up mobile state flow
- profile deposit record pagination

## Notes

- The bot currently ships with Chinese translation data and defaults to `zh`.
- Runtime configuration is loaded from `.env`.
- Do not commit real secrets such as Telegram tokens or production database credentials.
