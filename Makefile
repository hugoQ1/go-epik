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
debug: epik epik-storage-miner epik-seal-worker epik-seed

2k: GOFLAGS+=-tags=2k
2k: epik epik-storage-miner epik-seal-worker epik-seed

epik: $(BUILD_DEPS)
	rm -f epik
	go build $(GOFLAGS) -o epik ./cmd/epik
	go run github.com/GeertJohan/go.rice/rice append --exec epik -i ./build

.PHONY: epik
BINS+=epik

epik-storage-miner: $(BUILD_DEPS)
	rm -f epik-storage-miner
	go build $(GOFLAGS) -o epik-storage-miner ./cmd/epik-storage-miner
	go run github.com/GeertJohan/go.rice/rice append --exec epik-storage-miner -i ./build
.PHONY: epik-storage-miner
BINS+=epik-storage-miner

epik-seal-worker: $(BUILD_DEPS)
	rm -f epik-seal-worker
	go build $(GOFLAGS) -o epik-seal-worker ./cmd/epik-seal-worker
	go run github.com/GeertJohan/go.rice/rice append --exec epik-seal-worker -i ./build
.PHONY: epik-seal-worker
BINS+=epik-seal-worker

epik-shed: $(BUILD_DEPS)
	rm -f epik-shed
	go build $(GOFLAGS) -o epik-shed ./cmd/epik-shed
	go run github.com/GeertJohan/go.rice/rice append --exec epik-shed -i ./build
.PHONY: epik-shed
BINS+=epik-shed

build: epik epik-storage-miner epik-seal-worker
	@[[ $$(type -P "epik") ]] && echo "Caution: you have \
an existing epik binary in your PATH. This may cause problems if you don't run 'sudo make install'" || true

.PHONY: build

install:
	install -C ./epik /usr/local/bin/epik
	install -C ./epik-storage-miner /usr/local/bin/epik-storage-miner
	install -C ./epik-seal-worker /usr/local/bin/epik-seal-worker

install-services: install
	mkdir -p /usr/local/lib/systemd/system
	mkdir -p /var/log/epik
	install -C -m 0644 ./scripts/epik-daemon.service /usr/local/lib/systemd/system/epik-daemon.service
	install -C -m 0644 ./scripts/epik-miner.service /usr/local/lib/systemd/system/epik-miner.service
	systemctl daemon-reload
	@echo
	@echo "epik-daemon and epik-miner services installed. Don't forget to 'systemctl enable epik-daemon|epik-miner' for it to be enabled on startup."

clean-services:
	rm -f /usr/local/lib/systemd/system/epik-daemon.service
	rm -f /usr/local/lib/systemd/system/epik-miner.service
	rm -f /usr/local/lib/systemd/system/chainwatch.service
	systemctl daemon-reload

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

pond: 2k
	go build -o pond ./epikpond
	(cd epikpond/front && npm i && CI=false npm run build)
.PHONY: pond
BINS+=pond

townhall:
	rm -f townhall
	go build -o townhall ./cmd/epik-townhall
	(cd ./cmd/epik-townhall/townhall && npm i && npm run build)
	go run github.com/GeertJohan/go.rice/rice append --exec townhall -i ./cmd/epik-townhall -i ./build
.PHONY: townhall
BINS+=townhall

fountain:
	rm -f fountain
	go build -o fountain ./cmd/epik-fountain
	go run github.com/GeertJohan/go.rice/rice append --exec fountain -i ./cmd/epik-fountain -i ./build
.PHONY: fountain
BINS+=fountain

chainwatch:
	rm -f chainwatch
	go build -o chainwatch ./cmd/epik-chainwatch
	go run github.com/GeertJohan/go.rice/rice append --exec chainwatch -i ./cmd/epik-chainwatch -i ./build
.PHONY: chainwatch
BINS+=chainwatch

install-chainwatch-service: chainwatch
	install -C ./chainwatch /usr/local/bin/chainwatch
	install -C -m 0644 ./scripts/chainwatch.service /usr/local/lib/systemd/system/chainwatch.service
	systemctl daemon-reload
	@echo
	@echo "chainwatch installed. Don't forget to 'systemctl enable chainwatch' for it to be enabled on startup."

bench:
	rm -f bench
	go build -o bench ./cmd/epik-bench
	go run github.com/GeertJohan/go.rice/rice append --exec bench -i ./build
.PHONY: bench
BINS+=bench

stats:
	rm -f stats
	go build -o stats ./tools/stats
	go run github.com/GeertJohan/go.rice/rice append --exec stats -i ./build
.PHONY: stats
BINS+=stats

health:
	rm -f epik-health
	go build -o epik-health ./cmd/epik-health
	go run github.com/GeertJohan/go.rice/rice append --exec epik-health -i ./build

.PHONY: health
BINS+=health

testground:
	go build -tags testground -o /dev/null ./cmd/epik

.PHONY: testground
BINS+=testground

# MISC

buildall: $(BINS)

completions:
	./scripts/make-completions.sh epik
	./scripts/make-completions.sh epik-storage-miner
.PHONY: completions

install-completions:
	mkdir -p /usr/share/bash-completion/completions /usr/local/share/zsh/site-functions/
	install -C ./scripts/bash-completion/epik /usr/share/bash-completion/completions/epik
	install -C ./scripts/bash-completion/epik-storage-miner /usr/share/bash-completion/completions/epik-storage-miner
	install -C ./scripts/zsh-completion/epik /usr/local/share/zsh/site-functions/_epik
	install -C ./scripts/zsh-completion/epik-storage-miner /usr/local/share/zsh/site-functions/_epik-storage-miner

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

method-gen:
	(cd ./epikpond/front/src/chain && go run ./methodgen.go)

gen: type-gen method-gen

print-%:
	@echo $*=$($*)
