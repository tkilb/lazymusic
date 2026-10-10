BINARY      := lazymusic
INSTALL_DIR := $(HOME)/bin
CMD_PKG     := ./cmd/lazymusic
VERSION     := $(shell cat VERSION)
LDFLAGS     := -X github.com/tylerkilburn/lazymusic/internal/version.Version=$(VERSION)

.PHONY: build install test vet clean release minor major patch

build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD_PKG)

## install builds the binary and copies it to $(INSTALL_DIR) (must be on PATH).
install: build
	mkdir -p $(INSTALL_DIR)
	install -m 755 $(BINARY) $(INSTALL_DIR)/$(BINARY)
	@echo "Installed $(BINARY) to $(INSTALL_DIR)/$(BINARY)"

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -f $(BINARY)

## release bumps VERSION and cuts a release. Defaults to a patch bump;
## pass `minor`/`major` as an extra goal to bump that part instead, e.g.
## `make release minor`. Commits just the VERSION file, tags v<version>,
## and pushes branch and tag to origin — triggering
## .github/workflows/release.yml. See scripts/release.sh.
BUMP_TYPE := patch
ifneq (,$(filter minor,$(MAKECMDGOALS)))
BUMP_TYPE := minor
endif
ifneq (,$(filter major,$(MAKECMDGOALS)))
BUMP_TYPE := major
endif

release:
	@./scripts/release.sh $(BUMP_TYPE)

# no-op targets so `make release minor` / `major` / `patch` don't error
minor major patch:
	@:
