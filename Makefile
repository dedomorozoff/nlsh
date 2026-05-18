LLAMA_DIR := third_party/llama.cpp
LLAMA_BUILD := $(LLAMA_DIR)/build

GO := go
GOFLAGS ?=
LDFLAGS ?= -s -w -X github.com/dedomorozoff/nlsh/internal/cli.Version=$(shell git describe --always --dirty 2>/dev/null || echo dev)

# По умолчанию собираем CPU-вариант. Через GPU=1 включаются ускорители.
GPU ?= 0
CMAKE_FLAGS := -DBUILD_SHARED_LIBS=OFF -DLLAMA_BUILD_TESTS=OFF -DLLAMA_BUILD_EXAMPLES=OFF -DLLAMA_BUILD_SERVER=OFF -DGGML_NATIVE=OFF -DGGML_CUDA=OFF
ifeq ($(GPU),cuda)
CMAKE_FLAGS += -DGGML_CUDA=ON
endif
ifeq ($(GPU),metal)
CMAKE_FLAGS += -DGGML_METAL=ON
endif
ifeq ($(GPU),vulkan)
CMAKE_FLAGS += -DGGML_VULKAN=ON
endif

# Ограничение параллелизма для сборки llama.cpp (чтобы не было out of memory)
LLAMA_JOBS ?= 2

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
	@echo "  make all         — собрать все платформы (Windows, Linux, macOS)"

.PHONY: submodule
submodule:
	git submodule update --init --recursive

# Windows-specific DLLs from MinGW
MINGW_BIN := /c/ProgramData/mingw64/mingw64/bin
WINDOWS_DLLS := $(MINGW_BIN)/libstdc++-6.dll $(MINGW_BIN)/libgcc_s_seh-1.dll $(MINGW_BIN)/libgomp-1.dll $(MINGW_BIN)/libwinpthread-1.dll

llama-prepare: submodule
ifdef OS
	powershell -Command "if (-not (Test-Path 'third_party/llama.cpp/build')) { New-Item -ItemType Directory -Path 'third_party/llama.cpp/build' }"
	cmake -G "MinGW Makefiles" -S $(LLAMA_DIR) -B $(LLAMA_BUILD) $(CMAKE_FLAGS)
	cmake --build $(LLAMA_BUILD) --config Release --parallel $(LLAMA_JOBS)
else
	cmake -S $(LLAMA_DIR) -B $(LLAMA_BUILD) $(CMAKE_FLAGS)
	cmake --build $(LLAMA_BUILD) --config Release --parallel $(LLAMA_JOBS)
endif

llama: llama-prepare

.PHONY: build
build:
ifdef OS
	powershell -Command "if (-not (Test-Path bin)) { New-Item -ItemType Directory -Path bin }"
	powershell -Command "go build -tags llama -ldflags '$(LDFLAGS)' -o bin/nlsh.exe ./cmd/nlsh"
	powershell -Command "if (Test-Path '$(MINGW_BIN)/libstdc++-6.dll') { Copy-Item '$(MINGW_BIN)/libstdc++-6.dll' bin/ -Force; Copy-Item '$(MINGW_BIN)/libgcc_s_seh-1.dll' bin/ -Force; Copy-Item '$(MINGW_BIN)/libgomp-1.dll' bin/ -Force; Copy-Item '$(MINGW_BIN)/libwinpthread-1.dll' bin/ -Force }"
else
	$(GO) build $(GOFLAGS) -tags llama -ldflags "$(LDFLAGS)" -o bin/nlsh ./cmd/nlsh
endif

.PHONY: build-stub
build-stub:
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o bin/nlsh ./cmd/nlsh

# Сборка для всех платформ (для локального создания релизов)
.PHONY: build-all
build-all: build-windows build-linux build-macos

.PHONY: build-windows
build-windows:
	powershell -Command "if (-not (Test-Path bin)) { New-Item -ItemType Directory -Path bin }"
	powershell -Command "go build -tags llama -ldflags '$(LDFLAGS)' -o bin/nlsh-windows-amd64.exe ./cmd/nlsh"
	powershell -Command "if (Test-Path '$(MINGW_BIN)/libstdc++-6.dll') { Copy-Item '$(MINGW_BIN)/libstdc++-6.dll' bin/ -Force; Copy-Item '$(MINGW_BIN)/libgcc_s_seh-1.dll' bin/ -Force; Copy-Item '$(MINGW_BIN)/libgomp-1.dll' bin/ -Force; Copy-Item '$(MINGW_BIN)/libwinpthread-1.dll' bin/ -Force }"

.PHONY: build-linux
build-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 CGO_CFLAGS="-I$(LLAMA_BUILD)/include" CGO_LDFLAGS="-L$(LLAMA_BUILD)/lib" $(GO) build -tags llama -ldflags "$(LDFLAGS)" -o bin/nlsh-linux-amd64 ./cmd/nlsh

.PHONY: build-macos
build-macos:
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 CGO_CFLAGS="-I$(LLAMA_BUILD)/include" CGO_LDFLAGS="-L$(LLAMA_BUILD)/lib" $(GO) build -tags llama -ldflags "$(LDFLAGS)" -o bin/nlsh-macos-amd64 ./cmd/nlsh
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 CGO_CFLAGS="-I$(LLAMA_BUILD)/include" CGO_LDFLAGS="-L$(LLAMA_BUILD)/lib" $(GO) build -tags llama -ldflags "$(LDFLAGS)" -o bin/nlsh-macos-arm64 ./cmd/nlsh

.PHONY: test
test:
	$(GO) test ./...

.PHONY: clean
clean:
ifdef OS
	if exist bin\ rmdir /s /q bin
	if exist $(LLAMA_BUILD) rmdir /s /q $(LLAMA_BUILD)
else
	rm -rf bin/ $(LLAMA_BUILD)
endif
