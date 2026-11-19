# Book state: Build Your Own Agent Harness in Go

Any agent picking this book up: read `AGENTS.md`, then this file, then
`concepts.md`. Update this file at every checkpoint (see the project-book
skill, references/book-state.md). If this file and the repo disagree, the
repo wins: fix this file.

## Book facts

- Title: Build Your Own Agent Harness in Go
- Teaches: building an AI agent harness in Go, by building `goagent`
- User's code: `~/goagent-book/goagent` (read only; the user writes it)
- Check command: `make check` (go vet + go test)
- Build the book: `mdbook build`; serve: `mdbook serve` (port 3000)
- API key for captures: `.env` (gitignored), `export ANTHROPIC_API_KEY=...`
- Decisions (user approved the outline and defaults, 2026-11-18):
  - Go 1.27 (1.27.4 current per go.dev/dl JSON, 2026-11-18). Why: user
    pointed out 1.27 is released; latest stable.
  - Provider: Claude only, raw `net/http`, no SDK. Why: the book teaches
    the wire format; a provider abstraction adds a chapter for little.
  - Model: `claude-haiku-5-5`; Sonnet 5.5 allowed. User rule: API calls
    use only Haiku or Sonnet, never Opus (2026-11-18).
  - API version header: `2026-10-21` (platform.claude.com migration guide).
  - MCP spec: `2026-07-28` (stateless; sessions removed from HTTP).
  - Tests: testify, a few per chapter where they earn it, from Ch2.
  - Deps: stdlib + testify only. No Anthropic SDK, no MCP SDK.
  - Walking skeleton: Ch6.
  - **User writes all code; solutions collapsed in `<details>`** (see
    AGENTS.md). Terminal output stays visible.
  - Docs: always Context7 / live docs, never memory (user, 2026-11-18).
  - Audience: knows Go basics + shell + HTTP (assumed; outline approved).
- Doc sources used for Ch1: platform.claude.com/docs/en/api/messages
  (response shape, stop_reason values), .../api/migration-2026-10-21
  (version header), go.dev/doc/go1.27, Context7 /websites/pkg_go_dev_net_http
  (Client.Do: no error on non-2xx).

## Progress

| Ch | Title | Status |
|---|---|---|
| 0 | How this book works (outline) | written |
| 1 | Hello, model | done |
| 2 | Typed messages | written (awaiting user) |
| 3 | A chat prompt | planned |
| 4 | When the API says no | planned |
| 5 | Streaming | planned |
| 6 | The agent loop (walking skeleton) | planned |
| 7 | A tool registry | planned |
| 8 | File tools | planned |
| 9 | The shell tool | planned |
| 10 | Asking permission | planned |
| 11 | Several tools at once | planned |
| 12 | Ctrl-C and cancelling | planned |
| 13 | Context size | planned |
| 14 | JSON-RPC by hand | planned |
| 15 | Handshake and tool list | planned |
| 16 | Calling MCP tools | planned |
| 17 | MCP over HTTP | planned |
| 18 | The real test | planned |

## Current work

- Chapter: 2, Typed messages
- Phase: awaiting user
- Design: package `claude` (types.go: Request, Message{Role, Content
  []ContentBlock}, ContentBlock{Type, Text}, TextBlock, Response.Text(),
  Usage, RoleUser/RoleAssistant; error.go: APIError + newAPIError;
  client.go: Client, NewClient(apiKey, baseURL, *http.Client),
  DefaultBaseURL, Send(ctx, Request)). main.go: messageSender interface,
  run(ctx, sender, prompt, stdout, stderr) error. Tests: claude/client_test.go
  (httptest: reply + APIError), main_test.go (fakeSender: prints, error).
- Scratch: goagent/ = final; ch01-final/, ch02-step3/, ch02-step4/ in-between.
- Captures ch02: 01 tidy, 02 step3 check, 03 broken version, 04 step4
  check, 05 final check, 06 build, 07 run, 08 wrong key.
- Sources: platform.claude.com messages content param (string = one text
  block), Context7 /stretchr/testify (v1.12.1 latest via proxy.golang.org),
  pkg.go.dev httptest.
- Chapter written, solutions verified identical to scratch files, mdbook builds.
- **Next:** when the user says Ch2 works, check their goagent/ (read
  only), then Ch3 (A chat prompt: history slice, RoleAssistant, system prompt).

## User's code vs the book's

- Module path `github.com/rijojohn85/goagent` (book: `goagent`); imports
  of `goagent/claude` become `github.com/rijojohn85/goagent/claude`.
- Go 1.24.5 installed, go.mod says `go 1.24.5` (book: 1.27.4). Ch2 code
  works on 1.24; told user to upgrade.
- Makefile build is `go build ./...`: with 2 packages it writes NO binary
  (verified 2026-11-19). Told user to use `-o goagent .`.
- `fail("API returned %s: ...", resp.StatusCode)`: %s with int → prints
  `%!s(int=401)`; vet misses it (format is concatenated). Gone in Ch2.
- Split `usage` into its own named type already (book does it in Ch2).
- `goagent/goagent` binary is tracked in git (no ignore rule); told user.

## Open questions

- (none)

## Log

- 2026-11-18 Claude Code (Opus 5.5): outline approved; book repo created;
  Ch1 code built and verified in scratch with Go 1.27.4; Ch1 drafted with
  real failure captures; success captures blocked on API credit.
- 2026-11-18 Claude Code: switched the book to claude-haiku-5-5 (user rule);
  re-captured 01, 02, 08, 09, 11, 16, 17; still no credit (400).
- 2026-11-19 Claude Code: key has credit again; captured real successes
  (18–22) and filled the three Ch1 placeholders.
- 2026-11-19 Claude Code: user finished Ch1 (commit 08ed2c9); Ch1 done.
  Ch2 code built in scratch, all captures real; Ch2 written.
