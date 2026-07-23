BINARY := aria2tui

.PHONY: build run test vet fmt install clean

build:
	go build -o $(BINARY) .

run: build
	./$(BINARY)

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

install:
	go install .

clean:
	rm -f $(BINARY)
