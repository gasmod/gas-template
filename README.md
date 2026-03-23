# gas-template

Template storage service for the [Gas](https://github.com/gasmod/gas) ecosystem. Provides multiple `gas.TemplateProvider`
implementations — in-memory, filesystem, database, and composite — for storing and retrieving raw template content.

## Install

```bash
go get github.com/gasmod/gas-template
```

## Backends

| Backend    | Package                                    | Use case                                                 |
|------------|--------------------------------------------|----------------------------------------------------------|
| Memory     | `github.com/gasmod/gas-template/memory`    | Development, testing, ephemeral storage                  |
| Filesystem | `github.com/gasmod/gas-template/fs`        | Static templates on disk with runtime overlay            |
| Database   | `github.com/gasmod/gas-template/db`        | Persistent, multi-instance deployments (Pg/MySQL/SQLite) |
| Composite  | `github.com/gasmod/gas-template/composite` | Chain multiple providers with fallback reads             |

Memory, filesystem, and composite stores implement `gas.TemplateProvider`.
The database store also implements `gas.Service` (with DI, migrations, and lifecycle management).

## Usage

### Memory backend

```go
import "github.com/gasmod/gas-template/memory"

store := memory.NewStore()
store.Register("emails/welcome.html", []byte("<h1>Welcome</h1>"))

content, err := store.Get("emails/welcome.html")
```

### Filesystem backend

```go
import tmplfs "github.com/gasmod/gas-template/fs"

store := tmplfs.NewStore("./templates")
defer store.Close()

// Reads from disk; overlay takes precedence.
content, err := store.Get("home.html")

// Programmatic additions go to the in-memory overlay.
store.Register("dynamic.html", []byte("<p>Dynamic</p>"))
```

### Database backend

```go
package main

import (
    "github.com/gasmod/gas"
    database "github.com/gasmod/gas-database"
    migrate "github.com/gasmod/gas-migrate"
    tmpldb "github.com/gasmod/gas-template/db"
)

func main() {
    app := gas.NewApp(
        gas.WithSingletonService[*database.Service](database.New()),
        gas.WithSingletonService[*migrate.Service](migrate.New()),
        gas.WithSingletonService[*tmpldb.Store](tmpldb.New()),
        // ...
    )

    app.Run()
}
```

With a custom namespace:

```go
tmpldb.New(tmpldb.WithNamespace("emails"))
```

### Composite backend

Chain multiple providers — writes go to the first, reads fall back through all:

```go
import (
    "github.com/gasmod/gas-template/composite"
    "github.com/gasmod/gas-template/memory"
    tmplfs "github.com/gasmod/gas-template/fs"
)

writable := memory.NewStore()
disk := tmplfs.NewStore("./templates")
defer disk.Close()

store := composite.NewStore(writable, disk)

// Get checks writable first, then disk.
content, err := store.Get("page.html")

// Register goes to the writable provider only.
store.Register("override.html", []byte("<p>Override</p>"))
```

### Dependency injection

Services receive templates through `gas.TemplateProvider` via constructor injection:

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

### Registering templates from embedded files

All stores support registering templates from an `fs.FS`:

```go
import "embed"

//go:embed templates/*.html
var templateFS embed.FS

store.RegisterFS(templateFS)
```

Only `.html` files are registered; other extensions are skipped.

## Database Backends

The `db` package supports three database dialects. The correct dialect is selected
automatically based on the configured database driver:

| Driver              | Dialect    |
|---------------------|------------|
| `postgres`, `pgx`   | PostgreSQL |
| `mysql`             | MySQL      |
| `sqlite`, `sqlite3` | SQLite     |

The templates table migration is registered automatically with `gas-migrate` during `Init()`.

### Namespaces

Multiple `db.Store` instances can share the same table by using different namespaces:

```go
gas.WithSingletonService[*tmpldb.Store](tmpldb.New(tmpldb.WithNamespace("emails")))
```

The default namespace is `"default"`.

### Extra methods

The database store exposes two additional methods beyond `gas.TemplateProvider`:

```go
store.Exists("page.html")  // (bool, error)
store.Delete("page.html")  // error — returns template.ErrTemplateNotFound if missing
```

## Testing

The `templatetest` package provides a mock implementation of `gas.TemplateProvider`:

```go
import "github.com/gasmod/gas-template/templatetest"

mock := &templatetest.MockTemplate{}
mock.GetFn = func(name string) ([]byte, error) {
    return []byte("<h1>Hello</h1>"), nil
}

// pass mock as gas.TemplateProvider
// assert calls:
if mock.CallCount("Get") != 1 {
    t.Error("expected one Get call")
}
```

## Sentinel Errors

The root `template` package defines a sentinel error used by all backends:

```go
template.ErrTemplateNotFound // returned by Get when the template does not exist
template.IsNotFound(err)     // helper to check if an error is ErrTemplateNotFound
```
