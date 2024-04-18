.PHONY: remove down quick-start dev debug

production:
	docker compose --profile production up

dev:
	docker compose --profile development up --build

debug:
	docker compose --profile debugging up --build

down:
	docker compose --profile production down
	docker compose --profile development down
	docker compose --profile debugging down

remove:
	docker compose --profile production down --volumes
	docker compose --profile development down --volumes
	docker compose --profile debugging down --volumes
