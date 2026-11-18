# Book writing rules: goagent

This file overrides the project-book skill on every rule it states.

## Audience
- Knows: Go basics (structs, slices, errors, packages), the shell, what HTTP is.
- New to: talking to an LLM API, agent loops, tool calling, MCP, JSON-RPC, SSE.

## The user's rule: they write the code, the book reveals it (2026-11-18)
- **The reader writes every piece of code first.** No code is shown up
  front, not even the first time a concept appears. (This replaces the
  skill's "first appearance: show code in full" rule.)
- Each step: explain the goal and the needed pieces in plain words (the
  file, names, signatures, the std-lib functions to use and what they do),
  then a **"Your turn"** box, then the solution inside a collapsible block:

  ```markdown
  <details>
  <summary>Show the solution</summary>

  ```go
  ...
  ```

  </details>
  ```

  Blank lines around the code inside `<details>` so markdown renders.
- New syntax or APIs get more explanation in the "Your turn" box (the exact
  signature, a one-line example of the call shape not tied to our code).
- Terminal commands and their real output stay visible (not collapsed): they
  are the target the reader checks against.
- Solutions carry plain-word inline comments.

## Style
- Plain words, always. Describe what a thing does before naming it; name
  it once. No unexplained jargon, ever.
- One working piece of the tool per chapter. No theory-only chapters.
- Clean code, DRY, SOLID: followed in all book code, and named at the exact
  line where they apply, in plain words. Never a lecture.
- Testing: a few tests per chapter where they earn it (from Chapter 2).
  testify (`require` for setup, `assert` for checks). Faking the network
  (`httptest`) and child processes is the main testing lesson.
- Re-explain a concept briefly every time it comes back.
- Real captured output only. Never hand-typed terminal output. If output
  needs something we can't run, a clearly marked placeholder, filled later.
- Every chapter ends with: `make check` green, one git commit, "what you
  should now be able to answer", one-paragraph preview of the next.
- Terminal blocks: commands start with `$ `. Output has no prefix. Command
  and output in separate blocks.
- Later chapters show edits, not full reprints. Say which file and where.
- Always show `make check` / `make build`, not raw go commands, once the
  Makefile exists.

## Model (user rule, 2026-11-18)
- Every API call, in the book's code and in captures, uses Haiku
  (`claude-haiku-5-5`) or Sonnet (`claude-sonnet-5-5`). Never Opus or Fable.
  The book's default is Haiku 5.5.

## Docs
- Always check live docs before drafting (Context7 first, then official
  docs on the web). Never rely on memory for API shapes, versions, headers.
  Note in `.book/state.md` which source answered.

## Keeping the "why" visible
- "Where we are" map after each chapter's opening: the six stages from
  `src/00-outline.md`, marked ✓ done, ▶ this chapter, · later.
- Open on the gap, shown with real output. Then a why paragraph and
  "By the end of this chapter you can:" (2-4 bullets).
- End on the next gap.
- Walking skeleton in Chapter 6 (the agent loop with one tool). Later
  chapters replace one shortcut each and say which.

## Write it whole, then split it
- Reuse a function if it exists; otherwise write new logic inline first.
- Split out helpers only when the split teaches something (a SOLID idea,
  testability, or a second use for DRY).
- Test the outer function first; those tests survive the split.

## Process
- One chapter at a time. Write it, stop, the user tries it, then the next.
- Commit straight to `main` in this repo.

## State (any agent, any tool)
- Progress, decisions, and the next action live in `.book/state.md`. Read
  it first; update it at every checkpoint and before stopping.
- What's been explained already: `.book/concepts.md`.
- Real captured output: `.book/captures/chNN/`. Scratch copy of the code:
  `.book/scratch/` (gitignored).
- API key for captures: `.env` in the book root (gitignored). Load with
  `set -a; . ./.env; set +a`. Never print it.
- Workflow: the project-book skill (~/.claude/skills/project-book).
- The user's code lives in `goagent/` in this repo: read only.
