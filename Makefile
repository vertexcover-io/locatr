SOCKET_FILE=/tmp/locatr.sock
BINARY_PATH=server/locatr.bin
PYTHON_CLIENT_BIN=python_client/bin

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


