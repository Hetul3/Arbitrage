EXPERIMENTS_DIR := experiments
DOCKER_COMPOSE ?= docker compose

.PHONY: run-polymarket-collector run-polymarket-collector-dev run-kalshi-collector run-kalshi-collector-dev run-collectors run-collectors-dev sqlite-create sqlite-drop sqlite-clear sqlite-migrate collectors-down experiments %

run-polymarket-collector:
	$(DOCKER_COMPOSE) run --rm --build polymarket-collector

run-polymarket-collector-dev:
	$(DOCKER_COMPOSE) run --rm --build polymarket-collector-dev

run-kalshi-collector:
	$(DOCKER_COMPOSE) run --rm --build kalshi-collector

run-kalshi-collector-dev:
	$(DOCKER_COMPOSE) run --rm --build kalshi-collector-dev

run-collectors:
	$(DOCKER_COMPOSE) up --build polymarket-collector kalshi-collector

run-collectors-dev:
	$(DOCKER_COMPOSE) up --build polymarket-collector-dev kalshi-collector-dev

sqlite-create:
	$(DOCKER_COMPOSE) run --rm --build sqlite-create

sqlite-drop:
	$(DOCKER_COMPOSE) run --rm --build sqlite-drop

sqlite-clear:
	$(DOCKER_COMPOSE) run --rm --build sqlite-clear

sqlite-migrate:
	$(DOCKER_COMPOSE) run --rm --build sqlite-migrate

collectors-down:
	$(DOCKER_COMPOSE) down --remove-orphans

experiments:
	@printf "Run 'make -C %s <target>' or simply 'make <target>' to invoke experiment commands.\n" "$(EXPERIMENTS_DIR)"

%:
	@$(MAKE) -C $(EXPERIMENTS_DIR) $@
