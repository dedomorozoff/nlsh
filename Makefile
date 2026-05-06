SHELL := /bin/bash

LLAMA_DIR := third_party/llama.cpp
LLAMA_BUILD := $(LLAMA_DIR)/build

GO := go
GOFLAGS ?=
LDFLAGS ?= -s -w -X github.com/nlsh/nlsh/internal/cli.Version=$(shell git describe --always --dirty 2>/dev/null || echo dev)

# По умолчанию собираем CPU-вариант. Через GPU=1 включаются ускорители.
GPU ?= 0
CMAKE_FLAGS := -DBUILD_SHARED_LIBS=OFF -DLLAMA_BUILD_TESTS=OFF -DLLAMA_BUILD_EXAMPLES=OFF -DLLAMA_BUILD_SERVER=OFF
ifeq ($(GPU),cuda)
CMAKE_FLAGS += -DGGML_CUDA=ON
endif
ifeq ($(GPU),metal)
CMAKE_FLAGS += -DGGML_METAL=ON
endif
ifeq ($(GPU),vulkan)
CMAKE_FLAGS += -DGGML_VULKAN=ON
endif

.PHONY: help
help:
	@echo "Targets:"
	@echo "  make submodule   — обновить submodule llama.cpp"
	@echo "  make llama       — собрать статическую libllama (CPU)"
	@echo "  make llama GPU=cuda|metal|vulkan — c GPU-ускорителем"
	@echo "  make build       — собрать nlsh с CGO (-tags llama)"
	@echo "  make build-stub  — собрать nlsh без llama.cpp (заглушка)"
	@echo "  make test        — go test без CGO"
	@echo "  make clean       — удалить build/ и бинари"

.PHONY: submodule
submodule:
	git submodule update --init --recursive

.PHONY: llama
llama: submodule
	cmake -S $(LLAMA_DIR) -B $(LLAMA_BUILD) $(CMAKE_FLAGS)
	cmake --build $(LLAMA_BUILD) --config Release -j

.PHONY: build
build:
	$(GO) build $(GOFLAGS) -tags llama -ldflags "$(LDFLAGS)" -o bin/nlsh ./cmd/nlsh

.PHONY: build-stub
build-stub:
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o bin/nlsh ./cmd/nlsh

.PHONY: test
test:
	$(GO) test ./...

.PHONY: clean
clean:
	rm -rf bin/ $(LLAMA_BUILD)
