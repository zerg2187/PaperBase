.PHONY: dev test lint build clean install stop

# =============================================================================
# 開発サーバー起動
# =============================================================================
dev:
	@echo "Starting backend and frontend in parallel..."
	@echo "Backend: http://localhost:8080"
	@echo "Frontend: http://localhost:5173"
	@echo "Press Ctrl+C to stop both"
	@(trap 'kill 0' EXIT; go run main.go & cd frontend && npm run dev & wait)

# =============================================================================
# テスト
# =============================================================================
test:
	@echo "Running backend tests..."
	go test ./...
	@echo "Running frontend tests..."
	cd frontend && npm run test:run

# 統合テスト（ローカルでバックエンド起動時のみ）
test-integration:
	cd frontend && npm run test:integration

# =============================================================================
# リント
# =============================================================================
lint:
	@echo "Running go vet..."
	go vet ./...
	@echo "Running ESLint..."
	cd frontend && npm run lint

# =============================================================================
# ビルド
# =============================================================================
build:
	@echo "Building frontend..."
	cd frontend && npm run build

# =============================================================================
# 依存関係インストール
# =============================================================================
install:
	@echo "Installing Go dependencies..."
	go mod download
	@echo "Installing frontend dependencies..."
	cd frontend && npm install

# =============================================================================
# クリーン
# =============================================================================
clean:
	@echo "Cleaning build artifacts..."
	rm -rf frontend/dist
	rm -rf frontend/node_modules
