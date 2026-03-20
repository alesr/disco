# Disco Go Style Guide

Version: 2026-03
Owner: Disco engineering
Language scope: Go

This guide is optimized for both humans and RAG review.

- `Rule (Must)`: objective and enforceable
- `Recommendation (Should)`: advisory and non-blocking

CI interpretation:

- `block`: can fail CI when evidence and confidence gates pass
- `warn`: non-blocking warning
- `info`: informational suggestion

Disco interpretation:

- `Rule (Must)` maps to stronger failure checks
- `Recommendation (Should)` maps to advisory checks only

## Retrieval Structure

Each item follows the same structure so retrieval chunks are predictable.

- `ID`: stable identifier
- `Type`: `rule` or `recommendation`
- `Enforcement`: `block`, `warn`, or `info`
- `Taxonomy`: machine-friendly category for Disco mapping
- `Skill_Primary`: required Disco skill voice for checks
- `Difficulty_Base`: required base check difficulty (`Trivial`..`Impossible`)
- `Difficulty_Min` and `Difficulty_Max`: optional bounds
- `Scope`: where the guidance applies
- `Statement`: concise policy
- `Rationale`: why this exists
- `Bad` and `Good`: concrete examples
- `Notes`: edge cases

## Guidance Set


### Error and Logging

## RG-ERR-001 - Wrap propagated errors with context

Type: rule
Enforcement: block
Taxonomy: error_handling
Skill_Primary: Logic
Difficulty_Base: Challenging
Scope: production code and library code

### Statement

When returning an upstream error to a higher layer, wrap it with context using `%w`.

### Rationale

Context makes logs and traces actionable without losing root-cause identity.

### Bad

```go
if err != nil {
	return err
}
```

### Good

```go
if err != nil {
	return fmt.Errorf("could not load config: %w", err)
}
```

### Notes

At process boundaries where context is already attached one layer above, returning as-is can be acceptable.

## RG-ERR-002 - Handle an error once

Type: rule
Enforcement: block
Taxonomy: reliability
Skill_Primary: Volition
Difficulty_Base: Challenging
Scope: all code

### Statement

Do not both log and return the same error unless there is explicit deduplication intent.

### Rationale

Double handling creates noisy logs and misleads incident timelines.

### Bad

```go
if err != nil {
	log.Printf("failed: %v", err)
	return fmt.Errorf("save user: %w", err)
}
```

### Good

```go
if err != nil {
	return fmt.Errorf("could not save user: %w", err)
}
```

### Notes

Log at ownership boundaries where the error is no longer returned.

## RG-ERR-003 - Never ignore returned errors

Type: rule
Enforcement: block
Taxonomy: reliability
Skill_Primary: Volition
Difficulty_Base: Challenging
Scope: all code

### Statement

Handle or intentionally document any error return value.

### Rationale

Ignored errors silently corrupt behavior and data integrity.

### Bad

```go
_ = file.Close()
doWork()
```

### Good

```go
if err := file.Close(); err != nil {
	return fmt.Errorf("could not close file: %w", err)
}
if err := doWork(); err != nil {
	return fmt.Errorf("could not run work: %w", err)
}
```

### Notes

`_ = fn()` is allowed only when the call is explicitly best-effort and documented in code.

## RG-ERR-004 - Use "could not ..." phrasing for created errors

Type: rule
Enforcement: warn
Taxonomy: readability
Skill_Primary: Logic
Difficulty_Base: Medium
Scope: all error creation sites (`fmt.Errorf`, `errors.New`)

### Statement

When creating new errors, use a concise `"could not <action>"` prefix.
For wrapped errors, use `fmt.Errorf("could not <action>: %w", err)`.
For non-wrapping errors, use `errors.New("could not <action>")`.

### Rationale

Consistent phrasing improves scanability, grep-ability, and operator ergonomics across logs, tests, and CLI output.

### Bad

```go
return fmt.Errorf("failed to load config: %w", err)
return fmt.Errorf("error while saving user: %w", err)
return errors.New("invalid request body")
```

### Good

```go
return fmt.Errorf("could not load config: %w", err)
return fmt.Errorf("could not save user: %w", err)
return errors.New("could not parse request body")
```

### Notes

- keep messages lowercase and without trailing punctuation
- prefer action-oriented wording (`load`, `parse`, `save`, `start`, `close`)
- if an existing exported sentinel error has established wording, keep it unchanged for compatibility

## RG-ERR-005 - Use `fmt.Errorf` and `errors.New` for their intended purpose

Type: rule
Enforcement: warn
Taxonomy: error_handling
Skill_Primary: Logic
Difficulty_Base: Medium
Scope: all error creation sites

### Statement

Use `fmt.Errorf` when formatting context and/or wrapping an underlying error.
Use `errors.New` when creating a fixed error message with no dynamic values and no wrapped cause.

### Rationale

Choosing the right constructor makes intent explicit, preserves error chains when needed, and avoids unnecessary formatting overhead.

### Bad

```go
if err := loadConfig(); err != nil {
	return errors.New("could not load config")
}

if token == "" {
	return fmt.Errorf("could not parse token")
}
```

### Good

```go
if err := loadConfig(); err != nil {
	return fmt.Errorf("could not load config: %w", err)
}

if token == "" {
	return errors.New("could not parse token")
}
```

### Notes

- when wrapping an error, always preserve the cause with `%w`
- if a message needs interpolated values but no wrapped cause, `fmt.Errorf` is acceptable
- prefer `errors.New` for stable sentinel-style messages and static guard failures

## RG-ERR-006 - Handle returned errors explicitly before returning

Type: rule
Enforcement: warn
Taxonomy: error_handling
Skill_Primary: Logic
Difficulty_Base: Medium
Scope: functions that call error-returning helpers

### Statement

When calling a helper that returns an error, do not directly return the call expression.
Assign the result, check `err`, and return an explicit wrapped error with context.

### Rationale

Explicit handling improves readability, makes ownership clear, and keeps error messages consistent and actionable.

### Bad

```go
func run() error {
	return anotherFunc()
}
```

### Good

```go
func run() error {
	if err := anotherFunc(); err != nil {
		return fmt.Errorf("could not run anotherFunc: %w", err)
	}

	return nil
}
```

### Also Good

```go
func load() (Config, error) {
	cfg, err := readConfig()
	if err != nil {
		return Config{}, fmt.Errorf("could not read config: %w", err)
	}

	return cfg, nil
}
```

### Notes

- this rule complements wrapping guidance; prefer explicit context at the current ownership boundary
- for tiny pass-through adapter functions where context would be duplicated verbatim, direct return can be acceptable if documented

## RG-ERR-007 - If an error is not returned, surface it explicitly

Type: recommendation
Enforcement: info
Taxonomy: reliability
Skill_Primary: Volition
Difficulty_Base: Easy
Scope: call sites where returned errors are intentionally not propagated

### Statement

If an error is intentionally not returned to the caller, surface it via logging or `stderr` output.

### Rationale

Dropped errors hide failures and make troubleshooting expensive. Explicit surfacing preserves observability when propagation is not possible or not desired.

### Bad

```go
if err := cleanupTmpDir(); err != nil {
	// ignore
}
```

### Good

```go
if err := cleanupTmpDir(); err != nil {
	log.Printf("Failed to cleanup temp directory: %v", err)
}
```

### Also Good

```go
if err := flushMetrics(); err != nil {
	fmt.Fprintf(os.Stderr, "Failed to flush metrics: %v\n", err)
}
```

### Notes

- this applies only when not returning the error is intentional
- follow logging rules for message format (for example, `Failed ...` for log lines)
- avoid both logging and returning the same error unless duplication is intentional and documented

## RG-ERR-008 - Keep error strings lowercase; capitalize log messages

Type: rule
Enforcement: block
Taxonomy: readability
Skill_Primary: Logic
Difficulty_Base: Easy
Scope: error creation and logging

### Statement

Start returned error strings with a lowercase letter.
Start failure log messages with a capitalized prefix (`Failed ...`).

### Rationale

Go error style expects lowercase error strings, while capitalized logs improve scanability in operational output.

### Bad

```go
return errors.New("Could not load config")
return fmt.Errorf("Failed to read file: %w", err)
log.Printf("failed to load config: %v", err)
```

### Good

```go
return errors.New("could not load config")
return fmt.Errorf("could not read file: %w", err)
log.Printf("Failed to load config: %v", err)
```

### Notes

- applies to `errors.New` and `fmt.Errorf` messages returned as `error`
- do not add trailing punctuation to error strings
- this rule complements `RG-ERR-004` and `RG-LOG-001`

## RG-LOG-001 - Start error log messages with "Failed" (capital F)

Type: rule
Enforcement: warn
Taxonomy: readability
Skill_Primary: Logic
Difficulty_Base: Easy
Scope: log messages for errors and failures

### Statement

When logging an error, start the log message with `Failed` (capitalized), followed by the action and contextual fields.

### Rationale

Consistent log prefixes improve searchability and make failure lines stand out in noisy output.

### Bad

```go
log.Printf("failed to load config: %v", err)
logger.Error("could not connect to db", "err", err)
```

### Good

```go
log.Printf("Failed to load config: %v", err)
logger.Error("Failed to connect to db", "err", err)
```

### Notes

- this rule applies to logs only; returned errors should follow the `could not ...` conventions in error rules
- keep log text concise and action-oriented (`Failed to load ...`, `Failed to start ...`)

### Context and Concurrency

## RG-CTX-001 - Pass context as the first argument

Type: rule
Enforcement: block
Taxonomy: api_contract
Skill_Primary: Interfacing
Difficulty_Base: Challenging
Scope: functions doing IO, RPC, db, or long-running work

### Statement

Use `context.Context` as the first parameter when cancellation or deadlines matter.

### Rationale

Consistent signatures improve composability and cancellation safety.

### Bad

```go
func FetchUser(id string, ctx context.Context) (User, error) { ... }
```

### Good

```go
func FetchUser(ctx context.Context, id string) (User, error) { ... }
```

### Notes

Pure helper functions without external effects do not require context.

## RG-CTX-002 - Do not store context in structs

Type: rule
Enforcement: block
Taxonomy: api_contract
Skill_Primary: Interfacing
Difficulty_Base: Challenging
Scope: all code

### Statement

Do not keep `context.Context` as a struct field.

### Rationale

Stored contexts outlive request scope and cause cancellation bugs.

### Bad

```go
type Service struct {
	ctx context.Context
}
```

### Good

```go
type Service struct {
	store Store
}

func (s *Service) Run(ctx context.Context) error { ... }
```

## RG-CON-001 - Avoid fire-and-forget goroutines

Type: rule
Enforcement: block
Taxonomy: reliability
Skill_Primary: Volition
Difficulty_Base: Challenging
Scope: production code

### Statement

Every goroutine must have ownership, shutdown behavior, and error strategy.

### Rationale

Untracked goroutines leak resources and hide failures.

### Bad

```go
go syncCache(ctx)
```

### Good

```go
group, ctx := errgroup.WithContext(ctx)
group.Go(func() error {
	return syncCache(ctx)
})
if err := group.Wait(); err != nil {
	return fmt.Errorf("could not sync cache: %w", err)
}
```

## RG-CON-002 - Prefer unbuffered or size-1 channels by default

Type: recommendation
Enforcement: info
Taxonomy: readability
Skill_Primary: Composure
Difficulty_Base: Medium
Scope: channel design

### Statement

Start with unbuffered channels or a buffer of one unless a larger buffer is measured and justified.

### Rationale

Small buffers make backpressure and ordering easier to reason about.

### Bad

```go
events := make(chan Event, 2048)
```

### Good

```go
events := make(chan Event)
// or
events := make(chan Event, 1)
```

### Notes

High-throughput pipelines may need larger buffers with benchmark evidence.

### API and Types

## RG-API-001 - Define interfaces at point of use

Type: recommendation
Enforcement: info
Taxonomy: api_contract
Skill_Primary: Interfacing
Difficulty_Base: Medium
Scope: package design

### Statement

Place interfaces in the consuming package, not the implementation package.

### Rationale

This keeps abstractions minimal and avoids speculative contracts.

### Bad

```go
// package store
type UserStore interface {
	Get(ctx context.Context, id string) (User, error)
}
```

### Good

```go
// package service
type userGetter interface {
	Get(ctx context.Context, id string) (User, error)
}
```

## RG-API-002 - Keep interfaces small

Type: recommendation
Enforcement: info
Taxonomy: api_contract
Skill_Primary: Interfacing
Difficulty_Base: Medium
Scope: public and private APIs

### Statement

Prefer focused interfaces with one to three methods.

### Rationale

Small interfaces reduce coupling and simplify tests.

### Bad

```go
type Repository interface {
	Get(context.Context, string) (User, error)
	List(context.Context) ([]User, error)
	Create(context.Context, User) error
	Update(context.Context, User) error
	Delete(context.Context, string) error
}
```

### Good

```go
type userReader interface {
	Get(context.Context, string) (User, error)
}
```

## RG-STR-001 - Use keyed fields in struct literals

Type: rule
Enforcement: block
Taxonomy: api_contract
Skill_Primary: Interfacing
Difficulty_Base: Challenging
Scope: struct initialization

### Statement

Use field names in struct literals except for local tests with tiny private structs.

### Rationale

Keyed literals survive field reordering and are self-documenting.

### Bad

```go
cfg := Config{"localhost", 8080, true}
```

### Good

```go
cfg := Config{
	Host:    "localhost",
	Port:    8080,
	Enabled: true,
}
```

## RG-STR-002 - Avoid embedding concrete types in public structs

Type: rule
Enforcement: warn
Taxonomy: api_contract
Skill_Primary: Interfacing
Difficulty_Base: Medium
Scope: exported structs

### Statement

Do not embed concrete types in exported structs unless composition is the intended API.

### Rationale

Embedding leaks implementation details into public contracts.

### Bad

```go
type API struct {
	http.Client
}
```

### Good

```go
type API struct {
	client *http.Client
}
```

### Naming, Variables, Functions, and Imports

## RG-NAM-001 - Use idiomatic MixedCaps naming

Type: rule
Enforcement: block
Taxonomy: readability
Skill_Primary: Composure
Difficulty_Base: Challenging
Scope: identifiers

### Statement

Use idiomatic Go names. Avoid `snake_case` in identifiers.

### Rationale

Consistent naming lowers cognitive load.

### Bad

```go
var user_id string
```

### Good

```go
var userID string
```

### Notes

`snake_case` is acceptable in JSON and DB tags.

## RG-NAM-002 - Avoid vague package names

Type: recommendation
Enforcement: info
Taxonomy: readability
Skill_Primary: Composure
Difficulty_Base: Medium
Scope: package names

### Statement

Avoid generic names such as `util`, `common`, `helper`, and `types`.

### Rationale

Specific names improve discoverability.

### Bad

```go
// package util
func NormalizeUserID(raw string) string { ... }
```

### Good

```go
// package userfmt
func NormalizeUserID(raw string) string { ... }
```

## RG-VAR-001 - Prefer zero-value declarations when initialization is default

Type: rule
Enforcement: warn
Taxonomy: readability
Skill_Primary: Composure
Difficulty_Base: Medium
Scope: local variables and package-level declarations

### Statement

When a variable is intentionally initialized to its type zero value, declare it with `var` instead of assigning an explicit default literal.

### Rationale

Go zero values are a language feature, not a fallback. Using `var` signals intent clearly, avoids redundant literals, and reduces noisy diffs when types evolve.

### Bad

```go
enabled := false
count := 0
name := ""
items := []string(nil)
client := (*http.Client)(nil)
```

### Good

```go
var enabled bool
var count int
var name string
var items []string
var client *http.Client
```

### Notes

- this rule applies across types: booleans, numbers, strings, pointers, interfaces, slices, maps, channels, and structs
- use `:=` when assigning a meaningful non-zero initial value

## RG-VAR-002 - Group related zero-value declarations in a var block

Type: recommendation
Enforcement: info
Taxonomy: readability
Skill_Primary: Composure
Difficulty_Base: Medium
Scope: function bodies and local state setup

### Statement

When multiple variables are introduced together and share purpose or lifecycle, group them in a `var (...)` block with explicit types.

### Rationale

A grouped declaration makes related state obvious at a glance, improves scanability, and prevents repetitive zero-value literals.

### Bad

```go
hardSeen := false
hardStreak := 0
softStreak := 0
checksSinceEvent := 0
```

### Good

```go
var (
	hardSeen         bool
	hardStreak       int
	softStreak       int
	checksSinceEvent int
)
```

### Also Good

```go
var (
	retryable   bool
	retryCount  int
	lastErr     error
	buffer      []byte
	workerReady chan struct{}
)
```

### Notes

- group only variables that are semantically related; avoid oversized declaration blocks
- keep alignment readable, but prioritize clarity over strict vertical formatting
- if one variable needs a non-zero initializer, declare it separately near first use

## RG-FNC-001 - Prefer early returns to reduce nesting

Type: recommendation
Enforcement: info
Taxonomy: readability
Skill_Primary: Composure
Difficulty_Base: Medium
Scope: function bodies

### Statement

Guard invalid states first, then keep the main path flat.

### Rationale

Flatter control flow improves readability and review speed.

### Bad

```go
if req != nil {
	if req.User != nil {
		return process(req.User)
	}
}
return errors.New("invalid request")
```

### Good

```go
if req == nil || req.User == nil {
	return errors.New("invalid request")
}
return process(req.User)
```

## RG-FNC-002 - Avoid naked returns

Type: rule
Enforcement: block
Taxonomy: readability
Skill_Primary: Composure
Difficulty_Base: Challenging
Scope: non-trivial functions

### Statement

Return explicit values instead of naked returns.

### Rationale

Explicit returns reduce ambiguity and refactor risk.

### Bad

```go
func parse(input string) (out string, err error) {
	if input == "" {
		err = errors.New("could not parse input")
		return
	}

	out = strings.TrimSpace(input)
	return
}
```

### Good

```go
func parse(input string) (string, error) {
	if input == "" {
		return "", errors.New("could not parse input")
	}

	return strings.TrimSpace(input), nil
}
```

## RG-IMP-001 - Keep imports grouped and ordered

Type: rule
Enforcement: block
Taxonomy: readability
Skill_Primary: Composure
Difficulty_Base: Challenging
Scope: file imports

### Statement

Use three groups separated by blank lines: standard library, third-party, local module imports.

### Rationale

Predictable import layout reduces merge churn and improves scanability.

### Bad

```go
import (
	"github.com/alesr/disco/internal/review"
	"fmt"
	"github.com/stretchr/testify/require"
	"context"
)
```

### Good

```go
import (
	"context"
	"fmt"

	"github.com/stretchr/testify/require"

	"github.com/alesr/disco/internal/review"
)
```

### Testing

## RG-TST-001 - Test naming conventions

Type: rule
Enforcement: block
Taxonomy: readability
Skill_Primary: Composure
Difficulty_Base: Challenging
Scope: tests

### Statement

Use identifier-based test names:

- function under test: `Test<FunctionName>`
- method under test: `Test<StructName>_<MethodName>`

Keep scenario detail in `t.Run(...)` labels rather than in the top-level test function name.

### Rationale

Readable and consistent test names make ownership and failure location immediately actionable in `go test` output.

### Bad

```go
func Test_for_parser(t *testing.T) { ... }
func TestManagerRun(t *testing.T) { ... }
func TestHandlesErrors(t *testing.T) { ... }
func TestRunMethod(t *testing.T) { ... }
```

### Good

```go
func TestParseDiff(t *testing.T) { ... }
func TestManager_Run(t *testing.T) { ... }
func TestReviewDiff(t *testing.T) { ... }
func TestManager_ReviewDiffStream(t *testing.T) { ... }
```

### Notes

- if multiple behaviors are covered for one target, keep one top-level test and split scenarios with `t.Run`
- for constructors, prefer `TestNew<TypeName>` (for example, `TestNewManager`)

## RG-TST-002 - Use table tests when variation is primarily input/output

Type: recommendation
Enforcement: info
Taxonomy: readability
Skill_Primary: Composure
Difficulty_Base: Easy
Scope: tests

### Statement

Use table-driven tests when cases share the same setup and assertion flow, and only inputs/expected outputs vary.

### Rationale

Table tests reduce repetition and make coverage easier to audit when variation is minimal.

### Bad

```go
func TestNormalizeA(t *testing.T) { ... }
func TestNormalizeB(t *testing.T) { ... }
func TestNormalizeC(t *testing.T) { ... }
```

### Good

```go
func TestNormalize(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "trim spaces", in: "  a  ", want: "a"},
		{name: "collapse tabs", in: "a\tb", want: "a b"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := Normalize(tt.in)
			require.Equal(t, tt.want, got)
		})
	}
}
```

### Notes

- if setup or assertions differ significantly per case, prefer separate tests or focused `t.Run` groups

## RG-TST-003 - Use `t.Run` for meaningful scenario grouping

Type: recommendation
Enforcement: info
Taxonomy: readability
Skill_Primary: Composure
Difficulty_Base: Easy
Scope: tests

### Statement

Use `t.Run` when scenarios are meaningfully distinct, when setup is shared, or when grouped output improves diagnosis.

### Rationale

Subtests provide structured output and isolate behavior cases without over-fragmenting top-level test functions.

### Bad

```go
func TestValidate(t *testing.T) {
	gotA := Validate("ok")
	require.True(t, gotA)

	gotB := Validate("")
	require.False(t, gotB)
}
```

### Good

```go
func TestValidate(t *testing.T) {
	t.Run("valid input", func(t *testing.T) {
		require.True(t, Validate("ok"))
	})

	t.Run("empty input", func(t *testing.T) {
		require.False(t, Validate(""))
	})
}
```

### Notes

- do not create subtests for one-off assertions with no scenario value
- keep subtest names behavior-focused and concise

## RG-TST-004 - Use t.Parallel only when safe

Type: recommendation
Enforcement: info
Taxonomy: reliability
Skill_Primary: Volition
Difficulty_Base: Easy
Scope: tests

### Statement

Prefer `t.Parallel()` for independent tests that do not share mutable state or global process state.

### Rationale

Parallel tests improve feedback speed and push code toward concurrency-safe design. Unsafe parallelism causes flakes, so use it deliberately.

### Bad

```go
func TestGlobalConfig(t *testing.T) {
	t.Parallel()
	os.Setenv("MODE", "test")
	...
}
```

### Good

```go
func TestParser_Parse(t *testing.T) {
	t.Parallel()
	...
}
```

### Notes

- avoid parallelizing tests that mutate environment variables, working directory, global singletons, or shared files
- if a test is not safe to parallelize, document why

## RG-TST-005 - Use assert and require from testify

Type: recommendation
Enforcement: info
Taxonomy: readability
Skill_Primary: Composure
Difficulty_Base: Medium
Scope: tests

### Statement

Use `require` for preconditions and `assert` for value checks.

### Rationale

Consistent assertion style improves readability and failure quality.

### Bad

```go
func TestParse(t *testing.T) {
	if err := setup(); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	if got := Parse("x"); got != "x" {
		t.Fatalf("unexpected value: %s", got)
	}
}
```

### Good

```go
func TestParse(t *testing.T) {
	require.NoError(t, setup())
	assert.Equal(t, "x", Parse("x"))
}
```

### Build and Tooling

## RG-BLD-001 - Declare `.PHONY` immediately above each Make target

Type: rule
Enforcement: warn
Taxonomy: reliability
Skill_Primary: Composure
Difficulty_Base: Easy
Scope: `Makefile` targets

### Statement

For every non-file Make target, add a matching `.PHONY` line directly above the target definition.

### Rationale

Explicit `.PHONY` declarations prevent accidental target/file name collisions and make target intent obvious during maintenance.

### Bad

```make
fmt:
	go fmt ./...

test:
	go test ./...
```

### Good

```make
.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: test
test:
	go test ./...
```

### Notes

- applies to command targets such as `fmt`, `lint`, `test`, `vet`, `build`, and `clean`
- file-producing targets may be omitted from `.PHONY` when Make should track file timestamps

## RG-BLD-002 - Include baseline quality targets in every Makefile

Type: rule
Enforcement: block
Taxonomy: reliability
Skill_Primary: Volition
Difficulty_Base: Easy
Scope: repository `Makefile`

### Statement

Every repository `Makefile` must define, at minimum, these four targets with matching `.PHONY` entries:

- `fmt` -> `gofmt -w .`
- `test` -> `go test -race -count=1 -v ./...`
- `vet` -> `go vet ./...`
- `vulncheck` -> `govulncheck ./...`

### Rationale

A fixed baseline target set makes local quality gates predictable for humans, CI, and coding agents.

### Bad

```make
.PHONY: test
test:
	go test ./...

.PHONY: lint
lint:
	golangci-lint run
```

### Good

```make
.PHONY: fmt
fmt:
	gofmt -w .

.PHONY: test
test:
	go test -race -count=1 -v ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: vulncheck
vulncheck:
	govulncheck ./...
```

### Notes

- additional targets are encouraged (`all`, `lint`, `bench`, `coverage`), but these four are mandatory
- if `govulncheck` is unavailable in a given environment, install it rather than removing the target

## RG-BLD-003 - Every Makefile must provide a `help` target

Type: rule
Enforcement: block
Taxonomy: readability
Skill_Primary: Composure
Difficulty_Base: Easy
Scope: repository `Makefile`

### Statement

Every repository `Makefile` must include a `help` target and set it as the default goal.
The `help` target should print available targets and short descriptions.

### Rationale

A default `help` target improves discoverability for humans and agents and reduces command misuse.

### Bad

```make
.PHONY: test
test:
	go test ./...
```

### Good

```make
.DEFAULT_GOAL := help

.PHONY: help
help:
	@grep -E '^[a-zA-Z0-9_/%\-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort

.PHONY: test
test: ## Run tests
	go test -race -count=1 -v ./...
```

### Notes

- keep target descriptions in `##` comments so `help` output stays useful
- this rule complements baseline target requirements in `RG-BLD-002`

### Performance

## RG-PRF-001 - Prefer strconv over fmt for scalar conversion

Type: recommendation
Enforcement: info
Taxonomy: performance
Skill_Primary: Perception
Difficulty_Base: Medium
Scope: hot paths and utility code

### Statement

Use `strconv` for numeric/string conversions in performance-sensitive code.

### Rationale

`fmt` is more flexible but adds overhead.

### Bad

```go
s := fmt.Sprintf("%d", n)
```

### Good

```go
s := strconv.Itoa(n)
```

## RG-PRF-002 - Pre-allocate slices and maps when size is known

Type: recommendation
Enforcement: info
Taxonomy: performance
Skill_Primary: Perception
Difficulty_Base: Medium
Scope: allocations

### Statement

When expected size is known, initialize capacity up front.

### Rationale

Pre-allocation reduces growth reallocations.

### Bad

```go
items := []Item{}
for _, raw := range rawItems {
	items = append(items, parse(raw))
}
```

### Good

```go
items := make([]Item, 0, len(rawItems))
for _, raw := range rawItems {
	items = append(items, parse(raw))
}
```

### Security

## RG-SEC-001 - Use crypto/rand for secrets and tokens

Type: rule
Enforcement: block
Taxonomy: security
Skill_Primary: Half Light
Difficulty_Base: Challenging
Scope: security-sensitive code

### Statement

Never use `math/rand` for secret generation.

### Rationale

`math/rand` is predictable and unsafe for security material.

### Bad

```go
token := make([]byte, 32)
for i := range token {
	token[i] = byte(mathrand.Intn(256))
}
```

### Good

```go
token := make([]byte, 32)
if _, err := rand.Read(token); err != nil {
	return nil, fmt.Errorf("could not generate token: %w", err)
}
```

## RG-SEC-002 - Do not leak sensitive values in errors or logs

Type: rule
Enforcement: block
Taxonomy: security
Skill_Primary: Half Light
Difficulty_Base: Challenging
Scope: logs and user-facing errors

### Statement

Do not print secrets, tokens, or private payloads in logs and returned errors.

### Rationale

Secrets in telemetry create compliance and breach risk.

### Bad

```go
log.Printf("Failed to authenticate: token=%s err=%v", token, err)
return fmt.Errorf("could not authenticate user with token %s: %w", token, err)
```

### Good

```go
log.Printf("Failed to authenticate user: %v", err)
return fmt.Errorf("could not authenticate user: %w", err)
```

### Dependency and Wiring

## RG-DEP-001 - Use dependency injection for testability

Type: recommendation
Enforcement: info
Taxonomy: reliability
Skill_Primary: Volition
Difficulty_Base: Medium
Scope: constructors and service wiring

### Statement

Inject collaborators via constructors or options so production code and tests can swap implementations safely.

### Rationale

Dependency injection improves test seams and reduces hidden coupling.

### Bad

```go
type Service struct{}

func (s *Service) Run(ctx context.Context) error {
	client := &http.Client{Timeout: 5 * time.Second}
	_, err := client.Get("https://example.com")
	if err != nil {
		return fmt.Errorf("could not fetch upstream: %w", err)
	}

	return nil
}
```

### Good

```go
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type Service struct {
	client HTTPClient
}

func NewService(client HTTPClient) *Service {
	return &Service{client: client}
}
```

### Notes

- inject at construction boundaries, not deep inside business functions
- prefer small consumer-side interfaces to keep mocks/fakes simple

## RG-DEP-002 - Do not initialize external dependencies inside business functions

Type: recommendation
Enforcement: info
Taxonomy: reliability
Skill_Primary: Volition
Difficulty_Base: Medium
Scope: methods and function bodies

### Statement

Avoid constructing external dependencies (clients, stores, clocks, random sources) inside business functions. Use injected fields or method parameters instead.

### Rationale

Inline initialization hides side effects, makes tests brittle, and increases runtime coupling.

### Bad

```go
func ProcessOrder(ctx context.Context, orderID string) error {
	repo := postgres.NewRepository()
	mailer := smtp.NewClient()

	if err := repo.MarkProcessed(ctx, orderID); err != nil {
		return fmt.Errorf("could not mark order processed: %w", err)
	}

	if err := mailer.Send(orderID); err != nil {
		return fmt.Errorf("could not send order email: %w", err)
	}

	return nil
}
```

### Good

```go
type Service struct {
	repo   OrderRepository
	mailer Mailer
}

func (s *Service) ProcessOrder(ctx context.Context, orderID string) error {
	if err := s.repo.MarkProcessed(ctx, orderID); err != nil {
		return fmt.Errorf("could not mark order processed: %w", err)
	}

	if err := s.mailer.Send(orderID); err != nil {
		return fmt.Errorf("could not send order email: %w", err)
	}

	return nil
}
```

### Notes

- initialization should happen in wiring/constructor layers
- passing dependencies as parameters is acceptable for pure helpers or one-off orchestration paths

### General Recommendations

## RG-REC-001 - Prefer well-maintained third-party packages over reinvention

Type: recommendation
Enforcement: info
Taxonomy: readability
Skill_Primary: Composure
Difficulty_Base: Easy
Scope: architecture and implementation choices

### Statement

Prefer established, maintained packages when they clearly reduce risk and maintenance burden.

### Rationale

Reusing proven components reduces defects and implementation churn.

### Bad

```go
// custom hand-rolled env parser for common tags and defaults
func parseEnv(dst any) error { ... }
```

### Good

```go
// prefer maintained package behavior over bespoke parser
if err := env.Parse(&cfg); err != nil {
	return Config{}, fmt.Errorf("could not parse environment config: %w", err)
}
```

### Notes

Evaluate package health, maintenance cadence, and license compatibility.

## RG-REC-002 - Avoid reflection unless there is no simpler option

Type: recommendation
Enforcement: info
Taxonomy: readability
Skill_Primary: Composure
Difficulty_Base: Easy
Scope: runtime type handling

### Statement

Avoid reflection in normal business logic.

### Rationale

Reflection hides control flow and weakens type safety.

### Bad

```go
func IsZero(v any) bool {
	rv := reflect.ValueOf(v)
	return rv.IsZero()
}
```

### Good

```go
func IsEmptyString(v string) bool {
	return v == ""
}
```

## RG-REC-003 - Use generics only when they improve clarity and reuse

Type: recommendation
Enforcement: info
Taxonomy: readability
Skill_Primary: Composure
Difficulty_Base: Easy
Scope: generic APIs

### Statement

Use type parameters when they remove duplication without obscuring intent.

### Rationale

Unnecessary generics increase complexity and onboarding cost.

### Bad

```go
func First[T any](values []T) (T, bool) {
	var zero T
	if len(values) == 0 {
		return zero, false
	}

	return values[0], true
}
```

### Good

```go
func FirstString(values []string) (string, bool) {
	if len(values) == 0 {
		return "", false
	}

	return values[0], true
}
```

## RG-REC-004 - Prefer explicit type constraints when using generics

Type: recommendation
Enforcement: info
Taxonomy: readability
Skill_Primary: Interfacing
Difficulty_Base: Easy
Scope: generic APIs

### Statement

When using generics, prefer specific type constraints over `any` whenever behavior depends on available operations.

### Rationale

Explicit constraints document intent, improve compile-time safety, and make generic APIs easier to understand.

### Bad

```go
func Max[T any](a, b T) T {
	if a > b {
		return a
	}

	return b
}
```

### Good

```go
type Ordered interface {
	~int | ~int64 | ~float64 | ~string
}

func Max[T Ordered](a, b T) T {
	if a > b {
		return a
	}

	return b
}
```

### Notes

- use the narrowest practical constraint that matches required behavior
- keep constraints close to usage; extract reusable constraints only when they are shared and stable

## RG-REC-005 - Prefer `github.com/caarlos0/env/v11` for environment configuration

Type: recommendation
Enforcement: info
Taxonomy: readability
Skill_Primary: Interfacing
Difficulty_Base: Easy
Scope: configuration loading and env parsing

### Statement

Prefer `github.com/caarlos0/env/v11` for environment-driven configuration structs.
Use struct tags and defaults for non-critical settings to keep bootstrap code simple.

### Rationale

Centralized config parsing reduces boilerplate, keeps defaults close to config fields, and makes runtime behavior easier to inspect.

### Bad

```go
cfg := Config{}
if value := os.Getenv("APP_TIMEOUT"); value != "" {
	n, err := strconv.Atoi(value)
	if err != nil {
		return Config{}, fmt.Errorf("could not parse APP_TIMEOUT: %w", err)
	}
	cfg.TimeoutSeconds = n
} else {
	cfg.TimeoutSeconds = 30
}
```

### Good

```go
type Config struct {
	TimeoutSeconds int    `env:"APP_TIMEOUT" envDefault:"30"`
	LogLevel       string `env:"APP_LOG_LEVEL" envDefault:"info"`
}

func Load() (Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("could not parse environment config: %w", err)
	}

	return cfg, nil
}
```

### Notes

- keep defaults for non-critical values only; required and security-sensitive values should remain explicit
- document defaults in README/runtime docs when they impact behavior

## RG-REC-006 - Keep `main` thin and move orchestration out of `cmd/*`

Type: recommendation
Enforcement: info
Taxonomy: readability
Skill_Primary: Composure
Difficulty_Base: Easy
Scope: CLI/application entry points

### Statement

Keep `main` minimal. Handle only top-level error printing and exit code in `main`.
If `run()` grows beyond lightweight argument plumbing, move orchestration into an application package (for example, `internal/app`) and call `app.Run(...)` from `main`.

### Rationale

Thin entry points improve readability, testability, and consistency across commands. Moving orchestration out of `cmd/*` prevents oversized `main.go` files.

### Bad

```go
func main() {
	// parse flags
	// load config
	// run app flow
	// print errors
	// exit handling
}
```

### Good (small CLI)

```go
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

### Good (larger CLI)

```go
func main() {
	if err := app.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
```

### Notes

- keep exit code decisions centralized in `main`
- prefer returning errors from `run()` over calling `os.Exit` from deep functions
- avoid packing command parsing, wiring, transport setup, and business flow into `cmd/*/main.go`

## RG-REC-007 - Prefer named constants over repeated hardcoded literals

Type: recommendation
Enforcement: info
Taxonomy: readability
Skill_Primary: Composure
Difficulty_Base: Easy
Scope: command names, fixed paths, sentinel strings, and stable numeric thresholds

### Statement

Avoid scattering hardcoded literals throughout the code. Prefer named `const` declarations for stable values that are reused or semantically important.

### Rationale

Named constants make intent explicit, reduce typo risk, simplify refactors, and centralize defaults.

### Bad

```go
if os.Args[1] == "ingest" {
	...
}

db, err := chromem.NewPersistentDB("./data/vector_db", false)
```

### Good

```go
const (
	ingestStyleCommand   = "ingest"
	retrieveStyleCommand = "retrieve"
	reviewCommand        = "review"
	serveCommand         = "serve"
	statusCommand        = "status"
	vectorDBPath         = "./data/vector_db"
)
```

### Notes

- reserve constants for values that are stable and shared; keep one-off local literals local
- if a value may change by environment or runtime, prefer configuration instead of a constant

## RG-REC-008 - Avoid empty interfaces in business logic

Type: recommendation
Enforcement: info
Taxonomy: readability
Skill_Primary: Interfacing
Difficulty_Base: Easy
Scope: public APIs, business logic, and internal contracts

### Statement

Avoid `interface{}` / `any` as a catch-all type in normal business logic.
Prefer concrete types, small purpose-specific interfaces, or constrained generics.

### Rationale

Empty interfaces hide intent, force runtime type assertions, and make code harder to reason about and test.

### Bad

```go
func Process(value any) error {
	v, ok := value.(map[string]any)
	if !ok {
		return errors.New("could not process value")
	}

	_ = v
	return nil
}
```

### Good

```go
type Payload struct {
	ID   string
	Name string
}

func Process(value Payload) error {
	if value.ID == "" {
		return errors.New("could not process payload")
	}

	return nil
}
```

### Notes

- boundary layers (for example, JSON decoding glue) may temporarily use `map[string]any`, but convert to typed structs quickly
- this recommendation complements `RG-REC-002` (avoid reflection) and `RG-REC-004` (prefer constrained generics)

## Disco Mapping Table

Taxonomy to default Disco mechanics:

- `error_handling` -> primary `Logic`; alternates `Volition`, `Composure`, `Rhetoric`, `Reaction Speed`, `Encyclopedia`
- `security` -> primary `Half Light`; alternates `Interfacing`, `Perception`, `Authority`, `Esprit de Corps`, `Hand/Eye Coordination`
- `api_contract` -> primary `Interfacing`; alternates `Logic`, `Rhetoric`, `Composure`, `Visual Calculus`, `Esprit de Corps`
- `reliability` -> primary `Volition`; alternates `Composure`, `Endurance`, `Reaction Speed`, `Shivers`, `Empathy`
- `readability` -> primary `Composure`; alternates `Conceptualization`, `Drama`, `Logic`, `Suggestion`, `Savoir Faire`
- `performance` -> primary `Perception`; alternates `Visual Calculus`, `Interfacing`, `Physical Instrument`, `Electrochemistry`, `Pain Threshold`

Event-to-skill defaults (use when finding metadata does not provide a better voice fit):

- `hard_failure` -> `Authority`, `Half Light`, `Volition`
- `warning_failure` -> `Composure`, `Rhetoric`, `Logic`
- `soft_failure` -> `Suggestion`, `Empathy`, `Conceptualization`
- `filtered` -> `Perception`, `Visual Calculus`
- `timeout` -> `Reaction Speed`, `Endurance`
- `model_error` -> `Shivers`, `Inland Empire`
- `success` -> `Composure`, `Savoir Faire`, `Drama`
- `no_rule` -> `Encyclopedia`, `Inland Empire`, `Esprit de Corps`

Full skill roster reference (all 24 skills):

- `INTELLECT`: `Logic`, `Encyclopedia`, `Rhetoric`, `Drama`, `Conceptualization`, `Visual Calculus`
- `PSYCHE`: `Volition`, `Inland Empire`, `Empathy`, `Authority`, `Esprit de Corps`, `Suggestion`
- `PHYSIQUE`: `Endurance`, `Pain Threshold`, `Physical Instrument`, `Electrochemistry`, `Shivers`, `Half Light`
- `MOTORICS`: `Hand/Eye Coordination`, `Perception`, `Reaction Speed`, `Savoir Faire`, `Interfacing`, `Composure`

Guidance type to check semantics:

- `rule + block` -> failure check, can block CI
- `rule + warn` -> failure check, non-blocking warning
- `recommendation + info` -> advisory check only, never blocking

## Reviewer Prompt Contract

When reviewing diffs:

1. Emit violation findings only for items with `Type: rule`
2. For `Enforcement: block`, treat as high-priority findings
3. For `Type: recommendation`, emit suggestion checks only
4. If no matching item is found, emit no finding
5. Always include citation metadata from retrieved evidence

## References

- Effective Go
- Go Code Review Comments
- Google Go Style Guide
- Uber Go Style Guide
