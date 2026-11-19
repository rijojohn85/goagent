# Chapter 2: Typed messages

goagent works, but all of it lives in `main`, and `main` talks to the real
API. In this chapter we split it into pieces that each do one job, and we
write the book's first tests. By the end, `make check` runs four tests in
a few milliseconds. They need no API key, no network, and no money.

goagent's behaviour doesn't change much. It asks one question and prints
one answer, as before. What changes is the shape of the code inside, and
that shape is what every later chapter builds on.

New in this chapter: Go packages, methods, your own error type, wrapping
errors with `%w`, interfaces, `httptest` (a fake HTTP server for tests),
and testify (`require` and `assert`).

## Where we are

1. ▶ **Talk to the model** (Chapter 1 and this chapter)
2. · Hold a conversation (Chapters 3–5)
3. · Loop (Chapters 6–7)
4. · Tools (Chapters 8–11)
5. · Stay in control (Chapters 12–13)
6. · MCP (Chapters 14–18)

Here's the gap. This is `make check` at the end of Chapter 1:

```sh
make check
```

```
/usr/local/go/bin/go vet ./...
/usr/local/go/bin/go test ./...
?   	goagent	[no test files]
```

There are no tests, and there's no easy way to add one. The only code to
test is `main`. It reads your real key, calls the real API, and calls
`os.Exit` when anything goes wrong, which would stop the test run itself.
A test of `main` would cost money on every run, fail whenever your Wi-Fi
drops, and give a different answer each time.

This matters more with every chapter. By Chapter 6 goagent loops, calling
Claude again and again and running tools in between. That logic is easy to
get wrong, and you'll want tests that run it a hundred times without
paying for it. The pieces we make now let tests swap the real API for a
fake one.

By the end of this chapter you can:

- move code into its own Go package and use it from `main`
- describe the API's messages and replies as Go types, with content as a
  list of blocks in both directions
- test HTTP code against a fake server that runs inside the test
- make a function depend on a small interface, so tests can hand it a fake

## The plan

Three pieces, each with one job:

```
main          reads the key and the question, builds the real pieces,
  │           prints "goagent: ..." and exits 1 on error
  ▼
run           asks one question, prints the answer
  │           needs only "something that can Send"
  ▼
claude.Client turns a Request into an HTTP call, and the reply
              (or the error) back into Go values
```

Each piece is tested in its own way:

- **`claude.Client`** is tested against a fake HTTP server that runs inside
  the test. That checks the real JSON and the real headers, with no
  network.
- **`run`** is tested with a fake sender, a tiny Go value that returns a
  ready-made reply. No HTTP at all.
- **`main`** keeps only the work that can't be tested cheaply: reading the
  environment and exiting. It's short enough to check by eye.

The steps:

1. A `claude` package with types for requests and replies.
2. An error type for the API's "no" answers.
3. The `Client` that sends a request.
4. Tests for the client, against a fake server.
5. `run`, the interface it needs, and tests for it.

## Step 1: The `claude` package and its types

A **package** is a folder of `.go` files that all start with the same
`package` line. So far everything has been in package `main`, the one Go
turns into a program. Now we add a second package in a folder named
`claude`. Code in the `claude` package only deals with the API. It doesn't
know there's a command line, a terminal, or a goagent.

A package is used through its **import path**: your module's name (the
`module` line in `go.mod`), then a slash, then the folder. The book's
module is called `goagent`, so the path is `goagent/claude`. If your
`go.mod` says something else, like `github.com/you/goagent`, then your
path is `github.com/you/goagent/claude`. Use yours wherever the book
writes `goagent/claude`.

Other packages can only see names that start with a capital letter. That's
the same rule `encoding/json` used for struct fields in Chapter 1. So
`claude.Request` can be used from `main`, but a lower-case
`claude.newAPIError` can't. That lets a package keep its helpers private.

### Content is a list of blocks, in both directions

In Chapter 1 we sent `content` as a plain string:

```json
{"role":"user","content":"What is the capital of France?"}
```

The API also accepts a longer form, a list of content blocks, which is
exactly the shape Claude's replies use:

```json
{"role":"user","content":[{"type":"text","text":"What is the capital of France?"}]}
```

The API reference calls the string a shorthand for a list holding one text
block. The two mean the same thing. We switch to the list form so that one
Go type, `Message`, describes both what we send and what Claude sends
back. In Chapter 3 that pays off: Claude's reply goes straight into the
conversation history without being changed. In Chapter 6 our tool results
go back as blocks too.

You need two new pieces of Go:

- **`omitempty`** in a JSON tag leaves the field out of the JSON when it
  holds its zero value (`""`, `0`, `nil`). A `ContentBlock` will have one
  field per kind of block. A text block shouldn't carry empty fields that
  belong to other kinds.

  ```go
  Nick string `json:"nick,omitempty"`   // shape only: no "nick" key when Nick == ""
  ```

- **Methods.** A method is a function attached to a type. The part in
  brackets before the name, the *receiver*, is the value it works on:

  ```go
  func (p *Point) Len() float64 { ... }   // shape only: call it as p.Len()
  ```

  A `*Point` receiver means the method gets a pointer to the value rather
  than a copy. Using pointer receivers everywhere for a type keeps things
  simple.

> **Your turn.** Make a folder `claude` and in it a file `types.go`,
> starting with `package claude`. Move the types out of `main.go` and
> rename them with capital letters so `main` can use them:
>
> 1. Two constants, `RoleUser = "user"` and `RoleAssistant = "assistant"`.
> 2. `Request` with `Model`, `MaxTokens` and `Messages []Message` (JSON
>    names as in Chapter 1).
> 3. `Message` with `Role string` and `Content []ContentBlock`.
> 4. `ContentBlock` with `Type string` (`"type"`) and `Text string`
>    (`"text,omitempty"`).
> 5. A function `TextBlock(text string) ContentBlock` that returns a block
>    with `Type: "text"`. Callers use it instead of typing `"text"`
>    themselves.
> 6. `Response` with `ID`, `Model`, `Content []ContentBlock`, `StopReason`
>    and `Usage Usage` (JSON `id`, `model`, `content`, `stop_reason`,
>    `usage`).
> 7. `Usage` as its own named type now, with `InputTokens` and
>    `OutputTokens`, so tests can build one.
> 8. A method `func (r *Response) Text() string` that joins the `Text` of
>    every `"text"` block with `"\n"` between them. Collect them in a
>    `[]string` and use `strings.Join`. That's the printing loop from
>    Chapter 1, moved to where the data is.
>
> Leave `main.go` alone for now. It keeps working, because nothing uses
> the new package yet.

<details>
<summary>Show the solution</summary>

```go
// Package claude talks to the Claude Messages API.
package claude

import "strings"

// Who said a message. The API only knows these two.
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

// Request is the JSON body we send to /v1/messages.
type Request struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []Message `json:"messages"`
}

// Message is one turn of the conversation. Its content is a list of
// blocks, the same shape Claude's replies use, so a reply can go straight
// back into the conversation later.
type Message struct {
	Role    string         `json:"role"`
	Content []ContentBlock `json:"content"`
}

// ContentBlock is one piece of a message. Type says which kind it is,
// and which of the other fields are filled in. Today that's only "text";
// later chapters add fields for other kinds.
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"` // only for "text" blocks
}

// TextBlock makes a "text" block, so callers never type the "text"
// string themselves.
func TextBlock(text string) ContentBlock {
	return ContentBlock{Type: "text", Text: text}
}

// Response holds the parts of the reply we use. encoding/json skips
// any JSON field that has no matching Go field.
type Response struct {
	ID         string         `json:"id"`
	Model      string         `json:"model"`
	Content    []ContentBlock `json:"content"`
	StopReason string         `json:"stop_reason"`
	Usage      Usage          `json:"usage"`
}

// Usage counts the tokens (word pieces) the request used.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Text joins the text of all the text blocks in the reply, one per line.
// Blocks of other kinds are skipped.
func (r *Response) Text() string {
	var parts []string
	for _, block := range r.Content {
		if block.Type == "text" {
			parts = append(parts, block.Text)
		}
	}
	return strings.Join(parts, "\n")
}
```

</details>

Why one `ContentBlock` struct with a `Type` field, and not a separate Go
type for each kind? Because that's how the JSON looks: a list where each
item says its own kind. One struct decodes any item. Each later chapter
that meets a new kind adds the fields that kind uses, with `omitempty`,
and every existing line keeps working. Chapter 6 adds the fields for
`tool_use` and `tool_result` blocks.

## Step 2: An error type for "no"

In Chapter 1, a failed request printed the raw error JSON:

```
goagent: API returned 401 Unauthorized: {"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"},"request_id":"req_011ChAe38zgAH364fqLissKV"}
```

That's fine for a person reading it. Code can't do much with it, though.
In Chapter 4 goagent must decide "is this a `429`, too many requests? Then
wait and retry." Searching an error string for `"429"` would be fragile.
We want the status code as a number, in a field.

In Go, an error is any value that has an `Error() string` method. That's
all the built-in `error` interface asks for (an *interface* is a list of
methods; any type that has those methods satisfies it, with no
`implements` keyword):

```go
type error interface {
	Error() string
}
```

So we can make our own struct, give it the fields we care about, add an
`Error` method, and return it anywhere Go expects an `error`. Callers who
only want a message print it. Callers who want the fields can get them
back with `errors.As`, which you'll use in Step 4.

You'll decode the error body into a struct used only inside one function.
You can write that struct's type right where you declare the variable,
with no name. It's an anonymous struct, like the `Usage` field in Chapter 1:

```go
var wire struct {          // shape only
	Name string `json:"name"`
}
```

> **Your turn.** In `claude/error.go`:
>
> 1. A struct `APIError` with `StatusCode int`, `Type string`,
>    `Message string`, `RequestID string`.
> 2. A method `func (e *APIError) Error() string` that returns
>    `API error <status> <type>: <message> (request <id>)` using
>    `fmt.Sprintf`.
> 3. A helper `newAPIError(status int, body []byte) *APIError` that
>    unmarshals the body (shape: `{"type":"error","error":{"type":...,
>    "message":...},"request_id":...}`) into an anonymous struct and copies
>    the fields across. If `Unmarshal` fails, or the error type comes out
>    empty, the body wasn't the API's error shape. A proxy might send an
>    HTML page, for example. In that case return an `APIError` with only
>    `StatusCode` set and the whole body as `Message`, so nothing is lost.

<details>
<summary>Show the solution</summary>

```go
package claude

import (
	"encoding/json"
	"fmt"
)

// APIError is what Send returns when the API answers with a status other
// than 200 OK. Callers can read its fields (Chapter 4 retries on some
// status codes) instead of picking apart an error string.
type APIError struct {
	StatusCode int    // the HTTP status, for example 401
	Type       string // the API's error type, for example "authentication_error"
	Message    string // the API's explanation
	RequestID  string // quote this if you ever ask Anthropic for help
}

// Error makes *APIError satisfy Go's error interface.
func (e *APIError) Error() string {
	return fmt.Sprintf("API error %d %s: %s (request %s)",
		e.StatusCode, e.Type, e.Message, e.RequestID)
}

// newAPIError reads an error body like
//
//	{"type":"error","error":{"type":"...","message":"..."},"request_id":"..."}
//
// If the body isn't that shape (a proxy's HTML page, say), the raw body
// becomes the message, so nothing is lost.
func newAPIError(status int, body []byte) *APIError {
	var wire struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
		RequestID string `json:"request_id"`
	}
	if err := json.Unmarshal(body, &wire); err != nil || wire.Error.Type == "" {
		return &APIError{StatusCode: status, Message: string(body)}
	}
	return &APIError{
		StatusCode: status,
		Type:       wire.Error.Type,
		Message:    wire.Error.Message,
		RequestID:  wire.RequestID,
	}
}
```

</details>

Notice `%d` for the status code. It's an `int`, and `%d` is the verb for
whole numbers. `%s` is for strings. Give `%s` an `int` and Go doesn't
refuse; it prints `%!s(int=401)` instead of `401`. `go vet` usually
catches this, but not always: in Chapter 1, `fail` builds its format
string by joining two strings, and that hides the format from vet.

## Step 3: The client

Now the HTTP part of Chapter 1's `main` moves into the `claude` package,
as a method on a `Client`.

The important decision is *where the client's pieces come from*. The
client needs three things: the API key, the address to send to, and an
`*http.Client` to do the sending. It could make them itself, by reading
the environment and hard-coding `https://api.anthropic.com`. Then every
`Client` would always talk to the real API, and no test could stop it.

So the client makes none of them. `NewClient` takes all three as
parameters, and whoever builds the client decides. `main` passes the real
address. A test passes the address of a fake server. The client's code is
the same either way. That's called **dependency injection**: a piece of
code is handed the things it depends on instead of making them.

You need:

- **`fmt.Errorf` with `%w`.** In Chapter 1, every error called `fail` and
  exited. A package mustn't do that: only the program decides when to
  stop. So `Send` returns errors instead. `%w` wraps the original error
  inside a new one that adds context, and `errors.As` can still find the
  original later:

  ```go
  return fmt.Errorf("reading config: %w", err)   // shape only
  ```

- **`context.Context`** stays as in Chapter 1, but now `Send` takes it as
  its first parameter, the Go habit for anything that might wait on the
  network. We'll use it properly in Chapter 12, for Ctrl-C.

> **Your turn.** In `claude/client.go`:
>
> 1. Constants `DefaultBaseURL = "https://api.anthropic.com"` (exported,
>    so `main` can pass it) and `apiVersion = "2026-10-21"` (private).
> 2. A struct `Client` with private fields `apiKey`, `baseURL` (strings)
>    and `httpClient *http.Client`.
> 3. `NewClient(apiKey, baseURL string, httpClient *http.Client) *Client`.
> 4. A method `func (c *Client) Send(ctx context.Context, req Request)
>    (*Response, error)` that does what `main` did in Chapter 1, but sends
>    to `c.baseURL + "/v1/messages"` and returns every failure as an
>    error, each wrapped with `%w` and a few words saying which step
>    failed. A status other than 200 returns `newAPIError(...)`. On
>    success, return a pointer to the decoded `Response`.
>
> Then run `make check`.

<details>
<summary>Show the solution</summary>

```go
package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// DefaultBaseURL is the real API. Tests pass their fake server's URL instead.
const DefaultBaseURL = "https://api.anthropic.com"

// apiVersion is the API version we were written against.
const apiVersion = "2026-10-21"

// Client sends requests to the Messages API.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewClient takes everything the client depends on as parameters: the key,
// where to send requests, and the *http.Client that does the sending.
// It creates none of them itself, so a test can pass a fake server's URL
// and client.
func NewClient(apiKey, baseURL string, httpClient *http.Client) *Client {
	return &Client{apiKey: apiKey, baseURL: baseURL, httpClient: httpClient}
}

// Send makes one request and returns Claude's reply. A status other than
// 200 OK comes back as an *APIError.
func (c *Client) Send(ctx context.Context, req Request) (*Response, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", apiVersion)
	httpReq.Header.Set("content-type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close() // now this runs on every path, errors included

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading reply: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, newAPIError(resp.StatusCode, raw)
	}

	var reply Response
	if err := json.Unmarshal(raw, &reply); err != nil {
		return nil, fmt.Errorf("decoding reply: %w", err)
	}
	return &reply, nil
}
```

</details>

```sh
make check
```

```
/usr/local/go/bin/go vet ./...
/usr/local/go/bin/go test ./...
?   	goagent	[no test files]
?   	goagent/claude	[no test files]
```

`./...` means "this folder and every folder below it", so vet and test now
cover both packages. Both compile. Neither has tests yet.

The comment on `defer` is worth a second look. In Chapter 1, `fail` called
`os.Exit`, which ends the program on the spot, so deferred calls never ran.
Now `Send` *returns* on errors, and `defer resp.Body.Close()` runs on every
return.

## Step 4: Test the client against a fake server

To test `Send` we need something at the other end of the HTTP request that
answers like the API does. The standard library has one: package
`net/http/httptest`.

- **`httptest.NewServer(handler)`** starts a real HTTP server on your
  machine, on a free port, inside the test. `server.URL` is its address,
  like `http://127.0.0.1:41235`. `server.Client()` is an `*http.Client`
  set up to talk to it. `server.Close()` shuts it down.
- **The handler** is a function you write, called once for each request
  the server gets. It reads the request `r` and writes the answer to `w`:

  ```go
  http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {   // shape only
  	w.WriteHeader(http.StatusTeapot)   // status (leave it out for 200)
  	w.Write([]byte("body"))            // then the body
  })
  ```

This is why `NewClient` takes the URL and the `*http.Client`: the test
passes `server.URL` and `server.Client()`, and `Send` talks to the fake
without knowing. The real HTTP code runs, with real JSON and real headers.
Only the other end is fake.

### Test files and testify

A file whose name ends in `_test.go` holds tests. `go test` builds it;
`go build` ignores it. A test is a function `func TestXxx(t *testing.T)`.

We'll use **testify** to write the checks. It has two packages with the
same functions:

- **`require`** stops the test at once if the check fails. Use it for
  things the rest of the test needs, like "there was no error" before you
  use the result.
- **`assert`** records the failure and carries on. Use it for the actual
  checks, so one run shows every value that's wrong, not just the first.

The ones you need:

```go
require.NoError(t, err)                 // err must be nil
assert.Equal(t, expected, actual)       // expected first, then what you got
assert.JSONEq(t, `{"a":1}`, actualJSON) // same JSON, ignoring spacing and key order
require.ErrorAs(t, err, &target)        // err (or one it wraps) has target's type; fills target
```

One rule: inside the handler, use only `assert`, never `require`. The
handler runs on the server's own goroutine (a separate line of execution),
and `require` may only stop a test from the test's own goroutine.

Two more bits of the `testing` package:

- **`t.Cleanup(f)`** runs `f` when the test ends, pass or fail. It's like
  `defer`, but it works from a helper function too.
- **A helper function** keeps the setup in one place. Both tests need "a
  fake server plus a client pointed at it", so that becomes one function.
  That's DRY ("don't repeat yourself"): the second test is the moment it
  pays off.

> **Your turn.** In `claude/client_test.go`, in `package claude`:
>
> 1. A helper `fakeAPI(t *testing.T, handler http.HandlerFunc) *Client`
>    that starts an `httptest` server with the handler, registers
>    `server.Close` with `t.Cleanup`, and returns
>    `NewClient("test-key", server.URL, server.Client())`.
> 2. `TestSendReturnsReply`: the handler asserts the method is POST, the
>    path is `/v1/messages`, the three headers are right, and the body
>    (read it with `io.ReadAll(r.Body)`) is the JSON you expect, checked
>    with `assert.JSONEq`. Then it writes a small reply in the API's
>    shape. The test calls `Send` with a one-message request (built with
>    `TextBlock`), requires no error, and asserts the reply's `Text()`,
>    `StopReason` and `Usage`.
> 3. `TestSendReturnsAPIError`: the handler writes status 401 and an error
>    body. The test calls `Send`, uses `require.ErrorAs` to get an
>    `*APIError` out of the error, and asserts all four fields with one
>    `assert.Equal` against a whole `&APIError{...}`.

<details>
<summary>Show the solution</summary>

```go
package claude

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeAPI starts a local HTTP server that runs handler for every request,
// and a Client pointed at it. The server stops when the test ends.
func fakeAPI(t *testing.T, handler http.HandlerFunc) *Client {
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return NewClient("test-key", server.URL, server.Client())
}

func TestSendReturnsReply(t *testing.T) {
	client := fakeAPI(t, func(w http.ResponseWriter, r *http.Request) {
		// Check what the client sent: method, path, headers, body.
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v1/messages", r.URL.Path)
		assert.Equal(t, "test-key", r.Header.Get("x-api-key"))
		assert.Equal(t, "2026-10-21", r.Header.Get("anthropic-version"))
		assert.Equal(t, "application/json", r.Header.Get("content-type"))

		// Only assert (never require) in here: this runs on the server's
		// goroutine, and require may only stop a test from the test's own.
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.JSONEq(t, `{
			"model": "claude-haiku-5-5",
			"max_tokens": 100,
			"messages": [{"role": "user", "content": [{"type": "text", "text": "Hi"}]}]
		}`, string(body))

		// Answer like the real API does.
		w.Write([]byte(`{
			"id": "msg_1",
			"model": "claude-haiku-5-5",
			"content": [{"type": "text", "text": "Hello!"}],
			"stop_reason": "end_turn",
			"usage": {"input_tokens": 8, "output_tokens": 3}
		}`))
	})

	reply, err := client.Send(context.Background(), Request{
		Model:     "claude-haiku-5-5",
		MaxTokens: 100,
		Messages:  []Message{{Role: RoleUser, Content: []ContentBlock{TextBlock("Hi")}}},
	})

	require.NoError(t, err)
	assert.Equal(t, "Hello!", reply.Text())
	assert.Equal(t, "end_turn", reply.StopReason)
	assert.Equal(t, Usage{InputTokens: 8, OutputTokens: 3}, reply.Usage)
}

func TestSendReturnsAPIError(t *testing.T) {
	client := fakeAPI(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"type":"error","error":{"type":"authentication_error",` +
			`"message":"invalid x-api-key"},"request_id":"req_1"}`))
	})

	_, err := client.Send(context.Background(), Request{})

	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, &APIError{
		StatusCode: 401,
		Type:       "authentication_error",
		Message:    "invalid x-api-key",
		RequestID:  "req_1",
	}, apiErr)
}
```

</details>

The test file is in `package claude` too, so it can call the private
`newAPIError` if it ever needs to. Each test has three parts, split by
blank lines: set up, do the one thing, check. That pattern is often
called *arrange, act, assert*.

testify isn't part of the standard library, so Go has to download it.
`go mod tidy` reads your imports, fetches anything missing, and records
it in `go.mod` (and the exact file hashes in `go.sum`):

```sh
go mod tidy
```

```
go: finding module for package github.com/stretchr/testify/require
go: finding module for package github.com/stretchr/testify/assert
go: found github.com/stretchr/testify/assert in github.com/stretchr/testify v1.12.1
go: found github.com/stretchr/testify/require in github.com/stretchr/testify v1.12.1
```

Run `tidy` *after* writing the imports. If you `go get` testify before
any code imports it, `go.sum` can end up missing entries, and the next
`make check` tells you so.

```sh
make check
```

```
/usr/local/go/bin/go vet ./...
/usr/local/go/bin/go test ./...
?   	goagent	[no test files]
ok  	goagent/claude	0.004s
```

Two tests, four milliseconds, no key.

### Watch a test catch a mistake

A test that has never failed might not be checking anything. So break the
client on purpose: in `client.go`, change `apiVersion` to `"2023-06-01"`
and run the check again.

```sh
make check
```

```
/usr/local/go/bin/go vet ./...
/usr/local/go/bin/go test ./...
?   	goagent	[no test files]
--- FAIL: TestSendReturnsReply (0.00s)
    client_test.go:28: 
        	Error Trace:	/home/rijojohn/goagent-book/.book/scratch/ch02-step4/claude/client_test.go:28
        	            				/home/rijojohn/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.4.linux-amd64/src/net/http/server.go:2362
        	            				/home/rijojohn/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.4.linux-amd64/src/net/http/server.go:3440
        	            				/home/rijojohn/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.4.linux-amd64/src/net/http/server.go:2161
        	            				/home/rijojohn/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.4.linux-amd64/src/runtime/asm_amd64.s:1264
        	Error:      	Not equal: 
        	            	expected: "2026-10-21"
        	            	actual  : "2023-06-01"
        	            	
        	            	Diff:
        	            	--- Expected
        	            	+++ Actual
        	            	@@ -1 +1 @@
        	            	-2026-10-21
        	            	+2023-06-01
        	Test:       	TestSendReturnsReply
FAIL
FAIL	goagent/claude	0.005s
FAIL
make: *** [Makefile:10: check] Error 1
```

Look at three lines: `client_test.go:28` is the check that failed (the
`anthropic-version` header), and `expected` / `actual` say what was wrong.
The rest of the error trace is the server code that called your handler,
because the check ran inside it. Your paths will differ. Put the version back
and make sure `make check` passes again.

## Step 5: `run`, and the interface it needs

`main` still does everything itself. Now we take out the part worth
testing: asking the question and printing the answer. We put it in a
function `run` that `main` calls.

`run` needs something it can call `Send` on. If its parameter is a
`*claude.Client`, a test of `run` needs a fake server again, plus JSON it
doesn't care about. `run` doesn't care about HTTP. It needs exactly one
thing: *"send this request, give me a reply"*. So we write that down as an
interface, in `main`, next to `run`:

```go
type messageSender interface {
	Send(ctx context.Context, req claude.Request) (*claude.Response, error)
}
```

`*claude.Client` already has that method, so it satisfies
`messageSender` without changing a line. A test can write a tiny type with
the same method that returns a ready-made reply.

This is **dependency inversion**, the "D" in SOLID. It means the code
that *uses* a dependency, `run`, says what it needs as an interface, and
the real thing, `claude.Client`, fits that. The interface lives with
`run`, not in the `claude` package. The `claude` package doesn't know
`messageSender` exists. Go makes this easy, because a type satisfies an
interface just by having the methods. Keep the interface small: one
method is easy to fake, and easy for later pieces to satisfy too.

You've already used this idea without noticing. `fmt.Fprintln(w, ...)`
takes an **`io.Writer`**, an interface with one method, `Write`.
`os.Stdout` is an `io.Writer`, and so is a **`bytes.Buffer`**, which just
keeps what's written in memory. So `run` takes `stdout, stderr io.Writer`:
`main` passes the real ones, and a test passes buffers and reads them back
with `buf.String()`.

> **Your turn.** Rewrite `main.go`:
>
> 1. Import `"goagent/claude"` (your module path + `/claude`). Keep only
>    the constants `main` still uses: `apiKeyEnvVar`, `model`,
>    `maxTokens`.
> 2. The `messageSender` interface above.
> 3. `run(ctx context.Context, sender messageSender, prompt string, stdout,
>    stderr io.Writer) error`: build the request with one user message,
>    call `Send`, return its error as it is, otherwise print `reply.Text()`
>    to `stdout` and the `[stop: ..., tokens in: ..., out: ...]` line to
>    `stderr`.
> 4. `main` keeps the key and prompt checks and `fail`. Then it builds the
>    client with `claude.NewClient(apiKey, claude.DefaultBaseURL,
>    &http.Client{Timeout: 2 * time.Minute})`, calls `run` with
>    `os.Stdout` and `os.Stderr`, and on error calls `fail("%v", err)`.
>
> Then in `main_test.go` (`package main`):
>
> 5. A `fakeSender` struct with fields `got claude.Request`, `reply
>    *claude.Response`, `err error`, and a `Send` method (pointer
>    receiver) that saves the request in `got` and returns `reply, err`.
> 6. `TestRunPrintsAnswer`: a fake with a one-block reply, two
>    `bytes.Buffer`s for output. Call `run`, then check what each buffer
>    holds, and that `sender.got.Messages` is the one user message you
>    asked.
> 7. `TestRunReturnsSendError`: a fake whose `err` is
>    `errors.New("network down")`. `run` must return that error
>    (`require.EqualError(t, err, "network down")` checks the message) and
>    print nothing to stdout (`assert.Empty`).

<details>
<summary>Show the solution: main.go</summary>

```go
// Command goagent sends one question to Claude and prints the answer.
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"goagent/claude"
)

// Names for values that would otherwise be "magic" strings in the code.
const (
	apiKeyEnvVar = "ANTHROPIC_API_KEY"
	model        = "claude-haiku-5-5"
	maxTokens    = 1024 // the most tokens (word pieces) the reply may use
)

// messageSender is the one thing run needs from Claude: send a request,
// get a reply. run depends on this small interface, not on *claude.Client,
// so a test can hand it a fake.
type messageSender interface {
	Send(ctx context.Context, req claude.Request) (*claude.Response, error)
}

// run asks one question and prints the answer to stdout and the token
// counts to stderr. It returns errors instead of exiting, so a test can
// check them.
func run(ctx context.Context, sender messageSender, prompt string, stdout, stderr io.Writer) error {
	reply, err := sender.Send(ctx, claude.Request{
		Model:     model,
		MaxTokens: maxTokens,
		Messages: []claude.Message{
			{Role: claude.RoleUser, Content: []claude.ContentBlock{claude.TextBlock(prompt)}},
		},
	})
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, reply.Text())
	fmt.Fprintf(stderr, "[stop: %s, tokens in: %d, out: %d]\n",
		reply.StopReason, reply.Usage.InputTokens, reply.Usage.OutputTokens)
	return nil
}

// fail prints a message to standard error and stops the program
// with exit code 1. Only main calls it now.
func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "goagent: "+format+"\n", args...)
	os.Exit(1)
}

func main() {
	// The key comes from the environment, never from the source code.
	apiKey := os.Getenv(apiKeyEnvVar)
	if apiKey == "" {
		fail("%s is not set", apiKeyEnvVar)
	}

	// Everything after the program name is the question.
	prompt := strings.Join(os.Args[1:], " ")
	if prompt == "" {
		fail(`usage: goagent "your question"`)
	}

	// main builds the real pieces and hands them to run.
	client := claude.NewClient(apiKey, claude.DefaultBaseURL,
		&http.Client{Timeout: 2 * time.Minute})
	if err := run(context.Background(), client, prompt, os.Stdout, os.Stderr); err != nil {
		fail("%v", err)
	}
}
```

</details>

<details>
<summary>Show the solution: main_test.go</summary>

```go
package main

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"goagent/claude"
)

// fakeSender satisfies messageSender without any network. It remembers
// the request it got and answers with whatever the test put in it.
type fakeSender struct {
	got   claude.Request   // the request run sent
	reply *claude.Response // what to answer
	err   error            // or the error to return
}

func (f *fakeSender) Send(ctx context.Context, req claude.Request) (*claude.Response, error) {
	f.got = req
	return f.reply, f.err
}

func TestRunPrintsAnswer(t *testing.T) {
	sender := &fakeSender{reply: &claude.Response{
		Content:    []claude.ContentBlock{claude.TextBlock("Paris.")},
		StopReason: "end_turn",
		Usage:      claude.Usage{InputTokens: 16, OutputTokens: 3},
	}}
	var stdout, stderr bytes.Buffer

	err := run(context.Background(), sender, "Capital of France?", &stdout, &stderr)

	require.NoError(t, err)
	assert.Equal(t, "Paris.\n", stdout.String())
	assert.Equal(t, "[stop: end_turn, tokens in: 16, out: 3]\n", stderr.String())
	// run asked the question we gave it, as one user message.
	assert.Equal(t, []claude.Message{
		{Role: claude.RoleUser, Content: []claude.ContentBlock{claude.TextBlock("Capital of France?")}},
	}, sender.got.Messages)
}

func TestRunReturnsSendError(t *testing.T) {
	sender := &fakeSender{err: errors.New("network down")}
	var stdout, stderr bytes.Buffer

	err := run(context.Background(), sender, "Hi", &stdout, &stderr)

	require.EqualError(t, err, "network down")
	assert.Empty(t, stdout.String())
}
```

</details>

Why does `Send` on `fakeSender` have a pointer receiver? It changes the
fake, by saving `got`. With a plain `fakeSender` receiver, it would save
into a copy that's thrown away, and the test would always see an empty
request. That's also why the tests pass `&fakeSender{...}`, a pointer.

```sh
make check
```

```
/usr/local/go/bin/go vet ./...
/usr/local/go/bin/go test ./...
ok  	goagent	0.006s
ok  	goagent/claude	0.008s
```

Four tests in two packages, all passing.

## Try it

The tests prove the pieces work alone. Now check they still work together
against the real API:

```sh
make build
./goagent "What is the capital of France?"
```

```
/usr/local/go/bin/go build -o goagent .
The capital of France is **Paris**.
[stop: end_turn, tokens in: 16, out: 14]
```

The same answer as Chapter 1, and the same 16 input tokens. We now send
`content` as a list with one text block instead of a plain string, and
the token count proves the API sees no difference.

Now the wrong key:

```sh
ANTHROPIC_API_KEY=sk-ant-not-a-real-key ./goagent "What is the capital of France?"
```

```
goagent: API error 401 authentication_error: invalid x-api-key (request req_011ChCPV1iccB5yTNYk9uLNa)
```

Compare it with Chapter 1's raw JSON:

```
goagent: API returned 401 Unauthorized: {"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"},"request_id":"req_011ChAe38zgAH364fqLissKV"}
```

It's the same information, but it was read into fields first and only
then printed. In Chapter 4, code will read the `StatusCode` field
instead of a person reading the line.

## What this chapter doesn't do

- **`main` has no tests.** Reading the environment and calling `os.Exit`
  are left in `main` on purpose. It's a few lines, and testing it would
  mean running goagent as a separate program. Everything with real logic
  lives in `run` and `claude`, which are tested.
- **Only text blocks have fields.** A `thinking` block or a `tool_use`
  block would decode as a `ContentBlock` with only `Type` set. Its other
  fields would be dropped, and `Text()` skips it. Chapter 6 adds the fields
  for tool blocks.
- **Still one question per run.** Chapter 3 keeps a conversation.
- **No retries.** `APIError` holds the status code, but nothing reads it
  yet. Chapter 4 does.

## Commit

```sh
make check
git add -A
git commit -m "Chapter 2: claude package, Client, and first tests"
```

## What you should now be able to answer

Try to answer each one in your own words first. Then open the answer
to check.

**1. Why does `NewClient` take the base URL and the `*http.Client` as
parameters?**

<details>
<summary>Answer</summary>

So the code that builds the client decides where it sends requests and
how. `main` passes the real API's address. A test passes an `httptest`
server's `URL` and `Client()`, and the same `Send` code talks to the fake.
If the client made these itself, every client would always call the real
API.

</details>

**2. What's the difference between `require` and `assert`?**

<details>
<summary>Answer</summary>

`require` stops the test at once when the check fails. Use it when the
rest of the test can't go on, for example no error before you use the
result. `assert` records the failure and keeps going, so you see every
wrong value in one run. Inside an `httptest` handler, use only `assert`,
because the handler runs on another goroutine and `require` may only stop
the test from the test's own goroutine.

</details>

**3. `run` takes a `messageSender`, not a `*claude.Client`. What does that
buy, and where does the interface live?**

<details>
<summary>Answer</summary>

`run` needs only "send a request, get a reply", so a test can pass a tiny
fake with one `Send` method and no HTTP at all. The interface lives next
to `run`, in `main`, the code that uses it. `claude.Client` satisfies it
just by having a matching `Send` method, without knowing the interface
exists. That's dependency inversion.

</details>

**4. How does a caller get the status code out of an error that `Send`
returned?**

<details>
<summary>Answer</summary>

With `errors.As`:

```go
var apiErr *claude.APIError
if errors.As(err, &apiErr) {
	// use apiErr.StatusCode
}
```

`errors.As` looks through the error, and through any errors wrapped
inside it with `%w`, for one of type `*claude.APIError`. If it finds one,
it fills in `apiErr` and returns true.

</details>

**5. We now send `content` as a list of blocks. Did that change what the
API saw?**

<details>
<summary>Answer</summary>

No. A string is the shorthand for a list holding one text block, and the
real run used the same 16 input tokens as Chapter 1. We use the list so
one `Message` type fits both what we send and what Claude replies, so a
reply can go back into the history as it is.

</details>

**6. Why did we break `apiVersion` on purpose?**

<details>
<summary>Answer</summary>

To see the test fail. A test that has never failed might not be checking
what you think. Seeing it go red on a real mistake, with a clear
"expected / actual" message, shows that it does.

</details>

## Next chapter

goagent still forgets everything between runs, and even within one run it
can only ask a single question. Ask "What's the capital of France?", then
"And its population?", and the second question means nothing to Claude.
The API doesn't remember: every request must carry the whole conversation
so far. In Chapter 3 goagent becomes a chat prompt. It reads your
questions one line at a time and keeps the conversation in a slice of
`claude.Message`. After each answer it appends Claude's reply, as the
blocks it came in. It also gets a *system prompt*, the standing
instructions that shape every answer. The `RoleAssistant` constant you
wrote today gets its first use.
