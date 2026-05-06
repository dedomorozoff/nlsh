# nlsh — Natural Language Shell

`nlsh` — это shell, в которой можно общаться с системой на естественном языке.
Локальная легковесная LLM (GGUF через `llama.cpp`) встраивается прямо в бинарь
через CGO — без HTTP-сервера и без внешних процессов.

> Linux-first. Поддержка macOS/Windows возможна, но не основной приоритет.

## Возможности (MVP)

- `nlsh ask "..."` — объяснить, что и как сделать, ничего не выполняя.
- `nlsh run "..."` — предложить shell-команду и выполнить её после подтверждения.
- `nlsh repl` — интерактивный режим с историей.
- Жёсткий JSON-контракт ответа модели.
- Safety policy gate против опасных команд (`rm -rf /`, `mkfs`, fork-бомбы и т. п.).

## Требования

- Go 1.22+
- C/C++ тулчейн (gcc/clang, make, cmake)
- Git с поддержкой submodules
- GGUF-модель (например, Qwen2.5-3B-Instruct, Llama-3.2-3B-Instruct, Phi-3.5-mini)

## Сборка

```bash
git submodule update --init --recursive
make llama       # собрать статическую libllama
make build       # собрать бинарь nlsh
```

Подробности по флагам/ускорителям — в `Makefile` (`make help`).

## Запуск

```bash
nlsh --model /path/to/model.gguf repl
```

## Структура проекта

```
cmd/nlsh/            точка входа CLI
internal/cli/        cobra-команды (ask, run, repl)
internal/llm/        CGO-обёртка над llama.cpp
internal/prompt/     системный промпт + JSON-контракт ответа
internal/policy/     safety gate (denylist + risk scoring)
internal/executor/   запуск shell-команд
internal/config/     загрузка конфига
third_party/llama.cpp/  submodule с llama.cpp
```

## Статус

Ранний MVP. См. план в `.cursor/plans/`.
