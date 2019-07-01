STATICCHECK := $(GOPATH)/bin/staticcheck
RELEASE := $(GOPATH)/bin/github-release

UNAME := $(shell uname)

$(STATICCHECK):
	go get honnef.co/go/tools/cmd/staticcheck

vet: $(STATICCHECK)
	go list ./... | grep -v vendor | xargs go vet
	go list ./... | grep -v vendor | xargs $(STATICCHECK)

test: vet
	go list ./... | grep -v vendor | xargs go test

$(RELEASE): test
	go get -u github.com/aktau/github-release

release: $(RELEASE)
ifndef version
	@echo "Please provide a version"
	exit 1
endif
ifndef GITHUB_TOKEN
	@echo "Please set GITHUB_TOKEN in the environment"
	exit 1
endif
	git tag $(version)
	git push origin --tags
	mkdir -p releases/$(version)
	GOOS=linux GOARCH=amd64 go build -o releases/$(version)/chroma-markdown-linux-amd64 .
	GOOS=darwin GOARCH=amd64 go build -o releases/$(version)/chroma-markdown-darwin-amd64 .
	GOOS=windows GOARCH=amd64 go build -o releases/$(version)/chroma-markdown-windows-amd64 .
	# these commands are not idempotent so ignore failures if an upload repeats
	$(RELEASE) release --user kevinburke --repo chroma-markdown --tag $(version) || true
	$(RELEASE) upload --user kevinburke --repo chroma-markdown --tag $(version) --name chroma-markdown-linux-amd64 --file releases/$(version)/chroma-markdown-linux-amd64 || true
	$(RELEASE) upload --user kevinburke --repo chroma-markdown --tag $(version) --name chroma-markdown-darwin-amd64 --file releases/$(version)/chroma-markdown-darwin-amd64 || true
	$(RELEASE) upload --user kevinburke --repo chroma-markdown --tag $(version) --name chroma-markdown-windows-amd64 --file releases/$(version)/chroma-markdown-windows-amd64 || true
