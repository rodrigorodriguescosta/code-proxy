.PHONY: build frontend backend dev clean

# Build everything: frontend → embed → backend binary
build: frontend backend

# Build frontend and copy to embed dir
frontend:
	cd frontend && npm install && npm run build

# Build backend binary (assumes frontend already built)
backend:
	cd backend && go build -o ../code-proxy .

# Dev mode: hot-reload backend (frontend must be built first)
dev:
	cd backend && air

# Dev frontend with hot-reload (proxies API to backend)
dev-frontend:
	cd frontend && npm run dev

# Clean build artifacts
clean:
	rm -f code-proxy
	rm -rf backend/embed/dist
	rm -rf backend/tmp
