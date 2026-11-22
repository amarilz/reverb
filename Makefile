BINARY := reverb
HOST_TARGET := $(shell rustc -vV | sed -n 's/^host: //p')

TARGETS := x86_64-apple-darwin aarch64-apple-darwin \
           x86_64-unknown-linux-gnu aarch64-unknown-linux-gnu x86_64-pc-windows-msvc
release_x86_64-apple-darwin := darwin-amd64
release_aarch64-apple-darwin := darwin-arm64
release_x86_64-unknown-linux-gnu := linux-amd64
release_aarch64-unknown-linux-gnu := linux-arm64
release_x86_64-pc-windows-msvc := windows-amd64.exe

.PHONY: build run test vet lint tidy install clean release release-all help
build:
	cargo build --locked --release

run:
	cargo run --locked --release

test:
	cargo test --locked

vet:
	cargo check --locked --all-targets

lint:
	cargo fmt --all --check
	cargo clippy --locked --all-targets -- -D warnings

tidy:
	cargo generate-lockfile

install:
	cargo install --locked --path . --force

release: release-$(HOST_TARGET)

release-all: $(addprefix release-,$(TARGETS))

.PHONY: $(addprefix release-,$(TARGETS))
$(addprefix release-,$(TARGETS)): release-%:
	cargo build --locked --release --target $*
	mkdir -p dist
	cp target/$*/release/$(BINARY)$(if $(findstring windows,$*),.exe) dist/$(BINARY)-$(release_$*)

clean:
	cargo clean
	rm -rf dist

help:
	@echo 'build run test vet lint tidy install clean'
	@echo 'release (current platform), release-<rust-target>, release-all'
