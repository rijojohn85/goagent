# Chapter 1: Hello, model

This is where goagent starts. By the end of the chapter it is a tiny Go
program that sends one question to Claude and prints the answer. There's no
loop and no tools yet, and it doesn't remember anything. It does the one
thing everything else in this book is built on: a single HTTP request to the
model, and a single reply.

New in this chapter: the Claude Messages API, `net/http` requests with
headers, `encoding/json` struct tags, reading environment variables, and
exit codes.

## Where we are

1. ▶ **Talk to the model** (this chapter and Chapter 2)
2. · Hold a conversation (Chapters 3–5)
3. · Loop (Chapters 6–7)
4. · Tools (Chapters 8–11)
5. · Stay in control (Chapters 12–13)
6. · MCP (Chapters 14–18)

There's nothing to run yet, so the gap is simply this: Go can't talk to
Claude. Every later feature (tools, the loop, MCP) ends up as more fields in
this same request, or more handling of this same reply. If you understand
one request and one reply completely, the rest of the book is adding pieces
to something you already know.

By the end of this chapter you can:

- call the Claude API by hand with `curl` and read every part of the reply
- send the same request from Go, using only the standard library
- tell a real failure (bad key, no credit) apart from a real answer
- pick the answer text out of the reply's JSON

## The real thing, by hand

Before writing Go, let's see exactly what goes over the wire. Then the Go
code has a clear target: do what this `curl` command does.

You need an API key from the Claude Console (console.anthropic.com). Put it
in an environment variable, a named value your shell passes to every
program it starts:

```sh
export ANTHROPIC_API_KEY=sk-ant-...your key...
```

Keep the key out of your code and out of git. Anyone who has it can spend
your credit.

Now send one question:

```sh
curl -s https://api.anthropic.com/v1/messages \
    -H "x-api-key: $ANTHROPIC_API_KEY" \
    -H "anthropic-version: 2026-10-21" \
    -H "content-type: application/json" \
    -d '{"model":"claude-haiku-5-5","max_tokens":1024,"messages":[{"role":"user","content":"What is the capital of France?"}]}' | jq
```

Here's what each part does:

- **`POST https://api.anthropic.com/v1/messages`**: the one address every
  request in this book goes to. `curl` uses POST because we pass `-d`
  (data).
- **`x-api-key`**: who you are. The API uses it to check you're allowed in
  and to bill you.
- **`anthropic-version: 2026-10-21`**: which version of the API's rules we
  were written against. When Anthropic changes something in a way that
  would break old programs, they release a new version date. Programs that
  still send the old date keep the old behavior. We send the newest one.
- **`content-type: application/json`**: tells the server the body is JSON.
- **The body** has three required fields:
  - `model`: which Claude model answers. `claude-haiku-5-5` is the
    current Haiku, the smallest and cheapest Claude model. It's plenty for
    this book and keeps your bill tiny. If you want stronger answers later,
    `claude-sonnet-5-5` (the current Sonnet) is the next step up.
  - `max_tokens`: the most the reply may use, in tokens. A token is a
    piece of a word; a short English word is often one token. If Claude
    hits this limit, the reply is cut off.
  - `messages`: the conversation so far. Here it's one message, from the
    `user` role.
- **`| jq`** pretty-prints the JSON reply so you can read it.

You should see something like this:

```json
{
  "model": "claude-haiku-5-5",
  "id": "msg_011ChBdvL6UTWsBBgt3UPuTF",
  "type": "message",
  "role": "assistant",
  "content": [
    {
      "type": "text",
      "text": "The capital of France is **Paris**."
    }
  ],
  "container": null,
  "stop_reason": "end_turn",
  "stop_sequence": null,
  "stop_details": null,
  "usage": {
    "input_tokens": 16,
    "cache_creation_input_tokens": 0,
    "cache_read_input_tokens": 0,
    "cache_creation": {
      "ephemeral_5m_input_tokens": 0,
      "ephemeral_1h_input_tokens": 0
    },
    "output_tokens": 14,
    "output_tokens_details": {
      "thinking_tokens": 0
    },
    "service_tier": "standard",
    "inference_geo": "global",
    "speed": "standard"
  },
  "diagnostics": null
}
```

The reply is JSON with a few important fields:

- **`content`** is a *list* of blocks, not one string. Each block has a
  `type`. A `text` block holds words for you to read. Later chapters meet
  other kinds: `tool_use` (Claude asking us to run a tool) and `thinking`
  (Claude's reasoning, when it thinks before answering). That's why
  it's a list: one reply can hold several kinds of thing.
- **`stop_reason`** says *why* Claude stopped. `end_turn` means "I'm done".
  `max_tokens` means "I ran out of room". In Chapter 6, `tool_use` will
  mean "please run a tool and tell me the result". The agent loop is built
  around this one field.
- **`usage`** counts tokens in (your request) and out (the reply). This is
  what you pay for. Here it's 16 in and 14 out. Its `output_tokens_details`
  says how many of the output tokens were spent thinking: 0 this time,
  because a question this easy needed no thinking, so there's no `thinking`
  block in `content` either.

The rest you can ignore for now. `id` names this one reply. `stop_sequence`,
`stop_details`, `container` and `diagnostics` are `null` because we didn't
use the features they belong to. Your `id` will be different, and Claude's
wording may be too. The *shape* is what matters.

Notice the `**Paris**`. Claude writes Markdown, a plain-text format where
`**word**` means bold. A web page would show it as bold; a terminal shows
the stars as they are. goagent prints it as it is, which is fine for now.

### When it goes wrong

It's worth seeing the failures before you write code for them. Run the
same `curl` command again with `unset ANTHROPIC_API_KEY` first, and add
`-i` (and drop `| jq`) so `curl` also prints the status line and headers.
With no key at all you get (headers trimmed):

```
HTTP/2 401
{"type":"error","error":{"type":"authentication_error","message":"x-api-key header is required"},"request_id":"req_011ChAe2wCWrksbbH5rkAnDc"}
```

`401` means "I don't know who you are". With a key that's wrong, the status is the same,
but the message is different:

```
{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"},"request_id":"req_011ChAe2xo1jDkfnexWwznXF"}
```

And with a real key on an account with no credit left, you get a `400`:

```
HTTP/2 400
{"type":"error","error":{"type":"invalid_request_error","message":"Your credit balance is too low to access the Anthropic API. Please go to Plans & Billing to upgrade or purchase credits."},"request_id":"req_011ChAe2zNkapJbzP4KFiUwA"}
```

Errors always have the same shape: `"type": "error"`, then an `error`
object with its own `type` and a `message` meant for people. That shape
matters in Chapter 4, where we decide which errors are worth retrying.

## The plan

goagent does exactly what `curl` did, in four steps:

1. **Set up the project**: a Go module and a Makefile.
2. **Build the request body**: turn the question into the JSON `curl` sent.
3. **Send it**: add the key and headers, POST it, and fail clearly on any
   error.
4. **Read the answer**: pull the text out of the reply's `content` list.

All of it goes in one file, `main.go`, inside one `main` function. That
function ends up long. That's on purpose: Chapter 2 splits it up, and you'll
see why each piece comes out.

## Step 1: Set up the project

Make the folder for your code and start a Go module in it. A module is Go's
name for a project: a folder with a `go.mod` file that names it.

```sh
mkdir goagent
cd goagent
go mod init goagent
```

```
go: creating new go.mod: module goagent
```

Open `go.mod`. The `go` line should say `1.27` or later. If it says
anything older, install Go 1.27 first.

Next, a `Makefile`, so every chapter checks your code with one command.

> **Your turn.** Create `Makefile` with two targets:
>
> - `check`: runs `go vet ./...` (which looks for likely bugs) and then
>   `go test ./...`.
> - `build`: runs `go build -o goagent .`, which makes a program called
>   `goagent` in this folder.
>
> Mark both as `.PHONY` (Make's way of saying "these are commands, not
> files"). Use a variable `GO := $(shell which go)` instead of plain `go`.
> It isn't needed yet, but it helps later if a target ever runs under
> `sudo`, which might not find your Go. Remember that Make needs a **tab**
> (not spaces) before each command line.

<details>
<summary>Show the solution</summary>

```make
# The Go found on your PATH.
GO := $(shell which go)

.PHONY: check build

# check: what every chapter ends with. vet looks for likely bugs,
# test runs the tests (none yet in Chapter 1).
check:
	$(GO) vet ./...
	$(GO) test ./...

# build: make the goagent program in this folder.
build:
	$(GO) build -o goagent .
```

</details>

You can't run `make check` yet, because there's no Go code. That's the
next step.

## Step 2: Build the request body

The body `curl` sent was this JSON:

```json
{"model":"claude-haiku-5-5","max_tokens":1024,"messages":[{"role":"user","content":"What is the capital of France?"}]}
```

We could build that string by hand, but one quote in the user's question
would break it. Instead we describe the shape as Go structs and let
`encoding/json` write the JSON.

Here's what you need:

- **Struct tags.** `encoding/json` uses a field's Go name as its JSON name,
  so `MaxTokens` would come out as `"MaxTokens"`. The API wants
  `"max_tokens"`. A tag after the field sets the JSON name:

  ```go
  Name string `json:"full_name"`   // shape only: field Name, JSON "full_name"
  ```

  The field must start with a capital letter. `encoding/json` can only see
  exported (capitalized) fields.
- **`json.Marshal(v any) ([]byte, error)`** turns a Go value into JSON
  bytes.
- **`os.Args`** is the list of command-line words. `os.Args[0]` is the
  program's own name, so the question is `os.Args[1:]`.
  `strings.Join(words, " ")` glues them back into one string.
- **Standard error and exit codes.** Normal output goes to *standard output*
  (`fmt.Println`). Error messages go to *standard error*
  (`fmt.Fprintf(os.Stderr, ...)`), so they don't mix with real output when
  you pipe goagent into another program. `os.Exit(1)` stops the program
  with exit code 1. The shell reads any code other than 0 as "this failed".

> **Your turn.** Write `main.go` in package `main`:
>
> 1. Two constants: `model` = `"claude-haiku-5-5"` and `maxTokens` = `1024`.
>    Naming them means no unexplained "magic" values buried in the code.
> 2. A struct `message` with `Role` and `Content` strings (JSON names
>    `role` and `content`).
> 3. A struct `messageRequest` with `Model` (string), `MaxTokens` (int) and
>    `Messages` (a slice of `message`), tagged `model`, `max_tokens`,
>    `messages`.
> 4. A helper `fail(format string, args ...any)` that prints
>    `goagent: ` + the formatted message + a newline to standard error, then
>    exits with code 1. `args ...any` means "any number of values of any
>    type", like Python's `*args`. Pass them on to `Fprintf` as `args...`.
> 5. In `main`: join `os.Args[1:]` into `prompt`. If it's empty, call
>    `fail` with ``usage: goagent "your question"``. Build a
>    `messageRequest` with one `user` message, `json.Marshal` it (on error,
>    `fail("encoding request: %v", err)`), and print the JSON with
>    `fmt.Println(string(body))`.
>
> We'll send it in Step 3. For now, printing it lets you compare it with
> the `curl` body.

<details>
<summary>Show the solution</summary>

```go
// Command goagent sends one question to Claude and prints the answer.
package main

import (
	"encoding/json" // turns Go structs into JSON and back
	"fmt"
	"os" // environment variables, command-line args, exit codes
	"strings"
)

// Names for values that would otherwise be "magic" strings in the code.
const (
	model     = "claude-haiku-5-5"
	maxTokens = 1024 // the most tokens (word pieces) the reply may use
)

// messageRequest is the JSON body we send. The `json:"..."` tags
// give each field its name in JSON, since the API uses snake_case.
type messageRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []message `json:"messages"`
}

// message is one turn of the conversation: who said it, and what.
type message struct {
	Role    string `json:"role"` // "user" or "assistant"
	Content string `json:"content"`
}

// fail prints a message to standard error and stops the program
// with exit code 1, which tells the shell "this went wrong".
func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "goagent: "+format+"\n", args...)
	os.Exit(1)
}

func main() {
	// Everything after the program name is the question.
	prompt := strings.Join(os.Args[1:], " ")
	if prompt == "" {
		fail(`usage: goagent "your question"`)
	}

	// Build the request body as a Go value, then turn it into JSON.
	body, err := json.Marshal(messageRequest{
		Model:     model,
		MaxTokens: maxTokens,
		Messages:  []message{{Role: "user", Content: prompt}},
	})
	if err != nil {
		fail("encoding request: %v", err)
	}

	// For now, just show what we would send.
	fmt.Println(string(body))
}
```

`fail` is used twice already, and Step 3 adds four more uses. Without it,
every error would repeat the same three lines: print to standard error, add
the prefix, exit. If we later want a different prefix, we change one
place. That's the DRY principle ("don't repeat yourself"): each piece of
knowledge, here "how goagent reports a fatal error", lives in one spot.

</details>

Check it, build it, and run it:

```sh
make check
```

```
/usr/local/go/bin/go vet ./...
/usr/local/go/bin/go test ./...
?   	goagent	[no test files]
```

(Your Go path may be different.) `[no test files]` is fine: tests start in
Chapter 2.

```sh
make build
./goagent "What is the capital of France?"
```

```
{"model":"claude-haiku-5-5","max_tokens":1024,"messages":[{"role":"user","content":"What is the capital of France?"}]}
```

That's exactly the body `curl` sent. Now try it with no question:

```sh
./goagent
```

```
goagent: usage: goagent "your question"
```

**Always quote the question.** If your shell is zsh, an unquoted `?` is a
filename pattern. zsh looks for files matching `France?`, finds none, and
refuses to run the command at all:

```sh
./goagent What is the capital of France?
```

```
zsh:1: no matches found: France?
```

goagent never even started. Quoting the question avoids this in every
shell.

## Step 3: Send it

Now send the body, the way `curl` did. You need:

- **`os.Getenv(name string) string`** reads an environment variable. It
  returns `""` if it isn't set.
- **`http.NewRequestWithContext(ctx, method, url, body)`** builds a
  request without sending it. For now, pass `context.Background()` as
  the context. A context is how Go cancels work that's in progress;
  `Background()` means "never cancelled". Chapter 12 makes Ctrl-C use it.
  The body must be an `io.Reader` (something you can read bytes from);
  `bytes.NewReader(body)` wraps our `[]byte` as one. Use
  `http.MethodPost` rather than the string `"POST"`.
- **`req.Header.Set(name, value)`** sets one header.
- **`http.Client{Timeout: ...}`** is a client with a time limit on the
  whole request. The ready-made `http.DefaultClient` has **no** timeout,
  so a stuck connection would hang goagent forever. `client.Do(req)` sends
  the request and returns `(*http.Response, error)`.
- **`defer resp.Body.Close()`**. `defer` runs a call when the function
  returns. You must close every reply body, or Go can't reuse the
  connection.
- **`io.ReadAll(resp.Body)`** reads the whole body into a `[]byte`.

One trap, straight from the `net/http` docs: **`client.Do` does not return
an error for a 401 or a 500.** Its error means "I couldn't talk to the
server at all" (no network, a timeout). If the server answers "your key is
wrong", that still counts as a successful HTTP exchange. You must check
`resp.StatusCode` yourself.

> **Your turn.** Edit `main.go`:
>
> 1. Add three constants: `apiURL` = `"https://api.anthropic.com/v1/messages"`,
>    `apiVersion` = `"2026-10-21"`, and `apiKeyEnvVar` = `"ANTHROPIC_API_KEY"`.
> 2. At the **top** of `main`, read the key. If it's empty,
>    `fail("%s is not set", apiKeyEnvVar)`. Checking first means we never
>    do any work we'd have to throw away.
> 3. Replace the final `fmt.Println(string(body))` with: build the POST
>    request, set the three headers (`x-api-key`, `anthropic-version`,
>    `content-type`), send it with a client that has a 2-minute timeout,
>    close the body when `main` returns, and read the whole body into `raw`.
>    `fail` on each error with a short note of what we were doing
>    (`"building request: %v"`, `"sending request: %v"`,
>    `"reading reply: %v"`).
> 4. If `resp.StatusCode` isn't `http.StatusOK` (200),
>    `fail("API returned %s: %s", resp.Status, raw)`. `resp.Status` is the
>    text form, like `401 Unauthorized`.
> 5. Otherwise, print `raw` as a string.
>
> `go vet` (in `make check`) will tell you which imports to add.

<details>
<summary>Show the solution</summary>

The imports and constants at the top of the file now read:

```go
import (
	"bytes"         // bytes.NewReader turns our []byte into something http can read from
	"context"       // context.Background() is the "no deadline, no cancel" starting context
	"encoding/json" // turns Go structs into JSON and back
	"fmt"
	"io"       // io.ReadAll reads a whole reply body
	"net/http" // the HTTP client
	"os"       // environment variables, command-line args, exit codes
	"strings"
	"time"
)

// Names for values that would otherwise be "magic" strings in the code.
const (
	apiURL       = "https://api.anthropic.com/v1/messages"
	apiVersion   = "2026-10-21" // the API version we were written against
	apiKeyEnvVar = "ANTHROPIC_API_KEY"
	model        = "claude-haiku-5-5"
	maxTokens    = 1024 // the most tokens (word pieces) the reply may use
)
```

At the top of `main`, before reading the question:

```go
	// The key comes from the environment, never from the source code.
	apiKey := os.Getenv(apiKeyEnvVar)
	if apiKey == "" {
		fail("%s is not set", apiKeyEnvVar)
	}
```

And in place of the last `fmt.Println`:

```go
	// A request is method + URL + headers + body.
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		fail("building request: %v", err)
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", apiVersion)
	req.Header.Set("content-type", "application/json")

	// Our own client, so a stuck network can't hang us forever.
	client := &http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		fail("sending request: %v", err)
	}
	defer resp.Body.Close() // runs when main returns, and frees the connection

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		fail("reading reply: %v", err)
	}
	// Any status other than 200 OK is an error. The body says why.
	if resp.StatusCode != http.StatusOK {
		fail("API returned %s: %s", resp.Status, raw)
	}

	// For now, print the raw JSON reply.
	fmt.Println(string(raw))
```

</details>

`make check` should still pass. Build it and try the failures first, since
they cost nothing. With no key:

```sh
make build
unset ANTHROPIC_API_KEY
./goagent "What is the capital of France?"
```

```
goagent: ANTHROPIC_API_KEY is not set
```

With a wrong key (setting a variable in front of a command sets it for that
one command only):

```sh
ANTHROPIC_API_KEY=sk-ant-not-a-real-key ./goagent "What is the capital of France?"
```

```
goagent: API returned 401 Unauthorized: {"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"},"request_id":"req_011ChAe38zgAH364fqLissKV"}
```

That's the same error `curl` got, now coming through Go. Set your real key
again and ask for real:

```sh
export ANTHROPIC_API_KEY=sk-ant-...your key...
./goagent "What is the capital of France?"
```

```
{"model":"claude-haiku-5-5","id":"msg_011ChBdvx6AzEQhWFLvgXX78","type":"message","role":"assistant","content":[{"type":"text","text":"The capital of France is **Paris**."}],"container":null,"stop_reason":"end_turn","stop_sequence":null,"stop_details":null,"usage":{"input_tokens":16,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"cache_creation":{"ephemeral_5m_input_tokens":0,"ephemeral_1h_input_tokens":0},"output_tokens":14,"output_tokens_details":{"thinking_tokens":0},"service_tier":"standard","inference_geo":"global","speed":"standard"},"diagnostics":null}
```

It's the same JSON `curl` printed, just not pretty-printed. It works, but
nobody wants to read that. Next we pull out the part you'd actually read.

## Step 4: Read the answer

Reading JSON into Go is the same idea in reverse:

- **`json.Unmarshal(data []byte, v any) error`** fills in the Go value
  that `v` points to. Pass a pointer (`&reply`) so it can change your
  variable.
- **Unknown fields are skipped.** If the JSON has a field your struct
  doesn't, `Unmarshal` ignores it. So your struct only needs the fields you
  use, and the API can add new fields without breaking goagent.
- **A struct can sit inside a struct**, with no name of its own, written
  in place:

  ```go
  Size struct {               // shape only
  	Width int `json:"w"`
  } `json:"size"`
  ```

> **Your turn.** In `main.go`:
>
> 1. Add a struct `contentBlock` with `Type` and `Text` strings (JSON
>    `type`, `text`).
> 2. Add a struct `messageResponse` with `Content` (a slice of
>    `contentBlock`), `StopReason` (JSON `stop_reason`), and `Usage`, an
>    in-place struct with `InputTokens` and `OutputTokens` ints (JSON
>    `input_tokens`, `output_tokens`).
> 3. Replace the final raw print: `Unmarshal` `raw` into a
>    `messageResponse` (`fail("decoding reply: %v", err)` on error). Then
>    print the `Text` of every block whose `Type` is `"text"`, one per line.
>    Skip the others. Claude can also send `thinking` blocks, and their
>    text isn't meant for you.
> 4. Last, print one line to **standard error**:
>    `[stop: end_turn, tokens in: N, out: M]`. It goes to standard error
>    because it's information *about* the answer, not the answer itself.

<details>
<summary>Show the solution</summary>

Put the two new types below `message`:

```go
// messageResponse holds only the parts of the reply we use.
// encoding/json skips any JSON field that has no matching Go field.
type messageResponse struct {
	Content    []contentBlock `json:"content"`
	StopReason string         `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// contentBlock is one piece of the reply. Its Type says what kind it is.
// For now we only read "text" blocks.
type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
```

And in place of the raw print at the end of `main`:

```go
	// Turn the JSON reply into our Go struct.
	var reply messageResponse
	if err := json.Unmarshal(raw, &reply); err != nil {
		fail("decoding reply: %v", err)
	}

	// Print only the text blocks. The reply may hold other kinds too.
	for _, block := range reply.Content {
		if block.Type == "text" {
			fmt.Println(block.Text)
		}
	}
	fmt.Fprintf(os.Stderr, "[stop: %s, tokens in: %d, out: %d]\n",
		reply.StopReason, reply.Usage.InputTokens, reply.Usage.OutputTokens)
```

`main.go` now reads, top to bottom: imports, constants, `messageRequest`,
`message`, `messageResponse`, `contentBlock`, `fail`, then `main`.

</details>

```sh
make check
```

```
/usr/local/go/bin/go vet ./...
/usr/local/go/bin/go test ./...
?   	goagent	[no test files]
```

## Try it

```sh
make build
./goagent "What is the capital of France?"
```

```
The capital of France is **Paris**.
[stop: end_turn, tokens in: 16, out: 14]
```

The answer goes to standard output. The line in brackets goes to standard
error. `end_turn` means Claude finished on its own, and the tokens match
what `curl` showed: 16 in, 14 out.

You can see the split by throwing standard error away. `2>` redirects
standard error (stream number 2), and `/dev/null` is a file that swallows
whatever is written to it:

```sh
./goagent "What is the capital of France?" 2>/dev/null
```

```
The capital of France is **Paris**.
```

Only the answer is left.

A longer question uses more tokens:

```sh
./goagent "Explain in two sentences what an HTTP status code is."
```

```
An HTTP status code is a three-digit number that a server returns in response to a client's request, indicating whether the request succeeded, failed, or requires further action. The first digit defines the general category of the response, such as 2xx for success, 4xx for client errors, and 5xx for server errors.
[stop: end_turn, tokens in: 26, out: 91]
```

### Why the status check matters

Delete the `if resp.StatusCode != http.StatusOK` block for a moment,
rebuild, and use a wrong key:

```sh
ANTHROPIC_API_KEY=sk-ant-not-a-real-key ./goagent "What is the capital of France?"
```

```
[stop: , tokens in: 0, out: 0]
```

No error, and the exit code is 0 ("success"). The error JSON had no
`content` field, `Unmarshal` skipped what it didn't recognise, and we
printed an empty answer. This is the trap from Step 3: `client.Do` was
happy, because the server *did* answer. Only the status code says the
answer was "no". **Put the check back** before you go on.

### No credit

If your account runs out of credit, the API answers `400`, and goagent says
so plainly:

```sh
./goagent "What is the capital of France?"
```

```
goagent: API returned 400 Bad Request: {"type":"error","error":{"type":"invalid_request_error","message":"Your credit balance is too low to access the Anthropic API. Please go to Plans & Billing to upgrade or purchase credits."},"request_id":"req_011ChAe3G84CvCT2zHuEyutF"}
```

## What this chapter doesn't do

- **No memory.** Every run is a fresh conversation of one message.
  Chapter 3 keeps a history.
- **No retries.** If the API is busy (`429`, `529`), goagent just fails.
  Chapter 4 waits and tries again.
- **The answer arrives all at once.** For a long answer you stare at
  nothing until it's done. Chapter 5 streams it.
- **No tests.** Everything is in `main`, which reaches the real network,
  so there's nothing we can test without spending money. Chapter 2 fixes
  that.
- **`fail` skips deferred calls.** `os.Exit` stops the program at once, so
  `defer resp.Body.Close()` never runs on that path. That's harmless here,
  because the program is ending anyway, but it's one reason Chapter 2 moves
  to returning errors instead of exiting.

## Commit

Create a `.gitignore` with the line `goagent`, so the built program isn't
committed. Then:

```sh
make check
git init
git add -A
git commit -m "Chapter 1: one request to Claude, print the answer"
```

## What you should now be able to answer

Try to answer each one in your own words first. Then open the answer
to check.

**1. What three things must every Messages API request body have?**

<details>
<summary>Answer</summary>

`model` (which Claude answers), `max_tokens` (the most tokens the reply
may use), and `messages` (the conversation so far, each with a `role` and
`content`).

</details>

**2. Why is the reply's `content` a list and not a single string?**

<details>
<summary>Answer</summary>

One reply can hold several kinds of block, each with its own `type`: text
for you to read, `thinking`, and later `tool_use` requests. A list lets
them sit side by side in order. That's why goagent prints only the blocks
whose type is `text`.

</details>

**3. `client.Do` returned no error, yet the request failed. How?**

<details>
<summary>Answer</summary>

`client.Do` only returns an error when it couldn't complete the HTTP
exchange at all, such as no network or a timeout. A `401` or `400` is a
complete exchange, where the server answered "no". You have to check
`resp.StatusCode` yourself. Without that check, goagent printed an empty
answer and exited with 0.

</details>

**4. Why does goagent use its own `http.Client` with a `Timeout`?**

<details>
<summary>Answer</summary>

The ready-made `http.DefaultClient` has no time limit. If the connection
gets stuck, goagent would wait forever. Our client gives up after two
minutes and reports an error.

</details>

**5. Why do errors and the token line go to standard error?**

<details>
<summary>Answer</summary>

Standard output is for the real result, the answer text. Keeping errors
and extra information on standard error means that if you pipe goagent's
output into a file or another program, only the answer goes through. The
exit code (1 on failure) also lets scripts tell success from failure.

</details>

**6. What does `stop_reason` tell you, and why will it matter later?**

<details>
<summary>Answer</summary>

It says why Claude stopped writing: `end_turn` (finished) or
`max_tokens` (ran out of room). In Chapter 6 it will also say `tool_use`,
meaning "run a tool and send me the result". The agent loop decides what
to do next by reading this field.

</details>

## Next chapter

goagent works, but every line of it lives in `main`, and `main` talks to
the real API. To test anything, you'd need a key, credit, and a network,
and each test run would cost money. In Chapter 2 we split the work into
pieces with one job each: a `Client` that sends requests, and types for
every kind of content block. The client takes the thing that does the
HTTP sending as a *parameter*, so a test can hand it a fake server that
answers instantly and for free. That idea is called dependency
inversion, and it's the testing lesson the rest of the book is built on.
