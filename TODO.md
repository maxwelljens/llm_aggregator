# Architecture Deepening (post-v1.0.2 refactor round) — all slices DONE

Each slice: RED (write failing test for one behavior) → GREEN (minimal code) →
REFACTOR → `go build ./... && go vet ./... && go test ./... -count=1` →
`golangci-lint run ./...` → commit.

## 1. [x] TUI path drops the result and disables signals
- `runWithTUI` never writes output (`--tui --output-file foo.json` loses the
  summary); pipeline errors exit 0; `sh.Stop()` is called before the TUI runs
  and `Execute` uses `context.Background()`, so SIGINT cannot cancel the LLM
  call or exit 130.
- Fix: inject an exec seam (`func(ctx) (Result, error)`) into the model, wire
  the signal context through, route the finished `Result` through the shared
  output-writing path; exit codes match the non-TUI path.

## 2. [x] Run-path assembly and exit codes smeared across main.go
- 11 `os.Exit` sites in main.go, 2 in `internal/cli/args.go`; progress-logger
  selection copy-pasted between run paths.
- Fix: a `Run(...)` module that owns path selection and returns the exit code;
  `main()` performs the single `os.Exit`; no `os.Exit` outside main.

## 3. [x] LLM client constructed inside `Execute` — seam exists only for tests
- Production `Summariser` never injected; constructor takes 6 positional
  params, re-applies defaults, reads the API key from env directly (env-name
  literal duplicated in 3 places).
- Fix: assemble the LLM client in config/main and inject it; options struct
  constructor; single API-key resolution point.

## 4. [x] Fat 8-method `Progress` interface
- `SimpleLogger` no-ops 5 of 8 methods; aggregator/processor/llm only need
  `Logf/Warningf/Debugf` but accept the full interface.
- Fix: split `Logger` from stage reporting; adapters implement only what they
  use.

## 5. [x] Aggregator: ctx ignored, two HTTP stacks, errors swallowed
- `ParseFeed` uses gofeed's default client (no timeout, not injectable → not
  httptest-able); errgroup context discarded; `ctx` never reaches fetching;
  `feedErrors` write-only — total network failure surfaces as
  `ErrNoArticles`.
- Fix: pass `ctx` down; one injectable HTTP client for both fetch paths
  (`g.SetLimit(10)` replaces hand-rolled semaphore); surface feed errors.

## 6. [x] CLI→config is a stringly `map[string]any` bridge into a god Runtime
- ~20 viper string keys synced by hand across defaults / env bindings /
  `ToViperMap` / `ViperToRuntime`; `Runtime` has 21 fields mixing four
  concerns; `Runtime.Verbose` is dead; no constructor.
- Fix: typed config struct with one key-mapping site; `Runtime` constructor;
  delete dead fields.

## 7. [x] Dead code + duplicated constants sweep
- Dead: `progressMsg`, `StepMsg`, `TUIProgress.Run()`, `Runtime.Verbose`,
  stringly `"date"` sort at single call site (title/source branches).
- Duplicated: `"... [truncated]"` suffix (aggregator + processor),
  user-agent `"0.1.0"` vs `cli.Version`, env var name in 3 places,
  magic numbers (`5000`, `100`/`200` thresholds).

## 8. [x] Tests for untested packages
- `internal/progress`, `internal/tokeniser`, `internal/tui`, `internal/style`
  have zero tests. Aggregator has no httptest coverage of the fetch path
  (arrives with slice 5).
