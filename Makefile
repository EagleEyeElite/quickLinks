.PHONY: check-env remove down quick-start dev debug

# Check if .env exists, if not, copy from .env.example
check-env:
	@if [ ! -f .env ]; then \
		echo "No .env found. Creating from .env.example..."; \
		cp .env.example .env; \
	else \
		echo ".env exists, proceeding..."; \
	fi

production: check-env
	docker compose --profile production up

dev: check-env
	docker compose --profile development up --build

debug: check-env
	docker compose --profile debugging up --build

down:
	docker compose --profile production down
	docker compose --profile development down
	docker compose --profile debugging down

clean:
	docker compose --profile production down --volumes
	docker compose --profile development down --volumes
	docker compose --profile debugging down --volumes
