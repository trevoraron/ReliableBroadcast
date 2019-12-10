# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=client

all: build
build:
	sh generate_certs.sh config.json
	$(GOBUILD) -o $(BINARY_NAME) -v
test:
	$(GOTEST) -v ./...
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm ./certs/*.pem
	rm ./certs/*.csr
	rm ./certs/*.tmp.json
run:
	Make build
	sh run_demo.sh
