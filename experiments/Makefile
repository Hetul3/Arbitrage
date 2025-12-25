DOCKER_COMPOSE ?= docker compose

run-go-hello:
	$(DOCKER_COMPOSE) run --rm --build go-hello

run-sqlite-create:
	$(DOCKER_COMPOSE) run --rm --build sqlite-create

run-sqlite-populate:
	$(DOCKER_COMPOSE) run --rm --build sqlite-populate

run-sqlite-query:
	$(DOCKER_COMPOSE) run --rm --build sqlite-query

run-sqlite-drop:
	$(DOCKER_COMPOSE) run --rm --build sqlite-drop

run-redis-cli:
	$(DOCKER_COMPOSE) up -d redis-server
	$(DOCKER_COMPOSE) run --rm --build redis-cli
	$(DOCKER_COMPOSE) stop redis-server

run-chroma-create:
	$(DOCKER_COMPOSE) up -d chromadb
	$(DOCKER_COMPOSE) run --rm --build chroma-create
	$(DOCKER_COMPOSE) stop chromadb

run-chroma-add:
	$(DOCKER_COMPOSE) up -d chromadb
	$(DOCKER_COMPOSE) run --rm --build chroma-add
	$(DOCKER_COMPOSE) stop chromadb

run-chroma-query:
	$(DOCKER_COMPOSE) up -d chromadb
	$(DOCKER_COMPOSE) run --rm --build chroma-query
	$(DOCKER_COMPOSE) stop chromadb

run-chroma-drop:
	$(DOCKER_COMPOSE) up -d chromadb
	$(DOCKER_COMPOSE) run --rm --build chroma-drop
	$(DOCKER_COMPOSE) stop chromadb

chroma-stop:
	$(DOCKER_COMPOSE) stop chromadb

run-llm:
	$(DOCKER_COMPOSE) run --rm --build llm-cli

run-polymarket-search:
	$(DOCKER_COMPOSE) run --rm --build polymarket-search

run-kalshi-events:
	$(DOCKER_COMPOSE) run --rm --build kalshi-events

run-kafka:
	@set -e; \
	trap '$(DOCKER_COMPOSE) stop kafka-broker zookeeper >/dev/null 2>&1 || true' EXIT; \
	$(DOCKER_COMPOSE) up -d zookeeper kafka-broker; \
	$(DOCKER_COMPOSE) run --rm --build kafka-cli

kafka-stop:
	$(DOCKER_COMPOSE) stop kafka-broker zookeeper

down:
	$(DOCKER_COMPOSE) down --remove-orphans
