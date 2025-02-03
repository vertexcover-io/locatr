SOCKET_FILE=/tmp/locatr.sock
BINARY_PATH=server/locatr.bin
PYTHON_CLIENT_BIN=python_client/locatr/bin/

.PHONY: all build run clean

all: build

build:
	cd server && go build -o locatr.bin . || { echo "Go build failed"; exit 1; }
	mv $(BINARY_PATH) $(PYTHON_CLIENT_BIN) || { echo "Failed to move the binary"; exit 1; }
	echo "Build and move successful."

run:
	cd server && \
	trap 'rm -rf $(SOCKET_FILE); exit' INT TERM EXIT && \
	go run . -socketFilePath=$(SOCKET_FILE)

# Go coverage targets
.PHONY: go-test-coverage
go-test-coverage:
	go test ./golang/... -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html

# Python coverage targets
.PHONY: python-test-coverage
python-test-coverage:
	uv run python -m pytest --cov=python_client --cov-report=html --cov-config=pyproject.toml

.PHONY: test-coverage
test-coverage: go-test-coverage python-test-coverage
