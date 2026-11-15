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

.PHONY: all build install test image auth token run container clean help

all: build

build: bin/gh-sweep

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

install: bin/gh-sweep
	$(INSTALL) -d "$(DESTDIR)$(BINDIR)"
	$(INSTALL) -m 755 bin/gh-sweep "$(DESTDIR)$(BINDIR)/gh-sweep"
	@if [ -z "$(DESTDIR)" ]; then \
		case ":$$PATH:" in \
			*":$(BINDIR):"*) ;; \
			*) printf '%s\n' '$(BINDIR) is not on PATH. Add it with:' \
				'export PATH="$(BINDIR):$$PATH"' ;; \
		esac; \
	fi

test:
	$(call go-run,) go test ./...

image: bin/image.stamp

bin/image.stamp: $(CONTAINERFILE) .containerignore go.mod go.sum $(SOURCES)
	mkdir -p bin
	$(CONTAINER_ENGINE) build -t $(IMAGE) -f $(CONTAINERFILE) .
	touch $@

# Log in on the host. The token goes to the system keyring, not ~/.config/gh.
auth:
	gh auth login

# Print the keyring token. The container cannot read the host keyring.
token:
	@gh auth token

run: bin/gh-sweep
	./bin/gh-sweep $(ARGS)

# Build the image if needed, then run it with a terminal and GH_TOKEN.
container: bin/image.stamp
	token="$${GH_TOKEN:-$$(gh auth token 2>/dev/null || true)}"; \
	if [ -z "$$token" ]; then \
		printf '%s\n' 'No token in the keyring. Run: make auth' >&2; \
		exit 1; \
	fi; \
	$(CONTAINER_ENGINE) run --rm -it \
		-e "GH_TOKEN=$$token" \
		-v "$(CURDIR):/work" -w /work \
		$(IMAGE) $(ARGS)

clean:
	rm -rf bin
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

help:
	@printf '%s\n' \
		'build       compile bin/gh-sweep for this machine, using a Go container' \
		'install     install to ~/.local/bin (PREFIX=$(PREFIX))' \
		'test        run the tests inside the Go container' \
		'auth        gh auth login (stores the token in the system keyring)' \
		'token       print the keyring token (gh auth token)' \
		'run         run bin/gh-sweep (ARGS="owner/repo")' \
		'image       build the container image with $(CONTAINER_ENGINE)' \
		'container   run the image; passes GH_TOKEN from the keyring' \
		'clean       remove bin/, leftover .cache, and the Go volumes' \
		'' \
		'Go is not installed on the host. Build and test use $(GO_IMAGE).' \
		'The Go cache is the volumes gh-sweep-gocache and gh-sweep-gomod.' \
		'The container cannot see the macOS Keychain or Linux keyring.' \
		'make auth && make container ARGS="owner/repo"'

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
