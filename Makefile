.PHONY: run test lint tools generate mock swagger check-generate migrate-up migrate-down migrate-create

run:
	GYM_API_PORT=8081 \
	DATABASE_URL=postgres://gym:gym_pass@localhost:5432/gym?sslmode=disable \
	REDIS_URL=redis://localhost:6379/0 \
	S3_ENDPOINT=http://localhost:9000 \
	S3_ACCESS_KEY=minioadmin \
	S3_SECRET_KEY=minioadmin \
	S3_BUCKET=gym-media \
	go run ./cmd/server

test:
	go test ./... -v -count=1

lint:
	golangci-lint run --timeout=5m

tools:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2
	go install github.com/pressly/goose/v3/cmd/goose@v3.27.3
	go install github.com/swaggo/swag/v2/cmd/swag@v2.0.0-rc5
	go install github.com/vektra/mockery/v2@v2.53.6

generate: mock swagger

mock:
	mockery

swagger:
	swag init --parseDependency -g cmd/server/main.go -o docs/

check-generate:
	$(MAKE) generate
	git diff --exit-code

migrate-up:
	goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	goose -dir migrations postgres "$(DATABASE_URL)" down

migrate-create:
	goose -dir migrations create $(NAME) sql
