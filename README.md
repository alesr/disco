# Disco

## THE RADIO CRACKLES TO LIFE...

Disco is a daemon-first CLI that reviews Go diffs against a style guide.

It ingests one markdown guide at daemon startup, retrieves relevant rule evidence per hunk, and emits interactive checks with citations and severity. The output carry Disco Elysium flavor, but the point is still technical signal.

## Where are my keys... and my badge?

### Install

```bash
go install ./cmd/disco
```

or:

```bash
make install
```

### Prerequisites

- Go toolchain matching `go.mod` (`go 1.26.0`)
- Mistral API key if you don't want to use Kronk all the way
- [![This Playlist](assets/revachol-radio.jpg)](https://open.spotify.com/playlist/5M37hQWaY1WN7E99zrve8g?si=f0731ed9d31b4363)

### Envars

- `STYLE_GUIDE_DIR` (default: `./styleguides`) 
- `EMBEDDING_PROVIDER` (`kronk` or `mistral`, default: `kronk`)
- `GENERATION_PROVIDER` (`kronk` or `mistral`, default: `kronk`)
- `MISTRAL_API_KEY` (required when either provider is `mistral`)
- `MISTRAL_MODEL` (default: `mistral-small-latest`, generation)
- `MISTRAL_EMBED_MODEL` (default: `mistral-embed`, embeddings)
- `MISTRAL_BASE_URL` (default: `https://api.mistral.ai`)

#### What works for me

```
- EMBEDDING_PROVIDER=kronk 
- GENERATION_PROVIDER=mistral
- MISTRAL_API_KEY=... (required)
- MISTRAL_MODEL=mistral-small-latest (set as default in the code, but you can overide here)
- MISTRAL_BASE_URL=https://api.mistral.ai (default exists)
```

## STARTING YOUR SHIFT

1. Start the deamon to ingest the style.md

```
disco daemon start
```

2. To evaluate a git diff against the guidelines

```
disco review --diff sample-review.diff
```

### Example output

![sample_output](https://private-user-images.githubusercontent.com/4883432/566985838-b6a83c00-c862-4978-82b1-0a7a9f1c8a3e.png?jwt=eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJnaXRodWIuY29tIiwiYXVkIjoicmF3LmdpdGh1YnVzZXJjb250ZW50LmNvbSIsImtleSI6ImtleTUiLCJleHAiOjE3NzQwMjg1MDIsIm5iZiI6MTc3NDAyODIwMiwicGF0aCI6Ii80ODgzNDMyLzU2Njk4NTgzOC1iNmE4M2MwMC1jODYyLTQ5NzgtODJiMS0wYTdhOWYxYzhhM2UucG5nP1gtQW16LUFsZ29yaXRobT1BV1M0LUhNQUMtU0hBMjU2JlgtQW16LUNyZWRlbnRpYWw9QUtJQVZDT0RZTFNBNTNQUUs0WkElMkYyMDI2MDMyMCUyRnVzLWVhc3QtMSUyRnMzJTJGYXdzNF9yZXF1ZXN0JlgtQW16LURhdGU9MjAyNjAzMjBUMTczNjQyWiZYLUFtei1FeHBpcmVzPTMwMCZYLUFtei1TaWduYXR1cmU9OWU4MWNjYzUxNTVhZmU5YmM1YTgzYTIwYWFjNTMyM2JiNWM4MGRjMmU5NjAyOGJmNTk0MzE0ZTE0MTgyMzlkZiZYLUFtei1TaWduZWRIZWFkZXJzPWhvc3QifQ.w0Rr1DFzjIb2SWOM9qtv4Nxnwy1JPTiNuWw-FUcBDvs)

![sample_output](https://private-user-images.githubusercontent.com/4883432/566986849-26fb405f-52e5-474f-9bbc-dfd6335d1e66.png?jwt=eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJnaXRodWIuY29tIiwiYXVkIjoicmF3LmdpdGh1YnVzZXJjb250ZW50LmNvbSIsImtleSI6ImtleTUiLCJleHAiOjE3NzQwMjg2NDAsIm5iZiI6MTc3NDAyODM0MCwicGF0aCI6Ii80ODgzNDMyLzU2Njk4Njg0OS0yNmZiNDA1Zi01MmU1LTQ3NGYtOWJiYy1kZmQ2MzM1ZDFlNjYucG5nP1gtQW16LUFsZ29yaXRobT1BV1M0LUhNQUMtU0hBMjU2JlgtQW16LUNyZWRlbnRpYWw9QUtJQVZDT0RZTFNBNTNQUUs0WkElMkYyMDI2MDMyMCUyRnVzLWVhc3QtMSUyRnMzJTJGYXdzNF9yZXF1ZXN0JlgtQW16LURhdGU9MjAyNjAzMjBUMTczOTAwWiZYLUFtei1FeHBpcmVzPTMwMCZYLUFtei1TaWduYXR1cmU9ZmE2MmEzODRhZDFmYjExZjZiNmUzZDI4M2Q0MzY0ZDFmMWI0MmY5NjNkZWEzOTRiMDQwNmM5ZGE0NWUyNjdkNyZYLUFtei1TaWduZWRIZWFkZXJzPWhvc3QifQ.0lkO3ZRIIQf9-u4HDPsHL4yk-ohMEpBeowHwIXEsk6Q)

![sample_output](https://private-user-images.githubusercontent.com/4883432/567001924-8df2f7ea-9a93-4253-9310-4bbfa0616a20.png?jwt=eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJnaXRodWIuY29tIiwiYXVkIjoicmF3LmdpdGh1YnVzZXJjb250ZW50LmNvbSIsImtleSI6ImtleTUiLCJleHAiOjE3NzQwMzAzOTQsIm5iZiI6MTc3NDAzMDA5NCwicGF0aCI6Ii80ODgzNDMyLzU2NzAwMTkyNC04ZGYyZjdlYS05YTkzLTQyNTMtOTMxMC00YmJmYTA2MTZhMjAucG5nP1gtQW16LUFsZ29yaXRobT1BV1M0LUhNQUMtU0hBMjU2JlgtQW16LUNyZWRlbnRpYWw9QUtJQVZDT0RZTFNBNTNQUUs0WkElMkYyMDI2MDMyMCUyRnVzLWVhc3QtMSUyRnMzJTJGYXdzNF9yZXF1ZXN0JlgtQW16LURhdGU9MjAyNjAzMjBUMTgwODE0WiZYLUFtei1FeHBpcmVzPTMwMCZYLUFtei1TaWduYXR1cmU9ZDYyNGQ4YzNmNmE0MzE3ZDcwMjZkNDM1MmFkNjhhYWNjYzBlNTIwYTYxNzdiMDg5OWU2NmY4YzBkNDA2YTk0MiZYLUFtei1TaWduZWRIZWFkZXJzPWhvc3QifQ.yV0d03i8_XywAxMui8KnDuzu3ViGgR9rDu27UFJcdCQ)

## The Thought Cabinet

Disco expects exactly one markdown style guide in `STYLE_GUIDE_DIR` at daemon startup.

Current behavior:
- startup ingests style guide automatically
- retrieval is evidence-driven and attached to findings as citations

Recommended layout:

```text
styleguides/
  disco/
    style.md
```

I have it set with my personal styleguide, but you can add your own, just be attent to increment it with the necessary fields for the "roleplaying" ;]

### Rule Example:

```
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

```

If the guide is missing, duplicated, or malformed, startup should fail fast.

## Interfacing

### Daemon lifecycle

```bash
disco daemon install
disco daemon start
disco daemon status
disco daemon stop
disco daemon uninstall
```

For RAG. Keep it running in the background.

### Review commands

Review from file:

```bash
disco review --diff sample-review.diff
```

Review from git refs:

```bash
disco review --repo . --base main --head HEAD
```

Common flags:
- `--diff` path to unified diff file
- `--repo` repository path for git diff mode
- `--base` base ref for git diff mode
- `--head` head ref (default: `HEAD`)

## Rolling the Dice

Typical review flow:

1. hunk progress stream (`evaluating hunk X/Y ...`)
2. queued checks
3. interactive check-by-check output with:
   - narrative line(s)
   - technical message
   - citation
   - good example snippet

Severity mapping:

- `high` -> likely blocker
- `medium` -> should be fixed before merge
- `low` -> advisory but still worth fixing

Difficulty labels (`Challenging`, `Formidable`, `Legendary`, etc.). You set them directly in the styleguide.

But don't take it too seriously. You can use as policy, but I've added them more to match with the game style.

## Kim Kitsuragi’s Notes

- Keep diffs focused. Smaller hunks improve retrieval and judgment quality. Also time.
- Treat model output as guidance, not scripture. Verify against citation and code.
- Run tests after fixes. Style checks do not replace correctness checks.

Known limits:
- review v1 focuses on Go hunks (`*.go`)
- model output can still produce false positives or awkward phrasing. it depends a lot on the model you're using.
- retrieval quality depends on guide structure and metadata consistency

## Joining Precinct 41

Contributions are welcome. Really! There a lot of improvements that I would like to make but don't have time.

For example:

```
- sounds? would be cool, huh?
- more AI cloud providers?
- integration with opencode and stuff like that?
- Improved UI? if you have ideas
- tests!
- zsh completion
- gh workflow "ravechol CI" =D
- [ insert your cool idea here ]
```

Just open an issue and we'll chat about it.

### Local dev loop:

```bash
make fmt
make test
make vet
make vulncheck
```

You know the drill.

Then what I usually run during development:

```
disco main• ❱ make install
disco main• ❱ disco daemon stop && disco daemon uninstall && disco daemon install && disco daemon start 
disco main• 1.3s ❱ disco review --diff sample-review.diff
```

- if you changed only `internal/pkg/cli/*` (client output/UX), go install/run is enough
- if you changed daemon-side code `(internal/runtime/*, internal/review/*, internal/llm/*, internal/policy/rag/*, server transport)` or daemon `env vars`, restart daemon

## THE COALITION

[disco](https://github.com/coignard/disco) was an inspiration for this project. so cool.
And i wanted to have a reason to play with [kronk](https://github.com/ardanlabs/kronk) that allowed me to spin-up the LLM directly from the Go code.

## The Pale

### `No actionable checks. summary: model errors=N`

Generation failed or returned invalid payloads for some/all hunks.
- verify daemon env (`GENERATION_PROVIDER`, model, key)
- restart daemon after env changes
- inspect daemon logs

### Daemon says `running` but behavior looks stale

Launch service env may be stale.
- run `stop/uninstall/install/start` again

The deamon takes around 10s to warmup and ingest all the rules. Running `disco deamon status` during this time might return `running`, but well, we you can make it better.

### Style guide ingestion fails on startup

Common causes:
- wrong `STYLE_GUIDE_DIR`
- zero or multiple markdown guides
- malformed guide metadata

### Fish shell completion not available

Install completion:

```bash
disco completion fish > ~/.config/fish/completions/disco.fish
```

### Machiiines

`macOS/fish/ghostty`. well, I haven't tested it anywhere else.

## Moralintern Compliance

Apache License 2.0. See `LICENSE`.
