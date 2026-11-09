BINARY=git-dbms
PKG=./cmd/git-dbms

build:
	GOWORK=off go build -o bin/$(BINARY) $(PKG)

test:
	GOWORK=off go test ./...

vet:
	GOWORK=off go vet ./...

fmt:
	GOWORK=off gofmt -w .

lint: vet
	GOWORK=off gofmt -l .

install:
	GOWORK=off go install $(PKG)

clean:
	rm -rf bin dist
