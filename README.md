## cargo-graph-analyzer
Анализатор графа зависимостей для пакетов Rust (Cargo).

### Сборка

```bash
go build ./...
```

### Использование

- **Crates.io**:

```bash
depgraph get <name> repo <version> <max-depth>
```

  - берёт зависимости с `https://crates.io/api/v1/crates/<name>/<version>/dependencies`;
  - автоматически раскрывает транзитивные зависимости (DFS без рекурсии).

- **Тестовый файл**:

```bash
depgraph get <name> test </abs/path/to/graph.txt> <max-depth>
```

  - файл формата `A:B,C` или `A -> B C`, комментарии `#`/`;`.

`max-depth = 0` означает «без ограничения». Любая команда выводит дерево зависимостей, повторяющиеся узлы и найденные циклы.

### Возможности

- Получение зависимостей напрямую без клонирования git‑репозиториев.
- Построение полного графа (включая транзитивные зависимости) алгоритмом DFS без рекурсии и с ограничением глубины.
- Обнаружение повторяющихся узлов (один пакет фигурирует у нескольких родителей) и циклических цепочек.
- ASCII‑дерево с указанием глубины каждого узла.
- Тестовый режим с кастомными графами из текстового файла.

### Основные компоненты

- `internal/cli/dependencies.go` — CLI-команда `get`, валидация аргументов, выбор режима (`repo`/`test`).
- `internal/crates/api.go` — клиент crates.io: HTTP‑запросы, кеширование, подбор версий с помощью `github.com/Masterminds/semver/v3`, построение смежности.
  - `BuildAdjacencyFromRegistry` — формирует граф `crate@version -> []deps`.
  - `resolveVersion` — подбирает конкретную версию зависимости по ограничению `req`.
- `internal/graph/graph.go` — анализ и вывод графа:
  - `AnalyzeAndRender` — нерекурсивный DFS, генерация дерева, список повторов и циклов.
- Тестовый парсер (`parseTestGraphFile`) — читает формат `A:B,C` или `A -> B C`.

### Пример работы

- **Вывод для serde v1.0.228**:

Dependency graph for serde@1.0.228 (max-depth 3):

serde@1.0.228
`-- serde_core@1.0.228 (depth=1)
    `-- serde_derive@1.0.228 (depth=2)
        |-- proc-macro2@1.0.103 (depth=3)
        |-- quote@1.0.42 (depth=3)
        `-- syn@2.0.110 (depth=3)

Repeated nodes:
 - none
