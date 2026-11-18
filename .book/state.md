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
  - Model: `claude-opus-5-5` (current default per claude-api skill).
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
| 1 | Hello, model | written (awaiting user), 3 placeholders |
| 2 | Typed messages | planned |
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

- Chapter: 1, Hello, model
- Phase: awaiting user (and awaiting API credit)
- Done: scratch code final in `.book/scratch/goagent` (step2/, step3/,
  nostatus/ hold in-between states). Captures 01–17 in captures/ch01.
- **Blocked:** the API account has no credit (400 "credit balance is too
  low"). Three placeholders in `src/01-hello-model.md` need real output:
  (1) curl success JSON via `| jq`, (2) Step 3 raw JSON from goagent,
  (3) Try it: goagent answer + `[stop: ...]` line. grep PLACEHOLDER.
  Also the outline's "What the finished tool does" (fill at Ch18).
- Check when filling: does the reply contain a `thinking` block (Opus 5.5
  thinking is always on, display omitted by default)? If so, mention it in
  the curl walkthrough.
- **Next:** once credit is added, run the three captures, replace the
  placeholders, mdbook build, commit. Then when the user says Ch1 works,
  check their `goagent/` (read only) and start Ch2.

## User's code vs the book's

- (none yet; user hasn't written Ch1)

## Open questions

- (none)

## Log

- 2026-11-18 Claude Code (Opus 5.5): outline approved; book repo created;
  Ch1 code built and verified in scratch with Go 1.27.4; Ch1 drafted with
  real failure captures; success captures blocked on API credit.
