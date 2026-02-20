---
name: go-best-practices
description: Go idioms, patterns, and standard library best practices. Use when writing or reviewing Go code, implementing Go services or CLIs, working with .go files or go.mod, or when the user mentions Go or golang. Do NOT use for other programming languages.
metadata:
  type: skill
  triggers: go, golang, .go, go.mod
  token_budget: medium
  tags: language, go
---

# Go Best Practices

> **Priority**: 🟠 High

## When to Use This Skill

**Use when:**
- Writing Go code
- Implementing Go services and CLIs
- Working with Go idioms and patterns

**Don't use when:**
- Working with other languages

---

## Core Principles

- **Simplicity**: Keep it simple and readable
- **Explicit**: Prefer explicit over implicit
- **Error Handling**: Handle errors explicitly at every level
- **Composition**: Use composition over inheritance
- **Small Packages**: Write small, focused packages

---

## Modern Go idioms (Go 1.21–1.26)

Prefer language and library features from recent Go versions. Run `go fix ./...` (or `go fix -diff ./...` to preview) when upgrading the toolchain to apply safe modernizers (**go fix** modernizers: **Go 1.26+**). Each subsection below states the minimum Go version for that idiom.

### Loops

```go
// ✅ Good: range over integer (Go 1.22+) — avoid C-style loop
for range n {
    f()
}

// ❌ Bad: old 3-clause form
for i := 0; i < n; i++ {
    f()
}

// ✅ Good: Go 1.22+ — each iteration gets new variables; no need for x := x
for _, x := range items {
    use(x) // x is already a fresh copy per iteration
}

// ❌ Redundant in Go 1.22+: x := x was needed pre-1.22 to avoid closure bugs; remove it when targeting 1.22+
for _, x := range items {
    x := x
    use(x)
}
```

### min / max (Go 1.21+)

```go
// ✅ Good: use built-in min/max for clamping
x := min(max(f(), 0), 100)

// ❌ Bad: verbose if statements
x := f()
if x < 0 { x = 0 }
if x > 100 { x = 100 }
```

### strings.Cut (Go 1.18+)

```go
// ✅ Good: split on first separator
before, after, ok := strings.Cut(s, "=")
if ok {
    use(before, after)
}

// ❌ Bad: Index + manual slicing
eq := strings.IndexByte(s, '=')
if eq >= 0 {
    before, after := s[:eq], s[eq+1:]
    use(before, after)
}
```

### any instead of interface{} (Go 1.18+)

```go
// ✅ Good: any is predeclared (Go 1.18+)
func Store(key string, value any) error

// ❌ Bad
func Store(key string, value interface{}) error
```

### maps package (Go 1.21+)

```go
// ✅ Good: use maps.Keys, maps.Values instead of manual loops
keys := maps.Keys(m)
values := maps.Values(m)

// ❌ Bad: manual key slice
var keys []string
for k := range m {
    keys = append(keys, k)
}
```

### clear, slices.Concat, cmp.Or, reflect.TypeFor (Go 1.21–1.22+)

```go
// ✅ Good: clear (Go 1.21+) — empty a map or zero a slice in place
clear(m)
clear(slice)

// ✅ Good: slices.Concat (Go 1.22+) — concatenate multiple slices
out := slices.Concat(a, b, c)

// ✅ Good: cmp.Or (Go 1.22+) — first non-zero value (e.g. default)
name := cmp.Or(user.Name, "anonymous")

// ✅ Good: reflect.TypeFor[T]() (Go 1.22+) — type for T without allocation
t := reflect.TypeFor[MyStruct]()
// ❌ Avoid: reflect.TypeOf((*MyStruct)(nil)).Elem()
```

### fmt.Appendf and strings.Builder

```go
// ✅ Good: fmt.Appendf — build byte slice without extra allocation
buf := fmt.Appendf(nil, "%d items", n)

// ❌ Bad: []byte(fmt.Sprintf(...)) allocates string then copy

// ✅ Good: strings.Builder — concatenation in a loop
var b strings.Builder
for _, segment := range segments {
    b.WriteString(segment)
}
s := b.String()

// ❌ Bad: s += segment in loop (quadratic, DoS risk)
```

### new(expr) for pointer to value (Go 1.26+)

```go
// ✅ Good: new(expr) initializes to the given value
data, _ := json.Marshal(&RequestJSON{
    URL:      url,
    Attempts: new(10),
})

// ❌ Bad: two-step or helper
attempts := new(int)
*attempts = 10
// or a newInt(10) helper — replace with new(10)
```

Require at least Go 1.26 in the module (or `//go:build go1.26`) before using `new(expr)`.

---

## Error Handling

- **Check err before using other return values** (e.g. use `f` only after `if err != nil`).
- **Error wrapping with %w** for wrapped errors.
- **errors.AsType**: Go 1.26+.

```go
// ✅ Good: Always handle errors explicitly; check err before using other return values
result, err := doSomething()
if err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}
use(result) // safe: err already checked

// ✅ Good: Check err first — if err != nil, the other value may be nil (spec; compiler fixed in 1.25 to not delay nil checks)
f, err := os.Open(path)
if err != nil {
    return err
}
defer f.Close()
use(f.Name()) // never use f before checking err

// ❌ Bad: Using result before checking err (nil pointer risk)
f, err := os.Open(path)
name := f.Name() // panic if err != nil
if err != nil { return err }

// ✅ Good: Wrap errors with context
if err := db.Save(user); err != nil {
    return fmt.Errorf("saving user %s: %w", user.ID, err)
}

// ✅ Good: errors.AsType (Go 1.26+) — type-safe, faster than errors.As when applicable
var pe *os.PathError
if errors.AsType(err, &pe) {
    use(pe.Path)
}

// ✅ Good: errors.Join (Go 1.20+) — wrap multiple errors
if err1 != nil || err2 != nil {
    return errors.Join(err1, err2)
}

// ❌ Bad: Ignoring errors
result, _ := doSomething()
```

---

## Context Usage

```go
// ✅ Good: Pass context as first parameter
func FetchUser(ctx context.Context, id string) (*User, error) {
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
        return db.GetUser(ctx, id)
    }
}

// ✅ Good: Add timeouts to operations
ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()
```

### defer and time.Since (Go 1.22+ vet)

```go
// ✅ Good: defer the call to time.Since so it runs when the deferred function runs
t := time.Now()
defer func() { log.Println("elapsed:", time.Since(t)) }()

// ❌ Bad: time.Since(t) runs immediately, not when defer runs (go vet reports this)
defer log.Println(time.Since(t))
```

---

## Networking

- **net.JoinHostPort** for addresses. **go vet hostport**: Go 1.25+.

```go
// ✅ Good: net.JoinHostPort for addresses (IPv6-safe; go vet hostport in Go 1.25+)
addr := net.JoinHostPort(host, strconv.Itoa(port))
conn, err := net.Dial("tcp", addr)

// ❌ Bad: fmt.Sprintf("%s:%d", host, port) — fails for IPv6
addr := fmt.Sprintf("%s:%d", host, port)
```

---

## Structs and Interfaces

```go
// ✅ Good: Accept interfaces, return structs
type UserRepository interface {
    Get(ctx context.Context, id string) (*User, error)
    Save(ctx context.Context, user *User) error
}

// ✅ Good: Small, focused interfaces
type Reader interface {
    Read(p []byte) (n int, err error)
}

// ✅ Good: Embedding for composition
type UserService struct {
    repo UserRepository
    log  *slog.Logger
} // log/slog: Go 1.21+

// ✅ Good: Generic self-reference (Go 1.26+) — type parameter can refer to the generic type
type Adder[A Adder[A]] interface {
    Add(A) A
}
func algo[A Adder[A]](x, y A) A { return x.Add(y) }
```

---

## Concurrency

- **sync.WaitGroup.Go**: Go 1.25+. **sync.OnceFunc / OnceValue / OnceValues**: Go 1.21+. **go vet waitgroup** (misplaced Add): Go 1.25+.

```go
// ✅ Good: Use channels for communication
results := make(chan Result)
go func() {
    results <- doWork()
}()

// ✅ Good: Use WaitGroup for synchronization; call Add before launching goroutine
var wg sync.WaitGroup
for _, item := range items {
    wg.Add(1)
    go func(item Item) {
        defer wg.Done()
        process(item)
    }(item)
}
wg.Wait()

// ✅ Good: WaitGroup.Go (Go 1.25+) — Add + goroutine + Done in one call
var wg sync.WaitGroup
for _, item := range items {
    wg.Go(func() { process(item) })
}
wg.Wait()

// ✅ Good: sync.OnceFunc / OnceValue / OnceValues (Go 1.21+) — lazy init without manual sync.Once
var initDB = sync.OnceValues(func() (*DB, error) { return openDB() })
db, err := initDB()

// ✅ Good: Avoid goroutine leaks — don't return early while workers still send to unbuffered channel
// Either: use buffered channel, drain on error, or cancel context so workers can exit
ch := make(chan result, len(ws)) // buffered so early return doesn't block workers
for _, w := range ws {
    go func(w workItem) { ch <- process(w) }(w)
}
for range len(ws) { // range over int: Go 1.22+
    r := <-ch
    if r.err != nil { return nil, r.err }
    results = append(results, r.res)
}

// ❌ Bad: Unbuffered ch + early return = leaked goroutines (Go 1.26 goroutineleak profile can detect)
ch := make(chan result)
for _, w := range ws {
    go func() { ch <- process(w) }()
}
for range len(ws) {
    r := <-ch
    if r.err != nil { return nil, r.err } // remaining senders block forever
    ...
}

// ✅ Good: Use errgroup for error handling (golang.org/x/sync/errgroup)
g, ctx := errgroup.WithContext(ctx)
g.Go(func() error { return fetchUser(ctx) })
g.Go(func() error { return fetchOrders(ctx) })
if err := g.Wait(); err != nil {
    return err
}
```

---

## Testing

- **B.Loop()**: Go 1.26+. **T.ArtifactDir / go test -artifacts**: Go 1.26+.

```go
// ✅ Good: Table-driven tests
func TestAdd(t *testing.T) {
    tests := []struct {
        name     string
        a, b     int
        expected int
    }{
        {"positive", 1, 2, 3},
        {"negative", -1, -2, -3},
        {"zero", 0, 0, 0},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := Add(tt.a, tt.b)
            if got != tt.expected {
                t.Errorf("Add(%d, %d) = %d, want %d", 
                    tt.a, tt.b, got, tt.expected)
            }
        })
    }
}

// ✅ Good: Benchmark loop (Go 1.26+) — use for b.Loop() { ... } like for range n
func BenchmarkFoo(b *testing.B) {
    for b.Loop() {
        doWork()
    }
}

// ✅ Good: Test artifacts (Go 1.26+) — t.ArtifactDir(), go test -artifacts
func TestArtifacts(t *testing.T) {
    dir := t.ArtifactDir()
    os.WriteFile(filepath.Join(dir, "out.txt"), data, 0644)
}
```

---

## Naming Conventions

- **Packages**: Short, lowercase, no underscores
- **Variables**: camelCase, short in small scopes
- **Exported**: PascalCase for exported identifiers
- **Interfaces**: -er suffix when single method (Reader, Writer)
- **Getters**: No Get prefix (user.Name(), not user.GetName())

---

## Boundaries

- ✅ **Always**: Handle errors and check err before using other return values; use context; write table-driven tests; prefer modern idioms by version: min/max (1.21+), strings.Cut (1.18+), for range n (1.22+), maps package (1.21+), new(expr) (1.26+); use net.JoinHostPort for addresses; run `go fix ./...` after upgrading (go fix modernizers: Go 1.26+).
- ⚠️ **Ask First**: Adding dependencies, changing public APIs
- 🚫 **Never**: Ignore errors; use init() for complex logic; panic in libraries; use result of f, err := ... before checking err; use `for i := 0; i < n; i++` (use `for range n` from 1.22+); repeated `s += x` in loops (use strings.Builder); use net/http/httputil ReverseProxy.Director (use Rewrite; deprecated, unsafe); `defer log.Println(time.Since(t))` (defer the func so time.Since runs on exit: `defer func() { log.Println(time.Since(t)) }()`)
