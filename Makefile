include .env
export $(shell sed 's/=.*//' .env)

# Директория, в которой лежат миграции
MIGRATIONS_DIR ?= migrations

# Команда для запуска Goose
GOOSE_CMD    ?= goose -dir $(MIGRATIONS_DIR) \
                    postgres "user=$(DB_USER) password=$(DB_PASSWORD) host=$(DB_HOST) port=$(DB_PORT) dbname=$(DB_NAME) sslmode=$(DB_SSLMODE)"

.PHONY: goose-up
goose-up:
	@$(GOOSE_CMD) up

.PHONY: goose-down
goose-down:
	@$(GOOSE_CMD) down

# Применить миграции до указанной версии
# Пример: make goose-down-to VERSION=20231010103044
.PHONY: goose-down-to
goose-down-to:
	@[ -n "$(VERSION)" ] || (echo "Необходимо указать VERSION, например make goose-down-to VERSION=20231010103044"; exit 1)
	@$(GOOSE_CMD) down-to $(VERSION)

.PHONY: goose-status
goose-status:
	@$(GOOSE_CMD) status

# Создать новую миграцию
# Пример: make goose-create NAME=add_users_table
.PHONY: goose-create
goose-create:
	@[ -n "$(NAME)" ] || (echo "Необходимо указать NAME, например make goose-create NAME=add_users_table"; exit 1)
	@$(GOOSE_CMD) create $(NAME) sql

.PHONY: build
build:
	go build ./cmd/app/main.go