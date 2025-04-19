.PHONY: dev prod down-dev down-prod clean build push help init-env logs-api logs-book logs-category logs-user logs-all logs-api-f logs-book-f logs-category-f logs-user-f logs-all-f install-dbmate migrate-dev migrate-prod migrate-create proto

DOCKER_COMPOSE_DEV = docker-compose -f deployment/docker/docker-compose.dev.yml --env-file .env.dev
DOCKER_COMPOSE_PROD = docker-compose -f deployment/docker/docker-compose.yml --env-file .env.prod
VERSION ?= latest
REGISTRY ?= fairuzald
MIGRATION_NAME ?= migration

include .env.dev
export $(shell sed 's/=.*//' .env.dev)

help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  init-env       Create environment files from templates if they don't exist"
	@echo "  install-dbmate Install dbmate for database migrations"
	@echo "  migrate-dev    Run database migrations for development environment"
	@echo "  migrate-prod   Run database migrations for production environment"
	@echo "  migrate-create Create a new migration file (use MIGRATION_NAME=name)"
	@echo "  dev            Start development environment"
	@echo "  prod           Start production environment"
	@echo "  down-dev       Stop development environment"
	@echo "  down-prod      Stop production environment"
	@echo "  clean          Remove all containers, networks, and volumes"
	@echo "  build          Build all Docker images"
	@echo "  push           Push all Docker images to registry"
	@echo "  logs-api       View API Gateway logs"
	@echo "  logs-book      View Book service logs"
	@echo "  logs-category  View Category service logs"
	@echo "  logs-user      View User service logs"
	@echo "  logs-all       View all logs"
	@echo "  logs-*-f       Follow logs (use -f suffix, e.g. logs-api-f)"
	@echo "  help           Show this help message"

init-env:
	@if [ ! -f .env.dev ]; then \
		cp .env.template .env.dev; \
		echo ".env.dev created from template"; \
	fi
	@if [ ! -f .env.prod ]; then \
		cp .env.template .env.prod; \
		echo ".env.prod created from template"; \
	fi
	@echo "Remember to update your environment files with secure values before use"

install-dbmate:
	@if command -v dbmate >/dev/null 2>&1; then \
		echo "dbmate is already installed"; \
	else \
		echo "Installing dbmate..."; \
		if [ "$(shell uname)" = "Darwin" ]; then \
			brew install dbmate || curl -fsSL -o /usr/local/bin/dbmate https://github.com/amacneil/dbmate/releases/latest/download/dbmate-darwin-amd64 && chmod +x /usr/local/bin/dbmate; \
		elif [ "$(shell uname)" = "Linux" ]; then \
			sudo curl -fsSL -o /usr/local/bin/dbmate https://github.com/amacneil/dbmate/releases/latest/download/dbmate-linux-amd64 && sudo chmod +x /usr/local/bin/dbmate; \
		else \
			echo "Please install dbmate manually from https://github.com/amacneil/dbmate/releases"; \
			exit 1; \
		fi; \
		echo "dbmate installed successfully"; \
	fi

migrate-dev:
	@echo "Running book service migrations..."
	DATABASE_URL=postgres://$(BOOK_DB_USER):$(BOOK_DB_PASSWORD)@localhost:$(BOOK_DB_EXPOSE_PORT)/$(BOOK_DB_NAME)?sslmode=$(DB_SSLMODE) dbmate -d migrations/book up
	@echo "Running category service migrations..."
	DATABASE_URL=postgres://$(CATEGORY_DB_USER):$(CATEGORY_DB_PASSWORD)@localhost:$(CATEGORY_DB_EXPOSE_PORT)/$(CATEGORY_DB_NAME)?sslmode=$(DB_SSLMODE) dbmate -d migrations/category up
	@echo "Running user service migrations..."
	DATABASE_URL=postgres://$(USER_DB_USER):$(USER_DB_PASSWORD)@localhost:$(USER_DB_EXPOSE_PORT)/$(USER_DB_NAME)?sslmode=$(DB_SSLMODE) dbmate -d migrations/user up

migrate-prod:
	@echo "Running book service migrations..."
	DATABASE_URL=postgres://$(BOOK_DB_USER):$(BOOK_DB_PASSWORD)@localhost:$(BOOK_DB_EXPOSE_PORT)/$(BOOK_DB_NAME)?sslmode=$(DB_SSLMODE) dbmate -d migrations/book up
	@echo "Running category service migrations..."
	DATABASE_URL=postgres://$(CATEGORY_DB_USER):$(CATEGORY_DB_PASSWORD)@localhost:$(CATEGORY_DB_EXPOSE_PORT)/$(CATEGORY_DB_NAME)?sslmode=$(DB_SSLMODE) dbmate -d migrations/category up
	@echo "Running user service migrations..."
	DATABASE_URL=postgres://$(USER_DB_USER):$(USER_DB_PASSWORD)@localhost:$(USER_DB_EXPOSE_PORT)/$(USER_DB_NAME)?sslmode=$(DB_SSLMODE) dbmate -d migrations/user up

migrate-create:
	@if [ -z "$(SERVICE)" ]; then \
		echo "Error: SERVICE parameter is required (book, category, or user)"; \
		echo "Usage: make migrate-create MIGRATION_NAME=name SERVICE=service"; \
		exit 1; \
	fi
	@if [ -z "$(MIGRATION_NAME)" ]; then \
		echo "Error: MIGRATION_NAME parameter is required"; \
		echo "Usage: make migrate-create MIGRATION_NAME=name SERVICE=service"; \
		exit 1; \
	fi
	@if [ "$(SERVICE)" = "book" ]; then \
		DATABASE_URL=postgres://$(BOOK_DB_USER):$(BOOK_DB_PASSWORD)@localhost:$(BOOK_DB_EXPOSE_PORT)/$(BOOK_DB_NAME)?sslmode=$(DB_SSLMODE) dbmate -d migrations/book new $(MIGRATION_NAME); \
	elif [ "$(SERVICE)" = "category" ]; then \
		DATABASE_URL=postgres://$(CATEGORY_DB_USER):$(CATEGORY_DB_PASSWORD)@localhost:$(CATEGORY_DB_EXPOSE_PORT)/$(CATEGORY_DB_NAME)?sslmode=$(DB_SSLMODE) dbmate -d migrations/category new $(MIGRATION_NAME); \
	elif [ "$(SERVICE)" = "user" ]; then \
		DATABASE_URL=postgres://$(USER_DB_USER):$(USER_DB_PASSWORD)@localhost:$(USER_DB_EXPOSE_PORT)/$(USER_DB_NAME)?sslmode=$(DB_SSLMODE) dbmate -d migrations/user new $(MIGRATION_NAME); \
	else \
		echo "Error: SERVICE must be one of: book, category, user"; \
		exit 1; \
	fi

dev:
	$(DOCKER_COMPOSE_DEV) up -d

prod:
	$(DOCKER_COMPOSE_PROD) up -d

down-dev:
	$(DOCKER_COMPOSE_DEV) down

down-prod:
	$(DOCKER_COMPOSE_PROD) down

clean:
	$(DOCKER_COMPOSE_DEV) down -v --remove-orphans
	$(DOCKER_COMPOSE_PROD) down -v --remove-orphans

build:
	docker build -t $(REGISTRY)/library-api-gateway:$(VERSION) -f api-gateway/Dockerfile .
	docker build -t $(REGISTRY)/library-book-service:$(VERSION) -f services/book-service/Dockerfile .
	docker build -t $(REGISTRY)/library-category-service:$(VERSION) -f services/category-service/Dockerfile .
	docker build -t $(REGISTRY)/library-user-service:$(VERSION) -f services/user-service/Dockerfile .

push:
	docker push $(REGISTRY)/library-api-gateway:$(VERSION)
	docker push $(REGISTRY)/library-book-service:$(VERSION)
	docker push $(REGISTRY)/library-category-service:$(VERSION)
	docker push $(REGISTRY)/library-user-service:$(VERSION)

logs-api:
	$(DOCKER_COMPOSE_DEV) logs -f api-gateway

logs-book:
	$(DOCKER_COMPOSE_DEV) logs -f book-service

logs-category:
	$(DOCKER_COMPOSE_DEV) logs -f category-service

logs-user:
	$(DOCKER_COMPOSE_DEV) logs -f user-service

logs:
	$(DOCKER_COMPOSE_DEV) logs -f

sql-book:
	@echo "Connecting to book database using Docker..."
	@$(DOCKER_COMPOSE_DEV) exec book-db psql -U $(BOOK_DB_USER) -d $(BOOK_DB_NAME)

sql-category:
	@echo "Connecting to category database using Docker..."
	@$(DOCKER_COMPOSE_DEV) exec category-db psql -U $(CATEGORY_DB_USER) -d $(CATEGORY_DB_NAME)

sql-user:
	@echo "Connecting to user database using Docker..."
	@$(DOCKER_COMPOSE_DEV) exec user-db psql -U $(USER_DB_USER) -d $(USER_DB_NAME)

proto:
	@echo "generate proto..."
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/user/user.proto
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/book/book.proto
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/category/category.proto
