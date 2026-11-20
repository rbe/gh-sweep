# GNU make 3.81 (the make shipped with macOS) and newer.
# Recipes are POSIX sh.

PREFIX ?= $(HOME)/.local
BINDIR ?= $(PREFIX)/bin
DESTDIR ?=
INSTALL ?= install

IMAGE ?= gh-sweep
CONTAINERFILE ?= Containerfile
# Podman on either OS. Override with CONTAINER_ENGINE=docker.
CONTAINER_ENGINE ?= $(shell command -v podman >/dev/null 2>&1 && echo podman || (command -v docker >/dev/null 2>&1 && echo docker || echo podman))
# Go itself runs in this image, so the host does not need a Go install.
GO_IMAGE ?= docker.io/library/golang:1.22-bookworm

uname_s := $(shell uname -s)
uname_m := $(shell uname -m)

ifeq ($(uname_s),Darwin)
HOST_GOOS := darwin
else
HOST_GOOS := linux
endif

ifeq ($(uname_m),x86_64)
HOST_GOARCH := amd64
else ifeq ($(uname_m),aarch64)
HOST_GOARCH := arm64
else ifeq ($(uname_m),arm64)
HOST_GOARCH := arm64
else
HOST_GOARCH := $(uname_m)
endif

# The source tree is read-only inside the container. Go's caches live in
# named volumes, not in .cache on the host. On macOS the engine runs in a
# virtual machine, and files it creates on a bind mount belong to a user
# the Mac account cannot delete.
SRC_MOUNT := $(CURDIR):/src:ro
CLEAN_MOUNT := $(CURDIR):/src
ifeq ($(uname_s),Linux)
ifeq ($(shell getenforce 2>/dev/null),Enforcing)
SRC_MOUNT := $(CURDIR):/src:ro,Z
CLEAN_MOUNT := $(CURDIR):/src:Z
endif
endif

# Extra arguments for the checklist, for example:
#   make run ARGS='rbe/erechnung --dry-run'
ARGS ?=

# $(1) is extra container -e flags, or empty. The source tree is not written.
define go-run
$(CONTAINER_ENGINE) run --rm \
	-e HOME=/tmp \
	-e CGO_ENABLED=0 \
	-e GOCACHE=/var/cache/gh-sweep/build \
	-e GOMODCACHE=/var/cache/gh-sweep/mod \
	$(1) \
	-v gh-sweep-gocache:/var/cache/gh-sweep/build \
	-v gh-sweep-gomod:/var/cache/gh-sweep/mod \
	-v "$(SRC_MOUNT)" \
	-w /src \
	$(GO_IMAGE)
endef

# The same container with a writable source tree. Only coverage and
# format-fix use it. They write build/coverage and the Go files.
define go-run-rw
$(CONTAINER_ENGINE) run --rm \
	-e HOME=/tmp \
	-e CGO_ENABLED=0 \
	-e GOCACHE=/var/cache/gh-sweep/build \
	-e GOMODCACHE=/var/cache/gh-sweep/mod \
	$(1) \
	-v gh-sweep-gocache:/var/cache/gh-sweep/build \
	-v gh-sweep-gomod:/var/cache/gh-sweep/mod \
	-v "$(CLEAN_MOUNT)" \
	-w /src \
	$(GO_IMAGE)
endef

# The Make contract of the project-setup skill. Each command variable is
# set before the include. Go runs in the container, as before.
SETUP_CMD = $(call go-run,) go mod download
BUILD_CMD = $(MAKE) bin/gh-sweep
TEST_CMD = $(call go-run,) go test ./...
INTEGRATION_CMD = true
# gofmt reads the package directories of the project and no other directory.
FORMAT_CMD = $(call go-run,) sh -c 'files=$$(gofmt -l $$(go list -f "{{.Dir}}" ./...)); if [ -n "$$files" ]; then printf "%s\n" "gofmt would change:" "$$files"; exit 1; fi'
FORMAT_FIX_CMD = $(call go-run-rw,) sh -c 'gofmt -w $$(go list -f "{{.Dir}}" ./...)'
LINT_CMD = $(call go-run,) go vet ./...
COVERAGE_CMD = set -eu; \
	base=$$(git merge-base HEAD origin/$(DEFAULT_BRANCH) 2>/dev/null || git rev-parse -q --verify HEAD 2>/dev/null || git hash-object -t tree /dev/null); \
	git add -N .; \
	git diff "$$base" > build/coverage/change.diff; \
	$(call go-run-rw,) sh -c 'set -eu; mkdir -p build/coverage/go; for pkg in $$(go list ./...); do go test -coverpkg=./... -coverprofile="build/coverage/go/$$(printf "%s" "$$pkg" | tr "/" "_").out" "$$pkg"; done'; \
	$(PYTHON) .claude/skills/project-report/scripts/go_cover_map.py; \
	REPORT=build/coverage/coverage.xml $(DIFF_COVER)
TEST_CHANGED_CMD = chosen=$$($(PYTHON) .claude/skills/project-report/scripts/changed_tests.py --print); \
	if [ -z "$$chosen" ]; then printf '%s\n' "test-changed: no test covers the changed files. make test runs the full suite."; \
	else printf 'test-changed: %s\n' "$$chosen"; $(call go-run,) go test $$chosen; fi

include .claude/skills/project-setup/project.mk

.PHONY: install image auth token run clean notes

SOURCES := $(wildcard *.go internal/*/*.go)

bin/gh-sweep: $(SOURCES) go.mod go.sum
	mkdir -p bin
	tmp=$$(mktemp); \
	if $(call go-run,-e GOOS=$(HOST_GOOS) -e GOARCH=$(HOST_GOARCH)) \
		sh -c 'go build -trimpath -ldflags="-s -w" -o /tmp/gh-sweep . && cat /tmp/gh-sweep' \
		> "$$tmp"; then \
		mv "$$tmp" $@; \
		chmod 755 $@; \
	else \
		status=$$?; rm -f "$$tmp"; exit $$status; \
	fi

install: bin/gh-sweep ## Install to ~/.local/bin. PREFIX and DESTDIR work
	$(INSTALL) -d "$(DESTDIR)$(BINDIR)"
	$(INSTALL) -m 755 bin/gh-sweep "$(DESTDIR)$(BINDIR)/gh-sweep"
	@if [ -z "$(DESTDIR)" ]; then \
		case ":$$PATH:" in \
			*":$(BINDIR):"*) ;; \
			*) printf '%s\n' '$(BINDIR) is not on PATH. Add it with:' \
				'export PATH="$(BINDIR):$$PATH"' ;; \
		esac; \
	fi

image: bin/image.stamp ## Build the container image

bin/image.stamp: $(CONTAINERFILE) .containerignore go.mod go.sum $(SOURCES)
	mkdir -p bin
	$(CONTAINER_ENGINE) build -t $(IMAGE) -f $(CONTAINERFILE) .
	touch $@

# Log in on the host. The token goes to the system keyring, not ~/.config/gh.
auth: ## gh auth login. The token goes to the system keyring
	gh auth login

# Print the keyring token. The container cannot read the host keyring.
token: ## Print the keyring token
	@gh auth token

# Build the image if needed, then run it. The host reads the token from
# the keyring. The container gets GH_TOKEN by name, so the value is not
# an argument of the engine. The run has a terminal when Make has one.
run: bin/image.stamp ## Run gh-sweep in the container with GH_TOKEN from the keyring. Set ARGS to owner/repo
	@GH_TOKEN="$${GH_TOKEN:-$$(gh auth token 2>/dev/null || true)}"; \
	if [ -z "$$GH_TOKEN" ]; then \
		printf '%s\n' 'No token in the keyring. Run: make auth' >&2; \
		exit 1; \
	fi; \
	export GH_TOKEN; \
	tty=""; if [ -t 0 ] && [ -t 1 ]; then tty="-it"; fi; \
	$(CONTAINER_ENGINE) run --rm $$tty \
		-e GH_TOKEN \
		-v "$(CURDIR):/work" -w /work \
		$(IMAGE) $(ARGS)

clean: ## Remove bin/, build/, a leftover .cache, and the Go volumes. FORCE=1 also removes the image
	rm -rf bin build
	if [ -d .cache ]; then \
		chmod -R u+w .cache 2>/dev/null || true; \
		if ! rm -rf .cache 2>/dev/null; then \
			$(CONTAINER_ENGINE) run --rm --user 0:0 \
				-v "$(CLEAN_MOUNT)" -w /src \
				$(GO_IMAGE) \
				rm -rf .cache; \
		fi; \
	fi
	-$(CONTAINER_ENGINE) volume rm -f gh-sweep-gocache gh-sweep-gomod >/dev/null 2>&1
	if [ "$(FORCE)" = "1" ]; then \
		ids=$$($(CONTAINER_ENGINE) ps -aq --filter ancestor=$(IMAGE) 2>/dev/null || true); \
		if [ -n "$$ids" ]; then $(CONTAINER_ENGINE) rm -f $$ids; fi; \
		$(CONTAINER_ENGINE) rmi -f $(IMAGE) >/dev/null 2>&1 || true; \
	fi

notes: ## Print the notes about Go, the cache, and the keyring
	@printf '%s\n' \
		'Go is not installed on the host. Build and test use $(GO_IMAGE).' \
		'The Go cache is the volumes gh-sweep-gocache and gh-sweep-gomod.' \
		'The container cannot see the macOS Keychain or Linux keyring.' \
		'make auth && make run ARGS="owner/repo"'
