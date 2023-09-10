BINARY_NAME=ipkg
LIBRARY_NAME=libipkg

# ----------------------------------------------
#                   BASIC
# ----------------------------------------------
all: test build

clean:
	go clean
	rm -rf ./bin

# ----------------------------------------------
#                  HELPERS
# ----------------------------------------------

confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

git/no-dirty:
	git diff --cached --exit-code



# ----------------------------------------------
#                 DEVELOPMENT
# ----------------------------------------------

build: build/main

build/main:
	GOARCH=amd64 GOOS=linux go build -o ./bin/${BINARY_NAME}-linux ./cmd
	GOARCH=amd64 GOOS=windows go build -o ./bin/${BINARY_NAME}-windows.exe ./cmd


test:
	go test -v -race -buildvcs ./...

test/cover:
	go test -v -race -buildvcs -coverprofile=/tmp/coverage.out ./...
	go tool cover -html=/tmp/coverage.out


# ---------------------------------------------
#                 DEPENDENCIES
# ---------------------------------------------

dep:
	go mod download

dep/lint:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.54.2

# ----------------------------------------------
#               QUALITY CONTROL
# ----------------------------------------------

tidy:
	go fmt ./...
	go mod tidy -v

audit:
	go mod verify
	go vet
	go run honnef.co/go/tools/cmd/staticcheck@latest -checks=all,-ST1000,-U1000 ./...
	govulncheck ./...
	go test -v -race -buildvcs -vet=off ./...

lint:
	golangci-lint run --enable-all

# ----------------------------------------------
#                 GIT/GITHUB
# ----------------------------------------------

push: tidy audit git/no-dirty
	git push

pull: git/no-dirty
	git pull

pull/request: push
	gh pr create
	gh pr view --web

issue:
	gh issue create
	gh issue view --web

update: pull
	git pull origin stable
	git merge stable

release: confirm build
	git tag $(TAGNAME)
	git push --tags
	gh release create $(TAGNAME)
	gh release view $(TAGNAME) --html

