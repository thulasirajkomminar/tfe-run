.PHONY: pre-commit

# Default target - runs when you type 'make' without arguments.
pre-commit: lint format build

lint:
	@TASK_X_REMOTE_TASKFILES=1 task remote:lint

format:
	@TASK_X_REMOTE_TASKFILES=1 task remote:format

build:
	go mod tidy && go build -o tfe-run .
