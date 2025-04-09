SOCKET_FILE=/tmp/locatr.sock
SERVER_PATH=ipc/server
PYTHON_CLIENT_PATH=ipc/python_client
BINARY_PATH=${SERVER_PATH}/locatr.bin
PYTHON_CLIENT_LOCATR_BIN_DIR=${PYTHON_CLIENT_PATH}/locatr/bin
PYTHON_CLIENT_DIST_DIR=${PYTHON_CLIENT_PATH}/dist

.PHONY: all build run clean

all: build

# Build the IPC server and the python client
build:
	echo "\nBuilding artifacts..."
	cd ${SERVER_PATH} && go build -o locatr.bin . || { echo "IPC server build failed"; exit 1; }
	mkdir -p ${PYTHON_CLIENT_LOCATR_BIN_DIR}
	mv ${BINARY_PATH} ${PYTHON_CLIENT_LOCATR_BIN_DIR} || { echo "Failed to move the binary"; exit 1; }
	uv --directory ${PYTHON_CLIENT_PATH} build || { echo "Python client build failed"; exit 1; }
	echo "Build and move successful.\n"

# Create a virtual environment in current directory and install the python client
install: clean build
	echo "\nInstalling python client..."
	test -d .venv || uv venv
	/bin/bash -c "source .venv/bin/activate && uv pip install ${PYTHON_CLIENT_DIST_DIR}/*.whl"
	echo "Installation successful.\n"

# Run the IPC server
run:
	echo "\nRunning IPC server..."
	cd ${SERVER_PATH} && \
	trap 'rm -rf ${SOCKET_FILE}; exit' INT TERM EXIT && \
	go run . -socketFilePath=${SOCKET_FILE}
	echo "IPC server running.\n"

# Cleanup the build artifacts
clean:
	echo "\nCleaning up build artifacts..."
	rm -rf ${PYTHON_CLIENT_DIST_DIR}
	rm -rf ${PYTHON_CLIENT_LOCATR_BIN_DIR}
	echo "Cleanup successful.\n"

# Publish the python client to pypi
publish: clean build
	echo "\nPublishing python client to pypi..."
	uv --directory ${PYTHON_CLIENT_PATH} publish
	echo "Publish successful.\n"

# Run go tests and generate a coverage report
.PHONY: go-test-coverage
go-test-coverage:
	echo "\nRunning go tests and generating coverage report..."
	go test ./pkg/... -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html
	echo "Go test coverage report generated.\n"

# Run python tests and generate a coverage report
.PHONY: python-test-coverage
python-test-coverage:
	echo "\nRunning python tests and generating coverage report..."
	uv --directory ${PYTHON_CLIENT_PATH} run python -m pytest --cov-report=html --cov-config=pyproject.toml
	echo "Python test coverage report generated.\n"

# Run both go and python tests and generate coverage reports
.PHONY: test-coverage
test-coverage: go-test-coverage python-test-coverage
