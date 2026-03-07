# Repository Guidelines

## Project Overview

`go-notes` is a knowledge repository for Go and backend engineering. It combines:

- Chinese technical notes and long-form explanations
- Runnable Go examples
- Unit tests, benchmarks, and fuzzing examples
- Diagrams and screenshots used as supporting evidence

This is not a single production service. Most work is topic-scoped and should preserve the repository's pattern of `explanation + runnable example + verification`.

## Primary Directories

- `gocore/`: Go language internals, primitives, concurrency, runtime, I/O, and standard library topics
- `goengineering/`: Go engineering practice such as testing, benchmarking, CI/CD, Docker, API design, logging, tracing, and dependency management
- `designpattern/`: design pattern writeups plus runnable Go examples and tests
- `middlewares/`: MySQL, Redis, and Kafka notes; mostly documentation and images
- `linux-perf/`: Linux performance tuning notes; mostly documentation and images
- `productivetools/`: productivity tooling, search workflow, AI tool usage, Git/Vim/IDE setup
- `shellscripts/`: shell-related notes and examples
- `softskill/`: writing and soft-skill notes
- `docs/plans/`: design and implementation plan artifacts created during agent work

## Repository Conventions

### Content structure

Common topic layout:

- `*.md`: concept explanation, best practices, tradeoffs, and conclusions
- `*.go`: runnable minimal examples
- `*_test.go`: unit tests and benchmark code
- `performance/`: controlled performance experiments and benchmark comparisons
- `trap/`: intentional anti-patterns, pitfalls, or counterexamples
- `images/`: diagrams, screenshots, benchmark output, or explanatory figures

When adding or editing a topic, preserve this structure instead of inventing a new one.

### Language and style

- Documentation is primarily in Chinese.
- Code should stay idiomatic Go and use English identifiers unless there is a strong reason not to.
- Favor concise explanation backed by runnable code or benchmark evidence.
- For technical claims about performance, memory, or concurrency behavior, prefer evidence over assertion.

### `trap/` directories

Files under `trap/` intentionally show bad patterns, failure modes, or edge cases.

- Do not "fix" them into production-style code unless the user explicitly asks.
- If you add a new trap example, make the surrounding documentation clear that it is a counterexample.

## Working Rules

### Prefer targeted changes

This repo is large and topic-based. Avoid broad refactors unless the user explicitly asks.

- Touch only the topic directory relevant to the task.
- If changing an example, also check the adjacent `.md` file for drift.
- If changing a `.md` explanation that references code behavior, verify the code or tests still support the claim.

### Preserve evidence-backed learning

Good changes usually include at least one of:

- runnable example code
- a unit test
- a benchmark
- a short verification note in the markdown

For performance topics, prefer benchmark evidence over prose-only claims.

### Keep docs and code aligned

If you modify:

- example behavior: update the corresponding `.md`
- benchmark code: update the surrounding explanation if conclusions changed
- images or screenshots: ensure the document still references the correct asset paths

## Validation

There is no single repository-wide validation command that is always appropriate. Use the narrowest command that matches the edited scope.

### Common commands

```bash
go test ./...
go test -race ./...
go test -cover ./...
```

### Topic-scoped examples

```bash
go test ./gocore/channel/performance
go test ./gocore/context/performance
go test ./gocore/concurrency/pattern ./gocore/concurrency/performance
go test ./gocore/interface/performance
go test ./gocore/struct/performance/set

go run ./designpattern
go test ./designpattern/...

go test ./goengineering/unit-test/...
go test ./goengineering/benchmark/...
go test ./goengineering/api-design/...

go test ./goengineering/fuzzingtest/byteparser \
  ./goengineering/fuzzingtest/multiparam \
  ./goengineering/fuzzingtest/roundtrip \
  ./goengineering/fuzzingtest/differential
```

### Benchmarks

Use standard Go benchmark commands when editing performance content:

```bash
go test -run='^$' -bench=. -benchmem
go test -run='^$' -bench=. -benchtime=3s -count=5 -benchmem
```

If benchmark conclusions matter, mention the exact command used.

## Editing Guidance

### When editing markdown

- Keep the tone instructional and concrete.
- Prefer clear section titles and short paragraphs.
- Use code snippets only when they materially improve understanding.
- Avoid turning notes into generic AI-generated tutorials; stay close to the repo's current level of specificity.

### When editing Go code

- Keep examples minimal and readable.
- Prefer table-driven tests where they help demonstrate boundaries clearly.
- Do not add unnecessary abstractions to teaching examples.
- Keep benchmark code focused on one comparison at a time.

### When adding new topics

Place them under the closest existing top-level area rather than creating a new top-level directory casually.

Examples:

- Go language/runtime topic -> `gocore/`
- Engineering workflow, testing, CI, tooling -> `goengineering/`
- MySQL/Redis/Kafka -> `middlewares/`
- Productivity workflow -> `productivetools/`

## Dependency Notes

`go.mod` includes several dependencies used by examples across the repository. Do not clean up dependencies casually unless the user explicitly asks for module maintenance.

If you remove code that truly eliminates a dependency, verify with:

```bash
go mod tidy
go mod verify
```

## Commit and Change Scope

- Use conventional commits if asked to commit.
- Prefer `docs:` for markdown-only changes.
- Prefer `test:`, `perf:`, `refactor:`, or `feat:` only when they clearly match the scope.

## Avoid

- Do not replace topic-specific notes with generic boilerplate.
- Do not "correct" intentional anti-patterns in `trap/` examples without instruction.
- Do not run heavyweight repo-wide commands if a narrow topic-scoped command is enough.
- Do not rewrite unrelated directories while working on one topic.
