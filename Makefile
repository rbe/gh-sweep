# GNU make 3.81 (the make shipped with macOS) and newer.
# Recipes are POSIX sh.

PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin
DESTDIR ?=
GO ?= go
INSTALL ?= install

IMAGE ?= gh-sweep
CONTAINERFILE ?= Containerfile
# Podman on either OS. Override with CONTAINER_ENGINE=docker.
CONTAINER_ENGINE ?= $(shell command -v podman >/dev/null 2>&1 && echo podman || (command -v docker >/dev/null 2>&1 && echo docker || echo podman))

# Extra arguments for the checklist, for example:
#   make run ARGS='rbe/erechnung --dry-run'
ARGS ?=

.PHONY: all build install test image auth token run container clean help

all: build

build: bin/gh-sweep

bin/gh-sweep: $(wildcard *.go) go.mod go.sum
	mkdir -p bin
	$(GO) build -trimpath -ldflags="-s -w" -o $@ .

install: bin/gh-sweep
	$(INSTALL) -d "$(DESTDIR)$(BINDIR)"
	$(INSTALL) -m 755 bin/gh-sweep "$(DESTDIR)$(BINDIR)/gh-sweep"

test:
	$(GO) test ./...

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
	rm -rf bin

help:
	@printf '%s\n' \
		'build       compile bin/gh-sweep (default)' \
		'install     install to $$(DESTDIR)$(BINDIR) (PREFIX=$(PREFIX))' \
		'test        go test ./...' \
		'auth        gh auth login (stores the token in the system keyring)' \
		'token       print the keyring token (gh auth token)' \
		'run         run bin/gh-sweep (ARGS="owner/repo")' \
		'image       build the container image with $(CONTAINER_ENGINE)' \
		'container   run the image; passes GH_TOKEN from the keyring' \
		'clean       remove bin/' \
		'' \
		'The container cannot see the macOS Keychain or Linux keyring.' \
		'make auth && make container ARGS='"'"'owner/repo'"'"
