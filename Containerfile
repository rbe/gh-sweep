# Build and run gh-sweep with Podman. Prefer the Makefile:
#
#	make auth
#	make container ARGS="rbe/erechnung"
#
# gh 2.40 and later keeps the token in the system keyring (service name
# gh:github.com), not in ~/.config/gh. The container cannot read that keyring.

FROM docker.io/library/golang:1.22-bookworm AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/gh-sweep .

FROM docker.io/library/debian:bookworm-slim

RUN apt-get update \
	&& apt-get install -y --no-install-recommends ca-certificates curl git \
	&& install -d -m 755 /etc/apt/keyrings \
	&& curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg \
		-o /etc/apt/keyrings/githubcli-archive-keyring.gpg \
	&& chmod go+r /etc/apt/keyrings/githubcli-archive-keyring.gpg \
	&& printf 'deb [arch=%s signed-by=/etc/apt/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main\n' \
		"$(dpkg --print-architecture)" > /etc/apt/sources.list.d/github-cli.list \
	&& apt-get update \
	&& apt-get install -y --no-install-recommends gh \
	&& rm -rf /var/lib/apt/lists/*

COPY --from=build /out/gh-sweep /usr/local/bin/gh-sweep

ENV TERM=xterm-256color
WORKDIR /work
ENTRYPOINT ["gh-sweep"]
