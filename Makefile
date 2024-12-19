.PHONY: help
help: # @HELP Print this message
help:
	@echo "TARGETS:"
	@grep -E '^.*: *# *@HELP' $(MAKEFILE_LIST)    \
	    | awk '                                   \
	        BEGIN {FS = ": *# *@HELP"};           \
	        { printf "  %-20s %s\n", $$1, $$2 };  \
	    '

.PHONY: setup
setup: # @HELP Build the development containers and install app dependencies
setup: update
	@echo "Successfully built containers and installed dependencies."

.PHONY: up
up: # @HELP Start the development server
up:
	@docker compose up

.PHONY: down
down: # @HELP Tear down the development containers
down:
	@docker compose down

.PHONY: update
update: # @HELP Install and/or update dependencies for the vite container
update:
	@echo "Installing / updating dependencies ..."
	@docker compose run --rm vite npm ci
