SHELL = /bin/bash
STATICCHECK := $(GOPATH)/bin/staticcheck

UNAME := $(shell uname)

# major, minor or patch: which part of the version in version.go to bump.
version ?= minor

# gh reads GITHUB_TOKEN from the environment; fall back to the token on disk.
GITHUB_TOKEN ?= $(shell cat $(HOME)/cfg/github-token 2>/dev/null)
export GITHUB_TOKEN

PLATFORMS := linux-amd64 darwin-amd64 darwin-arm64 windows-amd64


vet:
	go vet ./...
	go install honnef.co/go/tools/cmd/staticcheck@latest
	staticcheck ./...

test: vet
	go test ./...

# Bump version.go (commit + annotated tag, via bump_version), push both to
# origin master, build the binaries into releases/<tag>/ and publish a GitHub
# release with them attached. Run from a clean checkout whose HEAD is
# origin/master or a fast-forward of it; the push fails otherwise. Usage:
#
#   make release              # minor bump
#   make release version=patch
.PHONY: release
release: test
	@set -euo pipefail; \
	case "$(version)" in major|minor|patch) ;; \
	*) echo "version must be major, minor or patch, got '$(version)'" >&2; exit 1;; esac; \
	if [ -z "$$GITHUB_TOKEN" ]; then \
		echo "GITHUB_TOKEN is unset and ~/cfg/github-token is empty or missing" >&2; exit 1; \
	fi; \
	gh api user --jq .login >/dev/null || \
		{ echo "GITHUB_TOKEN was rejected by GitHub; refusing to bump and push" >&2; exit 1; }; \
	if [ -n "$$(git status --porcelain)" ]; then \
		echo "working tree is dirty; commit or stash first" >&2; exit 1; \
	fi; \
	git fetch origin master; \
	git merge-base --is-ancestor origin/master HEAD || \
		{ echo "HEAD does not contain origin/master; rebase first" >&2; exit 1; }; \
	go run github.com/kevinburke/bump_version@latest $(version) version.go; \
	tag="$$(go run github.com/kevinburke/bump_version/current_version@latest version.go)"; \
	git push origin HEAD:master "$$tag"; \
	mkdir -p "releases/$$tag"; \
	assets=(); \
	for p in $(PLATFORMS); do \
		out="releases/$$tag/chroma-markdown-$$p"; \
		GOOS="$${p%-*}" GOARCH="$${p#*-}" go build -trimpath -o "$$out" .; \
		assets+=("$$out"); \
	done; \
	gh release create "$$tag" --repo kevinburke/chroma-markdown --verify-tag \
		--title "$$tag" --generate-notes "$${assets[@]}"; \
	echo "released $$tag"
