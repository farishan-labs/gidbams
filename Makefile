BINARY=git-dbms
PKG=./cmd/git-dbms

build:
	GOWORK=off go build -o bin/$(BINARY) $(PKG)

test:
	GOWORK=off go test ./...

vet:
	GOWORK=off go vet ./...

install:
	GOWORK=off go install $(PKG)
EOF