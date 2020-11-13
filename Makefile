SHELL=/usr/bin/env bash

all: build
.PHONY: all

unexport GOFLAGS

GOVERSION:=$(shell go version | cut -d' ' -f 3 | cut -d. -f 2)
ifeq ($(shell expr $(GOVERSION) \< 14), 1)
$(warning Your Golang version is go 1.$(GOVERSION))
$(error Update Golang to version $(shell grep '^go' go.mod))
endif

# git modules that need to be loaded
MODULES:=

CLEAN:=
BINS:=

ldflags=-X=github.com/EpiK-Protocol/go-epik/build.CurrentCommit=+git.$(subst -,.,$(shell git describe --always --match=NeVeRmAtCh --dirty 2>/dev/null || git rev-parse --short HEAD 2>/dev/null))
ifneq ($(strip $(LDFLAGS)),)
	ldflags+=-extldflags=$(LDFLAGS)
endif

GOFLAGS+=-ldflags="$(ldflags)"


## FFI

FFI_PATH:=extern/filecoin-ffi/
FFI_DEPS:=.install-filcrypto
FFI_DEPS:=$(addprefix $(FFI_PATH),$(FFI_DEPS))

$(FFI_DEPS): build/.filecoin-install ;

build/.filecoin-install: $(FFI_PATH)
	$(MAKE) -C $(FFI_PATH) $(FFI_DEPS:$(FFI_PATH)%=%)
	@touch $@

MODULES+=$(FFI_PATH)
BUILD_DEPS+=build/.filecoin-install
CLEAN+=build/.filecoin-install

$(MODULES): build/.update-modules ;

# dummy file that marks the last time modules were updated
build/.update-modules:
	git submodule update --init --recursive
	touch $@

# end git modules

## MAIN BINARIES

CLEAN+=build/.update-modules

deps: $(BUILD_DEPS)
.PHONY: deps

debug: GOFLAGS+=-tags=debug
debug: epik epik-miner epik-worker epik-seed

2k: GOFLAGS+=-tags=2k
2k: epik epik-miner epik-worker epik-seed

epik: $(BUILD_DEPS)
	rm -f epik
	go build $(GOFLAGS) -o epik ./cmd/epik
	go run github.com/GeertJohan/go.rice/rice append --exec epik -i ./build

.PHONY: epik
BINS+=epik

epik-miner: $(BUILD_DEPS)
	rm -f epik-miner
	go build $(GOFLAGS) -o epik-miner ./cmd/epik-storage-miner
	go run github.com/GeertJohan/go.rice/rice append --exec epik-miner -i ./build
.PHONY: epik-miner
BINS+=epik-miner

epik-worker: $(BUILD_DEPS)
	rm -f epik-worker
	go build $(GOFLAGS) -o epik-worker ./cmd/epik-seal-worker
	go run github.com/GeertJohan/go.rice/rice append --exec epik-worker -i ./build
.PHONY: epik-worker
BINS+=epik-worker

epik-shed: $(BUILD_DEPS)
	rm -f epik-shed
	go build $(GOFLAGS) -o epik-shed ./cmd/epik-shed
	go run github.com/GeertJohan/go.rice/rice append --exec epik-shed -i ./build
.PHONY: epik-shed
BINS+=epik-shed

epik-gateway: $(BUILD_DEPS)
	rm -f epik-gateway
	go build $(GOFLAGS) -o epik-gateway ./cmd/epik-gateway
.PHONY: epik-gateway
BINS+=epik-gateway

build: epik epik-miner epik-worker
	@[[ $$(type -P "epik") ]] && echo "Caution: you have \
an existing epik binary in your PATH. This may cause problems if you don't run 'sudo make install'" || true

.PHONY: build

install: install-daemon install-miner install-worker

install-daemon:
	install -C ./epik /usr/local/bin/epik

install-miner:
	install -C ./epik-miner /usr/local/bin/epik-miner

install-worker:
	install -C ./epik-worker /usr/local/bin/epik-worker

# TOOLS

epik-seed: $(BUILD_DEPS)
	rm -f epik-seed
	go build $(GOFLAGS) -o epik-seed ./cmd/epik-seed
	go run github.com/GeertJohan/go.rice/rice append --exec epik-seed -i ./build

.PHONY: epik-seed
BINS+=epik-seed

benchmarks:
	go run github.com/whyrusleeping/bencher ./... > bench.json
	@echo Submitting results
	@curl -X POST 'http://benchmark.kittyhawk.wtf/benchmark' -d '@bench.json' -u "${benchmark_http_cred}"
.PHONY: benchmarks

epik-pond: 2k
	go build -o epik-pond ./epikpond
.PHONY: epik-pond
BINS+=epik-pond

epik-pond-front:
	(cd epikpond/front && npm i && CI=false npm run build)
.PHONY: epik-pond-front

epik-pond-app: epik-pond-front epik-pond
.PHONY: epik-pond-app

epik-townhall:
	rm -f epik-townhall
	go build -o epik-townhall ./cmd/epik-townhall
.PHONY: epik-townhall
BINS+=epik-townhall

epik-townhall-front:
	(cd ./cmd/epik-townhall/townhall && npm i && npm run build)
.PHONY: epik-townhall-front

epik-townhall-app: epik-touch epik-townhall-front
	go run github.com/GeertJohan/go.rice/rice append --exec epik-townhall -i ./cmd/epik-townhall -i ./build
.PHONY: epik-townhall-app

epik-fountain:
	rm -f epik-fountain
	go build -o epik-fountain ./cmd/epik-fountain
	go run github.com/GeertJohan/go.rice/rice append --exec epik-fountain -i ./cmd/epik-fountain -i ./build
.PHONY: epik-fountain
BINS+=epik-fountain

epik-chainwatch:
	rm -f epik-chainwatch
	go build $(GOFLAGS) -o epik-chainwatch ./cmd/epik-chainwatch
.PHONY: epik-chainwatch
BINS+=epik-chainwatch

epik-bench:
	rm -f epik-bench
	go build -o epik-bench ./cmd/epik-bench
	go run github.com/GeertJohan/go.rice/rice append --exec epik-bench -i ./build
.PHONY: epik-bench
BINS+=epik-bench

epik-stats:
	rm -f epik-stats
	go build $(GOFLAGS) -o epik-stats ./cmd/epik-stats
	go run github.com/GeertJohan/go.rice/rice append --exec epik-stats -i ./build
.PHONY: epik-stats
BINS+=epik-stats

epik-pcr:
	rm -f epik-pcr
	go build $(GOFLAGS) -o epik-pcr ./cmd/epik-pcr
	go run github.com/GeertJohan/go.rice/rice append --exec epik-pcr -i ./build
.PHONY: epik-pcr
BINS+=epik-pcr

epik-health:
	rm -f epik-health
	go build -o epik-health ./cmd/epik-health
	go run github.com/GeertJohan/go.rice/rice append --exec epik-health -i ./build
.PHONY: epik-health
BINS+=epik-health

epik-wallet:
	rm -f epik-wallet
	go build -o epik-wallet ./cmd/epik-wallet
.PHONY: epik-wallet
BINS+=epik-wallet

testground:
	go build -tags testground -o /dev/null ./cmd/epik
.PHONY: testground
BINS+=testground

install-chainwatch: epik-chainwatch
	install -C ./epik-chainwatch /usr/local/bin/epik-chainwatch

# SYSTEMD

install-daemon-service: install-daemon
	mkdir -p /etc/systemd/system
	mkdir -p /var/log/epik
	install -C -m 0644 ./scripts/epik-daemon.service /etc/systemd/system/epik-daemon.service
	systemctl daemon-reload
	@echo
	@echo "epik-daemon service installed. Don't forget to run 'sudo systemctl start epik-daemon' to start it and 'sudo systemctl enable epik-daemon' for it to be enabled on startup."

install-miner-service: install-miner install-daemon-service
	mkdir -p /etc/systemd/system
	mkdir -p /var/log/epik
	install -C -m 0644 ./scripts/epik-miner.service /etc/systemd/system/epik-miner.service
	systemctl daemon-reload
	@echo
	@echo "epik-miner service installed. Don't forget to run 'sudo systemctl start epik-miner' to start it and 'sudo systemctl enable epik-miner' for it to be enabled on startup."

install-chainwatch-service: install-chainwatch install-daemon-service
	mkdir -p /etc/systemd/system
	mkdir -p /var/log/epik
	install -C -m 0644 ./scripts/epik-chainwatch.service /etc/systemd/system/epik-chainwatch.service
	systemctl daemon-reload
	@echo
	@echo "chainwatch service installed. Don't forget to run 'sudo systemctl start epik-chainwatch' to start it and 'sudo systemctl enable epik-chainwatch' for it to be enabled on startup."

install-main-services: install-miner-service

install-all-services: install-main-services install-chainwatch-service

install-services: install-main-services

clean-daemon-service: clean-miner-service clean-chainwatch-service
	-systemctl stop epik-daemon
	-systemctl disable epik-daemon
	rm -f /etc/systemd/system/epik-daemon.service
	systemctl daemon-reload

clean-miner-service:
	-systemctl stop epik-miner
	-systemctl disable epik-miner
	rm -f /etc/systemd/system/epik-miner.service
	systemctl daemon-reload

clean-chainwatch-service:
	-systemctl stop epik-chainwatch
	-systemctl disable epik-chainwatch
	rm -f /etc/systemd/system/epik-chainwatch.service
	systemctl daemon-reload

clean-main-services: clean-daemon-service

clean-all-services: clean-main-services

clean-services: clean-all-services

# MISC

buildall: $(BINS)

completions:
	./scripts/make-completions.sh epik
	./scripts/make-completions.sh epik-miner
.PHONY: completions

install-completions:
	mkdir -p /usr/share/bash-completion/completions /usr/local/share/zsh/site-functions/
	install -C ./scripts/bash-completion/epik /usr/share/bash-completion/completions/epik
	install -C ./scripts/bash-completion/epik-miner /usr/share/bash-completion/completions/epik-miner
	install -C ./scripts/zsh-completion/epik /usr/local/share/zsh/site-functions/_epik
	install -C ./scripts/zsh-completion/epik-miner /usr/local/share/zsh/site-functions/_epik-miner

clean:
	rm -rf $(CLEAN) $(BINS)
	-$(MAKE) -C $(FFI_PATH) clean
.PHONY: clean

dist-clean:
	git clean -xdff
	git submodule deinit --all -f
.PHONY: dist-clean

type-gen:
	go run ./gen/main.go
	go generate ./...

method-gen:
	(cd ./epikpond/front/src/chain && go run ./methodgen.go)

gen: type-gen method-gen

docsgen:
	go run ./api/docgen "api/api_full.go" "FullNode" > documentation/en/api-methods.md
	go run ./api/docgen "api/api_storage.go" "StorageMiner" > documentation/en/api-methods-miner.md
	go run ./api/docgen "api/api_worker.go" "WorkerAPI" > documentation/en/api-methods-worker.md

print-%:
	@echo $*=$($*)
