# How this book works

You will build **goagent**, a small coding agent that runs in your terminal,
the same kind of program as Claude Code. You build it in Go, from the ground
up, one working piece per chapter.

An AI model on its own only does one thing: it reads text and writes text.
It can't open a file, run a command, or remember what you said yesterday.
The program wrapped around the model does all of that: it sends the
request, keeps the conversation, runs the tools the model asks for, and
decides when to stop. That wrapper is called a **harness**. This book is
about writing one.

## Words used in this book

You don't need these yet. Each one is explained again in the chapter
where it first matters.

- **API**: a way for one program to ask another for something. Here,
  our Go program asks Anthropic's servers for a reply from Claude.
- **Token**: a piece of a word. The API counts (and charges) in tokens.
  A short English word is often one token.
- **Agent loop**: send the conversation to the model. If it asks to use a
  tool, run the tool, add the result, and send again. Stop when the model
  says it's done.
- **Tool**: a Go function the model can ask us to run, like "read this
  file". We describe it to the model with a name, a description, and the
  shape of its inputs.
- **MCP** (Model Context Protocol): a standard way for a harness to find
  and call tools that live in a *separate program*, called an MCP server.
- **JSON-RPC**: a simple message format MCP uses: "call method X with
  these inputs; here's an ID so you can match up the reply."
- **SSE** (server-sent events): a way for a server to send a reply in
  small pieces over one HTTP response, so you see words as they arrive.

## How each chapter works: you write, then check

You write all the code. The book never hands it to you first.

Each step tells you what to build and gives you the pieces you need: which
file, which names, which standard-library functions, and what they do. Then
comes a **Your turn** box. Write the code, run it, and compare your output
to the output shown in the book. When you're done (or stuck), open the
collapsed **Show the solution** block to compare your code with the book's.

Your code doesn't have to match the book's word for word. It has to
behave the same way. Later chapters build on the book's version, so if
yours is different, check that the names the next chapter uses still
exist.

Every terminal block in this book was run for real. Commands sit in their
own block with no `$` prompt, so you can copy and paste them as they are.
The block right after a command shows its output; don't paste that.

The check command is `make check`. It runs `go vet` (which looks for likely
bugs) and `go test`. Every chapter ends with it passing and one git commit.

Later chapters show *changes* to files you already have ("add this below
`main`"), not full reprints.

## What you need

- **Go 1.27** or later. Check with `go version`.
- **An Anthropic API key** from the Claude Console
  (console.anthropic.com), with a little credit on the account. The
  whole book costs a few dollars of API usage at most.
- `curl` and `jq`, to try the API by hand before we write Go for it.

## What the finished tool does

<!-- PLACEHOLDER: replace with a real captured session once Chapter 18 is
built. Never hand-typed. -->

> *A real session with the finished goagent goes here once the last
> chapter is built. It will show goagent reading files, asking before it
> runs a shell command, and calling a tool from an MCP server.*

## The six stages

The finished goagent's job, split into six stages. Every chapter opens with
this list, marked ✓ (done), ▶ (this chapter), or · (later).

1. **Talk to the model**: one HTTP request, one reply (Chapters 1–2)
2. **Hold a conversation**: history, errors and retries, streaming
   (Chapters 3–5)
3. **Loop**: the agent loop and a tool registry (Chapters 6–7)
4. **Tools**: files, the shell, permission, several tools at once
   (Chapters 8–11)
5. **Stay in control**: cancelling with Ctrl-C, staying inside the context
   size (Chapters 12–13)
6. **MCP**: tools from other programs, over stdio and HTTP
   (Chapters 14–18)

## The walking skeleton

In **Chapter 6**, goagent first works end to end: Claude asks to read a
file, goagent reads it, sends it back, and Claude answers. That version
takes shortcuts on purpose. Each later chapter removes one:

| Shortcut in Chapter 6 | Fixed in |
|---|---|
| One tool, written straight into the loop | Chapter 7 (tool registry) |
| Only `read_file`, no writing or shell | Chapters 8–9 |
| Runs anything the model asks, no questions | Chapter 10 |
| Runs tools one at a time | Chapter 11 |
| Ctrl-C kills the whole program | Chapter 12 |
| Conversation grows until it's too big | Chapter 13 |
| Only tools compiled into goagent | Chapters 14–17 |

## Chapters

**Part 1: Talking to the model**

1. **Hello, model.** Call the API with `curl`, then from Go with only
   `net/http`.
2. **Typed messages.** Structs for every kind of content block, a client
   behind an interface, and the first tests against a fake server.
   *Dependency inversion.*
3. **A chat prompt.** Keep the conversation in a slice so Claude remembers
   earlier turns. A system prompt.
4. **When the API says no.** Rate limits, overload, `retry-after`, retrying
   with a growing wait, timeouts.
5. **Streaming.** Read SSE by hand so words print as they arrive.

**Part 2: The loop**

6. **The agent loop** (walking skeleton). One hard-coded `read_file` tool,
   `tool_use` → `tool_result`, loop until done.
7. **A tool registry.** A `Tool` interface; errors go back to the model
   instead of crashing. *Open/closed.*

**Part 3: Real tools**

8. **File tools.** `read_file`, `list_dir`, `grep`, `edit_file`; never leave
   the working folder.
9. **The shell tool.** `bash` with a timeout and an output limit.
10. **Asking permission.** Approve writes and commands; an allow-list.
11. **Several tools at once.** Run them in parallel with goroutines, keep
    the order.

**Part 4: Staying in control**

12. **Ctrl-C and cancelling.** `context.Context` everywhere.
13. **Context size.** Counting tokens, `max_tokens`, shrinking old history,
    prompt caching.

**Part 5: MCP client**

14. **JSON-RPC by hand.** Start an MCP server as a child process; match
    replies to requests by ID.
15. **Handshake and tool list.** `initialize`, `tools/list`, and an adapter
    that makes MCP tools look like ours. *The Chapter 7 interface pays off.*
16. **Calling MCP tools.** `tools/call`, cancelling, a crashed server, several
    servers from a config file.
17. **MCP over HTTP.** The Streamable HTTP transport. *A `Transport`
    interface.*
18. **The real test.** goagent against real MCP servers, doing a real
    multi-step task.

## Dependencies

- **The Go standard library**, for everything in goagent itself. The HTTP
  client, the streaming parser, the JSON-RPC layer and the MCP client are
  all written by hand, because writing them is what this book teaches.
  We don't use Anthropic's Go SDK or an MCP SDK: they would hide exactly the
  parts we want to understand.
- **testify** (from Chapter 2), for tests that read clearly.

Versions this book was checked against: Go 1.27.4, Claude API version
`2026-10-21`, MCP specification `2026-07-28`.
