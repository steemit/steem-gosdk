ROOT_DIR    = $(shell pwd)
GREEN := "\\033[0;32m"
NC := "\\033[0m"
define print
	echo $(GREEN)$1$(NC)
endef

.PHONY: unit-cover
unit-cover:
	@$(call print, "Running unit coverage tests.")

.PHONY: unit-race
unit-race:
	@$(call print, "Running unit race tests.")
