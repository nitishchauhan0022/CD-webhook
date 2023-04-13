# This Makefile helps automate the build, testing, and running process for a Go project

# Target 'vet' checks the project for potential issues
vet:
	go vet .

# Target 'test' runs the test suite
test:
	go test

# Target 'build' depends on 'vet' and builds the binary
build: vet
	go build -o bin/test main.go

# Target 'run' depends on 'build' and runs the generated binary
run: build
	./bin/test

# Target 'clean' removes the generated binary
clean:
	rm -f bin/test