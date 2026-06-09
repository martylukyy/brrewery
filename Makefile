# brrewery Makefile

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT := $(shell git rev-parse HEAD 2>/dev/null || echo "")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
BINARY_NAME = brrewery
BUILD_DIR = build
WEB_DIR = web
INTERNAL_WEB_DIR = internal/web

LDFLAGS = -ldflags "-X github.com/autobrr/brrewery/internal/buildinfo.Version=$(VERSION) -X github.com/autobrr/brrewery/internal/buildinfo.Commit=$(GIT_COMMIT) -X github.com/autobrr/brrewery/internal/buildinfo.Date=$(BUILD_DATE)"

PROD_BIN = /usr/local/bin/$(BINARY_NAME)
PROD_WEB_ROOT = /var/www/brrewery

.PHONY: all build frontend backend prod dev dev-backend dev-frontend clean test test-openapi lint lint-full lint-json lint-fix fmt gofix-changed gofix-check-changed precommit deps help ansible-syntax-check sync-ansible

all: build sync-ansible

build: frontend backend

frontend:
	@echo "Building frontend..."
	cd $(WEB_DIR) && pnpm install && pnpm build
	@echo "Copying frontend assets..."
	rm -rf $(INTERNAL_WEB_DIR)/dist
	cp -r $(WEB_DIR)/dist $(INTERNAL_WEB_DIR)/

backend:
	@echo "Building backend..."
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/brrewery

prod: build sync-ansible
	@echo "Installing $(BINARY_NAME) to $(PROD_BIN)..."
	sudo install -m 0755 $(BINARY_NAME) $(PROD_BIN)
	@echo "Deploying web assets to $(PROD_WEB_ROOT)..."
	sudo install -d -m 0755 $(PROD_WEB_ROOT)
	sudo rm -rf $(PROD_WEB_ROOT)/*
	sudo cp -a $(INTERNAL_WEB_DIR)/dist/. $(PROD_WEB_ROOT)/

dev:
	@echo "Starting development mode..."
	@make -j 2 dev-backend dev-frontend

dev-backend:
	@echo "Starting backend development server..."
	@mkdir -p tmp/brrewery-jobs
	BRREWERY_LISTEN_ADDR=127.0.0.1:8081 BRREWERY_ANSIBLE_ROOT=$(CURDIR)/ansible BRREWERY_JOBS_DIR=$(CURDIR)/tmp/brrewery-jobs air -c .air.toml

dev-frontend:
	@echo "Starting frontend development server..."
	cd $(WEB_DIR) && VITE_BACKEND_URL=http://127.0.0.1:8081 pnpm dev

clean:
	@echo "Cleaning..."
	rm -rf $(WEB_DIR)/dist $(INTERNAL_WEB_DIR)/dist $(BINARY_NAME) $(BUILD_DIR)

test:
	@echo "Running tests..."
	go test -race -count=1 -v ./...

test-openapi:
	@echo "Validating OpenAPI specification..."
	go test -count=1 -v ./internal/web/swagger

ansible-syntax-check:
	@echo "Checking Ansible playbooks..."
	@cd ansible && find playbooks -name '*.yml' -print0 | xargs -0 -n1 ansible-playbook --syntax-check

sync-ansible:
	@echo "Syncing ansible playbooks to /usr/share/brrewery/ansible..."
	sudo install -d -m 0755 /usr/share/brrewery/ansible
	sudo rm -rf /usr/share/brrewery/ansible/*
	sudo cp -a ansible/. /usr/share/brrewery/ansible/

fmt:
	@echo "Formatting changed Go code..."
	@gofiles=$$({ git diff --name-only --diff-filter=d; git diff --name-only --cached --diff-filter=d; } | sort -u | grep '\.go$$' || true); \
		if [ -n "$$gofiles" ]; then echo "$$gofiles" | xargs gofmt -w; fi
	@echo "Formatting changed frontend code..."
	@webfiles=$$({ git diff --name-only --diff-filter=d -- '$(WEB_DIR)/'; git diff --name-only --cached --diff-filter=d -- '$(WEB_DIR)/'; } | sort -u | sed 's|^$(WEB_DIR)/||' | grep -E '\.(ts|tsx|js|jsx)$$' || true); \
		if [ -n "$$webfiles" ]; then cd $(WEB_DIR) && echo "$$webfiles" | xargs pnpm eslint --fix; fi

gofix-changed:
	@echo "Running go fix on changed Go files..."
	@gofiles=$$({ git diff --name-only --diff-filter=d; git diff --name-only --cached --diff-filter=d; } | sort -u | grep '\.go$$' || true); \
		if [ -z "$$gofiles" ]; then echo "No changed Go files for go fix."; exit 0; fi; \
		gopkgs=$$(printf '%s\n' "$$gofiles" | xargs -n 1 dirname | sort -u); \
		printf '%s\n' "$$gopkgs" | while IFS= read -r pkg; do \
			[ -n "$$pkg" ] || continue; \
			go fix "./$$pkg" || true; \
		done

gofix-check-changed:
	@echo "Checking go fix drift on changed Go files..."
	@tmp=$$(mktemp); \
		gofiles=$$({ git diff --name-only --diff-filter=d; git diff --name-only --cached --diff-filter=d; } | sort -u | grep '\.go$$' || true); \
		if [ -z "$$gofiles" ]; then rm -f "$$tmp"; echo "No changed Go files."; exit 0; fi; \
		gopkgs=$$(printf '%s\n' "$$gofiles" | xargs -n 1 dirname | sort -u); \
		printf '%s\n' "$$gopkgs" | while IFS= read -r pkg; do \
			[ -n "$$pkg" ] || continue; \
			go fix -diff "./$$pkg" >> "$$tmp" || true; \
		done; \
		if [ -s "$$tmp" ]; then cat "$$tmp"; rm -f "$$tmp"; exit 1; fi; \
		rm -f "$$tmp"

precommit: fmt gofix-changed lint
	@echo "Pre-commit checks passed."

lint:
	@echo "Linting changed Go code..."
	golangci-lint run --new-from-rev=HEAD~1 --timeout=5m || golangci-lint run --timeout=5m
	@echo "Linting frontend..."
	cd $(WEB_DIR) && pnpm lint

lint-full:
	golangci-lint run --timeout=10m
	cd $(WEB_DIR) && pnpm lint

lint-json:
	golangci-lint run --output.json.path=./lint-report.json --timeout=5m || true

lint-fix:
	golangci-lint run --fix --timeout=10m
	cd $(WEB_DIR) && pnpm lint --fix

deps:
	go mod download
	cd $(WEB_DIR) && pnpm install

help:
	@echo "Targets: build, frontend, backend, prod, dev, test, test-openapi, lint, precommit, ansible-syntax-check"
