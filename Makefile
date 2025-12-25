EXPERIMENTS_DIR := experiments

.PHONY: % experiments

experiments:
	@printf "Run 'make -C %s <target>' or simply 'make <target>' to invoke experiment commands.\n" "$(EXPERIMENTS_DIR)"

%:
	@$(MAKE) -C $(EXPERIMENTS_DIR) $@
