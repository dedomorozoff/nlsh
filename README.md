# nlsh — Natural Language Shell

`nlsh` is a shell where you can communicate with the system in natural language.
A local lightweight LLM (GGUF via `llama.cpp`) is embedded directly into the binary
via CGO — no HTTP server and no external processes.

> Linux-first. macOS/Windows support is possible but not the primary focus.

## Features (MVP)

- `nlsh ask "..."` — explain what and how to do, without executing anything.
- `nlsh run "..."` — suggest a shell command and execute it after confirmation.
- `nlsh repl` — interactive mode with history and bash-like keybindings.
- Hard JSON contract for model responses.
- Safety policy gate against dangerous commands (`rm -rf /`, `mkfs`, fork bombs, etc.).

## REPL Features

### Bash-like Keybindings (via readline)

| Keybinding | Action |
|------------|--------|
| `Ctrl+A` | move to beginning of line |
| `Ctrl+E` | move to end of line |
| `Ctrl+U` | delete to beginning of line |
| `Ctrl+K` | delete to end of line |
| `Ctrl+L` | clear screen |
| `Ctrl+R` | reverse history search |
| `Ctrl+S` | forward history search |
| `Ctrl+P` | previous command |
| `Ctrl+N` | next command |
| `Alt+B` | backward by word |
| `Alt+F` | forward by word |
| `Alt+D` | delete word forward |
| `Ctrl+W` | delete word backward |

### Special Keys

| Key | Action |
|-----|--------|
| `Ctrl+C` | interrupt current operation (does not exit REPL) |
| `Ctrl+D` | exit REPL (EOF) |

### Slash Commands

| Command | Description |
|---------|-------------|
| `/help` | show help |
| `/exit` | exit REPL |
| `/cd [path]` | change directory |
| `/clear` | clear screen |
| `/pwd` | show current directory |
| `/history` | show history |
| `/bind keys` | show keybindings list |
| `!command` | execute command directly |

## Requirements

- Go 1.22+
- C/C++ тулчейн (gcc/clang, make, cmake)
- Git с поддержкой submodules
- GGUF-модель (например, Qwen2.5-3B-Instruct, Llama-3.2-3B-Instruct, Phi-3.5-mini)

## Building

```bash
git submodule update --init --recursive
make llama       # build static libllama
make build       # build nlsh binary
```

Details on flags/speedups — in `Makefile` (`make help`).

## Usage

```bash
nlsh --model /path/to/model.gguf repl
```

## Project Structure

```
cmd/nlsh/            CLI entry point
internal/cli/        cobra commands (ask, run, repl)
internal/llm/        CGO wrapper over llama.cpp
internal/prompt/     system prompt + JSON response contract
internal/policy/     safety gate (denylist + risk scoring)
internal/executor/   shell command execution
internal/config/     config loading
third_party/llama.cpp/  submodule with llama.cpp
```

## Status

Early MVP. See plan in `.cursor/plans/`.
