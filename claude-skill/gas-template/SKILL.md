---
name: gas-template
description: >
  Reference documentation for the gas-template Go package
  (github.com/gasmod/gas-template) — template storage and retrieval for the Gas
  ecosystem. Use this skill when writing, reviewing, or debugging Go code that
  uses gas-template for storing and retrieving raw template content with
  in-memory, filesystem, database, or composite backends. Covers the memory, fs,
  db, and composite sub-packages, templatetest mock, gas.TemplateProvider
  implementation, sentinel errors, DI wiring, multi-database support
  (PostgreSQL, MySQL, SQLite), namespace isolation, embedded fs.FS registration,
  and migration registration with gas-migrate. Make sure to use this skill
  whenever working with template storage in the Gas ecosystem, even if the user
  doesn't explicitly mention gas-template — any code that imports
  gasmod/gas-template or references gas.TemplateProvider should trigger this
  skill.
---

# Gas Template Package Reference

Template storage service for the Gas ecosystem. Provides four
`gas.TemplateProvider` implementations — in-memory, filesystem, database, and
composite.

```
import template "github.com/gasmod/gas-template"
import "github.com/gasmod/gas-template/memory"
import tmplfs "github.com/gasmod/gas-template/fs"
import tmpldb "github.com/gasmod/gas-template/db"
import "github.com/gasmod/gas-template/composite"
import "github.com/gasmod/gas-template/templatetest"
```

## Backends

| Backend    | Package                  | Service name      | Use case                                     |
|------------|--------------------------|-------------------|----------------------------------------------|
| Memory     | `gas-template/memory`    | —                 | Development, testing, ephemeral storage      |
| Filesystem | `gas-template/fs`        | —                 | Static disk templates with runtime overlay   |
| Database   | `gas-template/db`        | `gas-template-db` | Persistent, multi-instance (Pg/MySQL/SQLite) |
| Composite  | `gas-template/composite` | —                 | Chain multiple providers with fallback reads |

Memory, filesystem, and composite implement `gas.TemplateProvider`.
Database implements both `gas.TemplateProvider` and `gas.Service`.

## TemplateProvider Interface

Defined in the gas core package:

```go
type TemplateProvider interface {
    Get(name string) ([]byte, error)
    List() ([]string, error)
    Register(name string, content []byte)
    RegisterFS(fsys fs.FS) error
}
```

## Sentinel Errors

The root `template` package defines:

```go
template.ErrTemplateNotFound // Get returns this when the template does not exist
template.IsNotFound(err)     // helper: errors.Is(err, ErrTemplateNotFound)
```

## Memory Backend

### Constructor

```go
func NewStore() *Store
```

Creates an empty in-memory store. Thread-safe via `sync.RWMutex`.

### Behavior

- `Get` returns `ErrTemplateNotFound` for missing keys.
- `List` returns all names sorted alphabetically.
- `Register` adds or overwrites a template.
- `RegisterFS` walks the fs.FS and registers all `.html` files. Non-`.html`
  files are skipped.

## Filesystem Backend

### Constructor

```go
func NewStore(dir string) *Store
```

Creates a store rooted at `dir`. Only `.html` files are recognized from disk.
Implements `io.Closer` — call `Close()` when done.

### Behavior

- **Sandboxed I/O:** Uses `os.OpenRoot` for path traversal protection.
- **Lazy initialization:** The root directory is opened on first `Get` or `List`
  call via `sync.Once`.
- **In-memory overlay:** `Register` and `RegisterFS` add to an overlay map.
  `Get` checks the overlay first, then falls back to disk.
- **List merges:** `List` returns deduplicated, sorted names from both disk and
  overlay.
- `Close` releases the `os.Root` handle.

## Database Backend

### Constructor

```go
func New(opts ...Option) func(gas.DatabaseProvider, gas.Logger, gas.MigrationManager) *Store
```

`New` captures options and returns a DI-injectable constructor. The returned
func receives `gas.DatabaseProvider`, `gas.Logger`, and `gas.MigrationManager`
from the DI container.

### Options

| Option                  | Description                                            |
|-------------------------|--------------------------------------------------------|
| `WithNamespace(ns string)` | Scope queries to a namespace (default: `"default"`) |

### Lifecycle (gas.Service)

| Method  | Signature    | Description                                                     |
|---------|--------------|-----------------------------------------------------------------|
| `Name`  | `() string`  | Returns `"gas-template-db"`                                     |
| `Init`  | `() error`   | Registers migration, selects sqlc adapter based on driver       |
| `Close` | `() error`   | No-op                                                           |

### Supported Drivers

| Driver              | Dialect    |
|---------------------|------------|
| `postgres`, `pgx`   | PostgreSQL |
| `mysql`             | MySQL      |
| `sqlite`, `sqlite3` | SQLite     |

Unsupported drivers cause `Init` to return an error.

### Migration

`Init` registers a single migration (version `20260322235959`, "create __gas_templates
table") with the `gas.MigrationManager`. The migration SQL is driver-specific
and embedded from `.sql` files. The `gas-migrate` service applies it during the
app startup sequence.

### Behavior

- `Get` returns `ErrTemplateNotFound` when no row matches (maps
  `sql.ErrNoRows` to the sentinel).
- `List` returns names sorted alphabetically (ORDER BY in the query).
- `Register` upserts via `INSERT ... ON CONFLICT`. Errors are logged but not
  returned (interface constraint).
- `RegisterFS` walks the fs.FS and upserts all `.html` files.
- `Exists(name string) (bool, error)` — checks whether a template exists.
- `Delete(name string) error` — deletes a template; returns
  `ErrTemplateNotFound` if no row was affected.

### Namespaces

Multiple `Store` instances can share one `__gas_templates` table by using different
namespaces. All queries are scoped to `(namespace, name)`.

```go
tmpldb.New(tmpldb.WithNamespace("emails"))
tmpldb.New(tmpldb.WithNamespace("pages"))
```

### Templates Table Schema

```sql
-- PostgreSQL
CREATE TABLE __gas_templates (
    id         BIGSERIAL PRIMARY KEY,
    namespace  TEXT        NOT NULL,
    name       TEXT        NOT NULL,
    content    BYTEA       NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (namespace, name)
);
CREATE INDEX idx_templates_namespace ON templates (namespace);
```

MySQL uses `BIGINT AUTO_INCREMENT`, `VARCHAR(255)`, `MEDIUMBLOB`, `DATETIME`.
SQLite uses `INTEGER PRIMARY KEY AUTOINCREMENT`, `BLOB`, `TEXT` for timestamps.

## Composite Backend

### Constructor

```go
func NewStore(writable gas.TemplateProvider, readers ...gas.TemplateProvider) *Store
```

### Behavior

- `Get` checks the writable provider first, then each reader in order. Returns
  `ErrTemplateNotFound` if none have the template.
- `List` merges names from all providers, deduplicated and sorted. If some
  providers return names and others return errors, the names are still returned.
  An error is returned only when all providers fail and no names are collected.
- `Register` and `RegisterFS` delegate to the writable provider only.

## Test Mock

The `templatetest` package provides `MockTemplate`, a configurable mock of
`gas.TemplateProvider` for use in unit tests.

```go
import "github.com/gasmod/gas-template/templatetest"
```

### MockTemplate

```go
type MockTemplate struct {
    GetFn        func(name string) ([]byte, error)
    ListFn       func() ([]string, error)
    RegisterFn   func(name string, content []byte)
    RegisterFSFn func(fsys fs.FS) error
    Calls        []Call
}
```

Each method delegates to its `Fn` field if set, otherwise returns zero value.
All calls are recorded in `Calls` for assertions. Thread-safe.

| Method                  | Description                               |
|-------------------------|-------------------------------------------|
| `Reset()`               | Clear all recorded calls                  |
| `CallCount(method) int` | Count calls by method name (e.g. `"Get"`) |

## DI Wiring Patterns

### Memory backend (dev/test)

```go
app := gas.NewApp(
    gas.WithServiceInstance[gas.TemplateProvider](memory.NewStore()),
)
```

### Database backend (production)

```go
app := gas.NewApp(
    gas.WithSingletonService[*database.Service](database.New()),
    gas.WithSingletonService[*migrate.Service](migrate.New()),
    gas.WithSingletonService[*tmpldb.Store](tmpldb.New()),
)
```

### Database backend with namespace

```go
app := gas.NewApp(
    gas.WithSingletonService[*database.Service](database.New()),
    gas.WithSingletonService[*migrate.Service](migrate.New()),
    gas.WithSingletonService[*tmpldb.Store](
        tmpldb.New(tmpldb.WithNamespace("emails")),
    ),
)
```

### Composite (disk + database fallback)

```go
app := gas.NewApp(
    gas.WithSingletonService[*database.Service](database.New()),
    gas.WithSingletonService[*migrate.Service](migrate.New()),
    gas.WithSingletonService[*tmpldb.Store](tmpldb.New()),
    // After DB store is available, wire the composite:
    gas.WithServiceInstance[gas.TemplateProvider](
        composite.NewStore(dbStore, fsStore),
    ),
)
```

### Consuming via gas.TemplateProvider

Services receive templates through the provider interface, never importing
gas-template backends directly:

```go
type Service struct {
    templates gas.TemplateProvider
}

func New(templates gas.TemplateProvider) *Service {
    return &Service{templates: templates}
}

func (s *Service) Init() error {
    content, err := s.templates.Get("emails/welcome.html")
    if err != nil {
        return err
    }
    // use content...
    return nil
}
```

### Swapping backends

Because all backends satisfy `gas.TemplateProvider`, switching requires only
changing the service registration — no consumer code changes:

```go
// Development
gas.WithServiceInstance[gas.TemplateProvider](memory.NewStore())

// Production
gas.WithSingletonService[*tmpldb.Store](tmpldb.New())
```

### Testing with MockTemplate

```go
mock := &templatetest.MockTemplate{}
mock.GetFn = func(name string) ([]byte, error) {
    return []byte("<h1>Hello</h1>"), nil
}

// inject mock as gas.TemplateProvider in tests
```
