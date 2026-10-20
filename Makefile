.PHONY: help setup run test cover fmt vet tidy web-install web-dev web-build up down logs clean

help: ## 显示帮助
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

setup: ## 安装前后端依赖
	cd server && go mod download
	cd web && npm install

run: ## 启动后端（默认 :8088）
	cd server && go run ./cmd/server

test: ## 运行全部 Go 测试
	cd server && go test ./... -count=1

cover: ## 生成覆盖率报告
	cd server && go test ./... -coverpkg=./internal/... -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1

fmt: ## 格式化 Go 代码
	cd server && gofmt -w ./cmd ./internal ./tests

vet: ## 静态检查
	cd server && go vet ./...

tidy: ## 整理依赖
	cd server && go mod tidy

web-install: ## 安装前端依赖
	cd web && npm install

web-dev: ## 启动前端开发服务器（:7802）
	cd web && NODE_OPTIONS=--openssl-legacy-provider npm run serve

web-build: ## 构建前端
	cd web && NODE_OPTIONS=--openssl-legacy-provider npm run build

up: ## docker compose 启动 mysql + server + web
	docker compose up -d --build

down: ## 停止并删除容器
	docker compose down

logs: ## 查看服务日志
	docker compose logs -f server

clean: ## 清理构建产物
	rm -rf server/bin server/coverage.out web/dist
