IMG ?= nitishchauhan0022/gitops:latest

# Target 'vet' checks the project for potential issues
.PHONY: vet
vet:
	go vet ./...

# Target 'fmt' formats the codebase
.PHONY: fmt
fmt:
	go vet ./...

# Target 'test' runs the test suite
.PHONY: test
test:
	go test ./...

# Target 'build' depends on 'vet' and builds the binary
.PHONY: build
build: fmt vet
	go build -o bin/gitops main.go

# Target 'run' depends on 'build' and runs the generated binary
.PHONY: run
run: build
	./bin/gitops

# Target 'clean' removes the generated binary
.PHONY: clean
clean:
	rm -f bin/
	
.PHONY: docker-build
docker-build: ## Build docker image with the manager.
	docker build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${IMG}