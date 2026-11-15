# GNU make 3.81 (the make shipped with macOS) and newer.
# Recipes are POSIX sh.

PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin
DESTDIR ?=
INSTALL ?= install

IMAGE ?= gh-sweep
CONTAINERFILE ?= Containerfile
# Podman on either OS. Override with CONTAINER_ENGINE=docker.
CONTAINER_ENGINE ?= $(shell command -v podman >/dev/null 2>&1 && echo podman || (command -v docker >/dev/null 2>&1 && echo docker || echo podman))
# Go itself runs in this image, so the host does not need a Go install.
GO_IMAGE ?= docker.io/library/golang:1.22-bookworm

UID := $(shell id -u)
GID := $(shell id -g)
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

# Podman on enforcing SELinux needs a private volume label. macOS does not.
SELINUX_VOL :=
ifeq ($(uname_s),Linux)
ifeq ($(shell getenforce 2>/dev/null),Enforcing)
SELINUX_VOL := :Z
endif
endif

# Rootless Podman maps container uid 0 to the host user. keep-id makes
# uid $(UID) inside the container the same person, so bin/ is not root-owned.
CONTAINER_USERNS :=
ifeq ($(CONTAINER_ENGINE),podman)
CONTAINER_USERNS := --userns=keep-id
endif

# Extra arguments for the checklist, for example:
#   make run ARGS='rbe/erechnung --dry-run'
ARGS ?=

.PHONY: all build install test image auth token run container clean help

all: build

build: bin/gh-sweep

bin/gh-sweep: $(wildcard *.go) go.mod go.sum
	$(call go-in-container,$(HOST_GOOS),$(HOST_GOARCH),go build -trimpath -ldflags="-s -w" -o bin/gh-sweep .)

install: bin/gh-sweep
	$(INSTALL) -d "$(DESTDIR)$(BINDIR)"
	$(INSTALL) -m 755 bin/gh-sweep "$(DESTDIR)$(BINDIR)/gh-sweep"

test:
	$(call go-in-container,,,go test ./...)

image: bin/image.stamp

bin/image.stamp: $(CONTAINERFILE) .containerignore go.mod go.sum $(wildcard *.go)
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
	rm -rf bin .cache

help:
	@printf '%s\n' \
		'build       compile bin/gh-sweep for this machine, using a Go container' \
		'install     install to $$(DESTDIR)$(BINDIR) (PREFIX=$(PREFIX))' \
		'test        run the tests inside the Go container' \
		'auth        gh auth login (stores the token in the system keyring)' \
		'token       print the keyring token (gh auth token)' \
		'run         run bin/gh-sweep (ARGS="owner/repo")' \
		'image       build the container image with $(CONTAINER_ENGINE)' \
		'container   run the image; passes GH_TOKEN from the keyring' \
		'clean       remove bin/ and .cache/' \
		'' \
		'Go is not installed on the host. Build and test use $(GO_IMAGE).' \
		'The container cannot see the macOS Keychain or Linux keyring.' \
		'make auth && make container ARGS="owner/repo"'

# $(1) GOOS, $(2) GOARCH, $(3) command. Empty GOOS/GOARCH means the image's own OS.
define go-in-container
	mkdir -p .cache/go-mod .cache/go-build bin
	$(CONTAINER_ENGINE) run --rm $(CONTAINER_USERNS) \
		-u "$(UID):$(GID)" \
		-e HOME=/tmp \
		-e GOCACHE=/src/.cache/go-build \
		-e GOMODCACHE=/src/.cache/go-mod \
		-e CGO_ENABLED=0 \
		-e GOOS="$(1)" \
		-e GOARCH="$(2)" \
		-v "$(CURDIR):/src$(SELINUX_VOL)" \
		-w /src \
		$(GO_IMAGE) \
		$(3)
endef
