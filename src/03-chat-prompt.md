# Chapter 3: A chat prompt

goagent can ask one question. In this chapter it learns to hold a
conversation: you type a question, it answers, you type the next one, and
Claude remembers what was said before. We also give it a *system prompt*,
standing instructions that shape every answer.

On the way we hit a real surprise from the API, and fix it. The fix keeps
a promise Chapter 2 made: Claude's reply goes back into the conversation
exactly as it came.

New in this chapter: reading input line by line (`bufio.Scanner`), how Go
slices share memory, the system prompt, Claude's "thinking" blocks, and
teaching `encoding/json` to encode and decode a type your own way.

## Where we are

1. ✓ **Talk to the model** (Chapters 1–2)
2. ▶ **Hold a conversation** (Chapters 3–5)
3. · Loop (Chapters 6–7)
4. · Tools (Chapters 8–11)
5. · Stay in control (Chapters 12–13)
6. · MCP (Chapters 14–18)

Here's the gap. With Chapter 2's goagent, ask a follow-up question:

```sh
./goagent "And its population?"
```

```
I don't have a previous message to go on, so I'm not sure what place you mean. Which city, country, or region would you like the population for?
[stop: end_turn, tokens in: 12, out: 87]
```

Claude has no idea what "its" means. That's not goagent forgetting: the
API itself keeps nothing between requests. The docs say it plainly: *"The
Messages API is stateless, which means that you always send the full
conversational history to the API."* Each request is a stranger. If you
want Claude to remember the first question, the second request has to
carry it.

This matters for everything that follows. An agent is a conversation that
goes on for many turns: Claude asks for a tool, we run it, we send the
result back, Claude asks for another. Every one of those turns rides in
the history we're about to build. Get the history wrong and the agent
falls apart.

By the end of this chapter you can:

- read a terminal's input one line at a time, and test that code with a
  string
- keep a conversation in a slice and send all of it with every request
- explain how two Go slices can share memory, and when that bites
- make a Go type decide its own JSON, so blocks we don't understand still
  go back unchanged

## The plan

A conversation on the wire is the `messages` list we already send, with
more entries. Turns go user, assistant, user, and the newest question is
last. The system prompt is a separate top-level field, `system`. It isn't
a turn in the conversation:

```json
{
  "model": "claude-haiku-5-5",
  "max_tokens": 1024,
  "system": "You are goagent, an assistant in a terminal. ...",
  "messages": [
    {"role": "user",      "content": [{"type": "text", "text": "What is the capital of France?"}]},
    {"role": "assistant", "content": [{"type": "text", "text": "Paris."}]},
    {"role": "user",      "content": [{"type": "text", "text": "And its population?"}]}
  ]
}
```

I sent exactly this with `curl` before writing any Go, and Claude answered
about Paris's population. So the plan is:

1. Add the `system` field to `claude.Request`.
2. Turn `run` into a chat loop: read a line, add it to the history, send
   the whole history, add the reply to the history, print it, repeat.
3. Move the history into its own small type, so `run` only deals with
   the terminal.
4. Run it for real, meet Claude's thinking blocks, and make every block
   from the API go back byte for byte.

## Step 1: A system prompt

A **system prompt** is text that's sent with every request, apart from
the conversation. Use it for things that should hold for every answer:
who the assistant is, what it should and shouldn't do, how to format
answers. Chapter 2's goagent answered `The capital of France is
**Paris**.` The `**` is Markdown for bold, and in a terminal it's just
noise. A system prompt that says "plain text, without Markdown" fixes that
for every answer at once.

The API takes `system` as a string (it also takes a list of text blocks,
which we don't need). It's optional, so the field should vanish from the
JSON when it's empty. That's `omitempty` again, from Chapter 2: it leaves
a field out when it holds its zero value, here `""`.

> **Your turn.**
>
> 1. In `claude/types.go`, add a field `System string` to `Request`,
>    with the JSON tag `json:"system,omitempty"`. Put it after
>    `MaxTokens`. Order doesn't matter to the API; it reads like the
>    request above.
> 2. In `claude/client_test.go`, in `TestSendReturnsReply`, set `System:
>    "Be brief."` in the request you send, and add `"system": "Be
>    brief."` to the JSON the fake server expects. Now the test also
>    checks that the field reaches the wire with the right name.

<details>
<summary>Show the solution</summary>

In `claude/types.go`:

```go
// Request is the JSON body we send to /v1/messages.
type Request struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system,omitempty"` // standing instructions; left out when empty
	Messages  []Message `json:"messages"`
}
```

In `claude/client_test.go`, the expected body and the request:

```go
		assert.JSONEq(t, `{
			"model": "claude-haiku-5-5",
			"max_tokens": 100,
			"system": "Be brief.",
			"messages": [{"role": "user", "content": [{"type": "text", "text": "Hi"}]}]
		}`, string(body))
```

```go
	reply, err := client.Send(context.Background(), Request{
		Model:     "claude-haiku-5-5",
		MaxTokens: 100,
		System:    "Be brief.",
		Messages:  []Message{{Role: RoleUser, Content: []ContentBlock{TextBlock("Hi")}}},
	})
```

</details>

```sh
make check
```

```
/usr/local/go/bin/go vet ./...
/usr/local/go/bin/go test ./...
ok  	goagent	0.005s
ok  	goagent/claude	0.004s
```

## Step 2: A chat loop

`run` changes the most. It stops taking one question as a parameter, and
reads questions from the terminal until you're done.

### Reading lines

A program's input is called **standard input**, or stdin. When you type
into a running program, that's stdin. When you pipe something into it
(`echo hi | ./goagent`), that's stdin too. Go gives you `os.Stdin`.

`os.Stdin` is an **`io.Reader`**: an interface with one method, `Read`,
that hands out bytes. It's the reading twin of the `io.Writer` from
Chapter 2. So `run` takes `stdin io.Reader`, just as it takes `stdout
io.Writer`. `main` passes `os.Stdin`, and a test passes a string made
into a reader with `strings.NewReader("first\nsecond\n")`.

To read those bytes one line at a time, wrap the reader in a
**`bufio.Scanner`**:

```go
lines := bufio.NewScanner(r)   // shape only
for lines.Scan() {             // Scan reads the next line; false at the end
	fmt.Println(lines.Text())  // Text is that line, without its "\n"
}
err := lines.Err()             // nil if the input simply ended
```

`Scan` returns `false` in two cases: the input ended, or reading it
failed. `Err` tells them apart. It returns `nil` when the input just
ended. In a terminal, you end the input by pressing **Ctrl-D** on an
empty line. A pipe ends when the command before it finishes.

### The history

The conversation is a slice, `var history []claude.Message`, that grows
by two messages per question:

1. Append the question, as a user message.
2. Send a request whose `Messages` is the whole `history`.
3. Append Claude's reply, as an assistant message whose `Content` is
   `reply.Content`, the blocks exactly as they came. This is why
   Chapter 2 made our messages use blocks too.

If `Send` fails, the question has no answer. Leaving it in would give the
next request two user turns in a row. The API would quietly merge them
into one turn, so Claude would see both questions as one. So take it back
out with `history = history[:len(history)-1]` (a slice of everything but
the last item), print the error, and let the user carry on. One network
hiccup shouldn't end the chat.

### Where the prompt goes

Before each read, `run` prints `> ` so you know it's waiting. That goes
to **stderr**, like the token counts. stdout carries answers and nothing
else, so `./goagent > answers.txt` saves clean answers, and a test of
stdout doesn't have to skip prompts.

> **Your turn.** In `main.go`:
>
> 1. Add a constant `systemPrompt`. The book uses `"You are goagent, an
>    assistant in a terminal. Answer in plain text, without Markdown.
>    Keep answers short."` Write your own if you like.
> 2. Change `run` to `run(ctx context.Context, sender messageSender, stdin
>    io.Reader, stdout, stderr io.Writer) error`. Make a `history`
>    slice and a `bufio.Scanner` on `stdin`. Then loop:
>    - print `> ` to `stderr`;
>    - `Scan`; if it returns `false`, leave the loop;
>    - trim the line with `strings.TrimSpace`; if it's empty, go round
>      again (`continue`);
>    - append the user message; `Send` a request with `System:
>      systemPrompt` and `Messages: history`;
>    - on an error, remove the question, print `goagent: <the error>` to
>      `stderr`, and `continue`;
>    - otherwise append the assistant message, then print the answer and
>      the `[stop: ...]` line as before.
>
>    After the loop, print a newline to `stderr` (the last `> ` has no
>    line end), and return `lines.Err()`.
> 3. In `main`, drop the `os.Args` question and its usage check, and
>    pass `os.Stdin` to `run`.

<details>
<summary>Show the solution: run</summary>

```go
// Names for values that would otherwise be "magic" strings in the code.
const (
	apiKeyEnvVar = "ANTHROPIC_API_KEY"
	model        = "claude-haiku-5-5"
	maxTokens    = 1024 // the most tokens (word pieces) one reply may use

	// systemPrompt is sent with every request. It shapes every answer
	// but isn't part of the conversation itself.
	systemPrompt = "You are goagent, an assistant in a terminal. " +
		"Answer in plain text, without Markdown. Keep answers short."
)
```

```go
// run reads questions from stdin, one per line, until the input ends.
// It sends each one with the whole conversation so far, prints the answer
// to stdout, and the prompt and token counts to stderr.
func run(ctx context.Context, sender messageSender, stdin io.Reader, stdout, stderr io.Writer) error {
	var history []claude.Message // the conversation so far, oldest first
	lines := bufio.NewScanner(stdin)

	for {
		fmt.Fprint(stderr, "> ")
		if !lines.Scan() {
			break // the input ended (Ctrl-D), or reading it failed
		}
		question := strings.TrimSpace(lines.Text())
		if question == "" {
			continue // an empty line: just ask again
		}

		// The question joins the history, and the whole history is sent:
		// the API remembers nothing between requests.
		history = append(history, claude.Message{
			Role:    claude.RoleUser,
			Content: []claude.ContentBlock{claude.TextBlock(question)},
		})
		reply, err := sender.Send(ctx, claude.Request{
			Model:     model,
			MaxTokens: maxTokens,
			System:    systemPrompt,
			Messages:  history,
		})
		if err != nil {
			// Take the unanswered question back out, so the history
			// still goes user, assistant, user, ... and keep chatting.
			history = history[:len(history)-1]
			fmt.Fprintf(stderr, "goagent: %v\n", err)
			continue
		}

		// Claude's reply joins the history as it came: the same blocks.
		history = append(history, claude.Message{
			Role:    claude.RoleAssistant,
			Content: reply.Content,
		})
		fmt.Fprintln(stdout, reply.Text())
		fmt.Fprintf(stderr, "[stop: %s, tokens in: %d, out: %d]\n",
			reply.StopReason, reply.Usage.InputTokens, reply.Usage.OutputTokens)
	}

	fmt.Fprintln(stderr) // end the last "> " line
	return lines.Err()   // nil when the input simply ended
}
```

</details>

<details>
<summary>Show the solution: main</summary>

```go
func main() {
	// The key comes from the environment, never from the source code.
	apiKey := os.Getenv(apiKeyEnvVar)
	if apiKey == "" {
		fail("%s is not set", apiKeyEnvVar)
	}

	// main builds the real pieces and hands them to run.
	client := claude.NewClient(apiKey, claude.DefaultBaseURL,
		&http.Client{Timeout: 2 * time.Minute})
	if err := run(context.Background(), client, os.Stdin, os.Stdout, os.Stderr); err != nil {
		fail("%v", err)
	}
}
```

`strings` is still imported: `run` uses `strings.TrimSpace`. Add `bufio`
to the imports.

</details>

```sh
make check
```

```
/usr/local/go/bin/go vet ./...
# goagent
# [goagent]
vet: ./main_test.go:36:43: cannot use "Capital of France?" (constant of type string) as io.Reader value in argument to run: string does not implement io.Reader (missing method Read)
make: *** [Makefile:9: check] Error 1
```

That's the old tests, which still pass a string where `run` now wants an
`io.Reader`. They tested asking one question. Time to replace them.

### Tests for a conversation

The fake sender needs two upgrades:

- **Several answers, in order.** A conversation means several calls to
  `Send`. Give the fake a list of scripted answers. Each one is a reply
  or an error. Call number one gets the first answer, call two the
  second, and so on.
- **Every request it got,** in a slice `got []claude.Request`, so a test
  can look at the second request and check it carried the first
  question and answer.

Two tests:

- **`TestRunRemembersTheConversation`**: stdin holds two questions with an
  empty line between them. Check stdout holds both answers, and that the
  second request's messages are user, assistant, user, in that order. Also
  check that it carries the system prompt.
- **`TestRunDropsAQuestionThatFailed`**: the first `Send` returns an error
  and the second works. Check the error reached stderr, that `run` still
  returned `nil`, and that each request held only its own question.

Small helpers keep the tests readable. `textReply("Paris.")` builds a
reply, and `userText` / `assistantText` build one history message. Without
them, every expected message is a long line of struct literals. That's
DRY, again: say it once, use it everywhere.

> **Your turn.** Rewrite `main_test.go`:
>
> 1. A struct `answer` with `reply *claude.Response` and `err error`.
> 2. `fakeSender` with `answers []answer` and `got []claude.Request`.
>    Its `Send` appends `req` to `got`, then returns the answer at
>    position `len(f.got)-1`.
> 3. The helpers `textReply`, `userText`, `assistantText`.
> 4. The two tests above. `assert.Contains(t, s, part)` checks that a
>    string contains a smaller one.

<details>
<summary>Show the solution: main_test.go (first try)</summary>

This is the test file as I first wrote it. Write yours the same way, with
`Send` storing `req` as it comes. There's a lesson in what happens next.

```go
package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"goagent/claude"
)

// answer is one scripted response from fakeSender: a reply, or an error.
type answer struct {
	reply *claude.Response
	err   error
}

// fakeSender satisfies messageSender without any network. It hands out
// its answers in order and remembers every request it got.
type fakeSender struct {
	answers []answer         // what to return, one per call
	got     []claude.Request // every request run sent, in order
}

func (f *fakeSender) Send(ctx context.Context, req claude.Request) (*claude.Response, error) {
	f.got = append(f.got, req)
	next := f.answers[len(f.got)-1]
	return next.reply, next.err
}

// textReply builds a finished reply holding one text block.
func textReply(text string) *claude.Response {
	return &claude.Response{
		Content:    []claude.ContentBlock{claude.TextBlock(text)},
		StopReason: "end_turn",
		Usage:      claude.Usage{InputTokens: 10, OutputTokens: 2},
	}
}

// userText and assistantText build one message of the history.
func userText(text string) claude.Message {
	return claude.Message{Role: claude.RoleUser, Content: []claude.ContentBlock{claude.TextBlock(text)}}
}

func assistantText(text string) claude.Message {
	return claude.Message{Role: claude.RoleAssistant, Content: []claude.ContentBlock{claude.TextBlock(text)}}
}

func TestRunRemembersTheConversation(t *testing.T) {
	sender := &fakeSender{answers: []answer{
		{reply: textReply("Paris.")},
		{reply: textReply("About 2 million.")},
	}}
	// Two questions, with an empty line between them that run should skip.
	stdin := strings.NewReader("Capital of France?\n\nAnd its population?\n")
	var stdout, stderr bytes.Buffer

	err := run(context.Background(), sender, stdin, &stdout, &stderr)

	assert.NoError(t, err)
	assert.Equal(t, "Paris.\nAbout 2 million.\n", stdout.String())
	// The second request carries the whole conversation so far.
	assert.Equal(t, []claude.Message{
		userText("Capital of France?"),
		assistantText("Paris."),
		userText("And its population?"),
	}, sender.got[1].Messages)
	assert.Equal(t, systemPrompt, sender.got[1].System)
}

func TestRunDropsAQuestionThatFailed(t *testing.T) {
	sender := &fakeSender{answers: []answer{
		{err: errors.New("network down")},
		{reply: textReply("Hello!")},
	}}
	stdin := strings.NewReader("Hi\nHi again\n")
	var stdout, stderr bytes.Buffer

	err := run(context.Background(), sender, stdin, &stdout, &stderr)

	assert.NoError(t, err) // one failed question doesn't end the chat
	assert.Contains(t, stderr.String(), "goagent: network down\n")
	assert.Equal(t, "Hello!\n", stdout.String())
	// Each request held only its own question: the failed one was
	// taken back out before the next was added.
	assert.Equal(t, []claude.Message{userText("Hi")}, sender.got[0].Messages)
	assert.Equal(t, []claude.Message{userText("Hi again")}, sender.got[1].Messages)
}
```

</details>

```sh
make check
```

```
/usr/local/go/bin/go vet ./...
/usr/local/go/bin/go test ./...
--- FAIL: TestRunDropsAQuestionThatFailed (0.00s)
    main_test.go:89: 
        	Error Trace:	/home/rijojohn/goagent-book/.book/scratch/goagent/main_test.go:89
        	Error:      	Not equal: 
        	            	expected: []claude.Message{claude.Message{Role:"user", Content:[]claude.ContentBlock{claude.ContentBlock{Type:"text", Text:"Hi"}}}}
        	            	actual  : []claude.Message{claude.Message{Role:"user", Content:[]claude.ContentBlock{claude.ContentBlock{Type:"text", Text:"Hi again"}}}}
        	            	
        	            	Diff:
        	            	--- Expected
        	            	+++ Actual
        	            	@@ -6,3 +6,3 @@
        	            	     Type: (string) (len=4) "text",
        	            	-    Text: (string) (len=2) "Hi"
        	            	+    Text: (string) (len=8) "Hi again"
        	            	    }
        	Test:       	TestRunDropsAQuestionThatFailed
FAIL
FAIL	goagent	0.004s
ok  	goagent/claude	(cached)
FAIL
make: *** [Makefile:10: check] Error 1
```

The first request, the one that failed, now holds `"Hi again"`, the
*second* question. `run` sent `"Hi"`. Something changed the request
after it was sent.

### Two slices, one array

A Go slice doesn't hold its items. It points at an **array** (a fixed row
of items in memory) and records how many of them it uses (its *length*)
and how many fit (its *capacity*). Copying a slice copies that pointer,
not the items. So the slice inside `req.Messages`, which the fake kept,
and `run`'s `history` point at the *same* array.

Here's what happened, step by step:

```
history = append(history, "Hi")      array: ["Hi"]          history: length 1
Send(... Messages: history)          fake keeps a slice of length 1 → ["Hi"]
history = history[:0]                array: ["Hi"]          history: length 0
history = append(history, "Hi again") array: ["Hi again"]   ← same spot, overwritten
                                     the fake's slice still points there → ["Hi again"]
```

Taking the question back out didn't erase it. It only made `history`
shorter. The next `append` had room in the same array, so it wrote over
the spot the fake's slice still pointed at.

Is `run` wrong? No. The real `claude.Client` turns the request into JSON
bytes inside `Send`, and never looks at the slice again. Reusing the array
afterwards is safe, and it saves memory. The fake is what's wrong: it
keeps a slice that it doesn't own. The fix belongs in the fake. It should
keep its own copy, with **`slices.Clone`**, which makes a new array with
the same items:

```go
b := slices.Clone(a) // shape only: b has a's items, in its own array
```

> **Your turn.** In `fakeSender.Send`, before appending, replace
> `req.Messages` with a clone of itself. Import `"slices"`.

<details>
<summary>Show the solution</summary>

```go
func (f *fakeSender) Send(ctx context.Context, req claude.Request) (*claude.Response, error) {
	// Keep a copy of the messages: run goes on using its history slice
	// after Send returns, and could overwrite what this one points at.
	req.Messages = slices.Clone(req.Messages)
	f.got = append(f.got, req)
	next := f.answers[len(f.got)-1]
	return next.reply, next.err
}
```

</details>

```sh
make check
```

```
/usr/local/go/bin/go vet ./...
/usr/local/go/bin/go test ./...
ok  	goagent	0.003s
ok  	goagent/claude	(cached)
```

Keep this in mind for later chapters: **whenever you keep a slice that
someone else will go on changing, keep a clone.**

## Step 3: Split the conversation out of `run`

`run` now does two jobs. It works the terminal: reading lines, printing
prompts and answers. And it keeps the conversation: building messages,
sending them, taking a failed question back out. These change for
different reasons. In Chapter 5 the printing changes (words arrive one
at a time). In Chapter 6 the conversation changes (one question can take
several requests, with tools run in between). Each job should change
without touching the other. That's the *single responsibility* idea, the
"S" in SOLID: one piece of code, one reason to change.

So the conversation moves into its own type, in a new file `chat.go`:

```go
type chat struct {
	sender  messageSender
	history []claude.Message
}
```

It has one method, `ask(ctx, question) (*claude.Response, error)`, which
does steps 1 to 3 from the history section above, including the take-back
on error. `run` makes one `chat` before its loop and calls `ask` for each
question.

The tests don't change. They test `run` from outside: text in, text out,
requests sent. None of that changes when the code inside `run` is split
up. That's why we wrote tests of `run` first: they show the split broke
nothing.

> **Your turn.**
>
> 1. Create `chat.go` (`package main`) with the `chat` type and its
>    `ask` method. Use a pointer receiver, `(c *chat)`, because `ask`
>    changes `c.history`.
> 2. In `run`, replace the history code with `conversation :=
>    &chat{sender: sender}` before the loop, and `reply, err :=
>    conversation.ask(ctx, question)` inside it.

<details>
<summary>Show the solution: chat.go</summary>

```go
package main

import (
	"context"

	"goagent/claude"
)

// chat is one conversation with Claude. It knows nothing about terminals:
// run does the reading and printing, chat does the remembering.
type chat struct {
	sender  messageSender
	history []claude.Message // the conversation so far, oldest first
}

// ask sends question along with the whole conversation so far, because
// the API remembers nothing between requests. The question and the reply
// join the history only if the request worked, so the history always
// goes user, assistant, user, ...
func (c *chat) ask(ctx context.Context, question string) (*claude.Response, error) {
	c.history = append(c.history, claude.Message{
		Role:    claude.RoleUser,
		Content: []claude.ContentBlock{claude.TextBlock(question)},
	})
	reply, err := c.sender.Send(ctx, claude.Request{
		Model:     model,
		MaxTokens: maxTokens,
		System:    systemPrompt,
		Messages:  c.history,
	})
	if err != nil {
		c.history = c.history[:len(c.history)-1] // take the question back out
		return nil, err
	}

	// Claude's reply joins the history as it came: the same blocks.
	c.history = append(c.history, claude.Message{
		Role:    claude.RoleAssistant,
		Content: reply.Content,
	})
	return reply, nil
}
```

</details>

<details>
<summary>Show the solution: run</summary>

```go
// run reads questions from stdin, one per line, until the input ends.
// It prints each answer to stdout, and the prompt and token counts to
// stderr. chat does the remembering; run only does the terminal.
func run(ctx context.Context, sender messageSender, stdin io.Reader, stdout, stderr io.Writer) error {
	conversation := &chat{sender: sender}
	lines := bufio.NewScanner(stdin)

	for {
		fmt.Fprint(stderr, "> ")
		if !lines.Scan() {
			break // the input ended (Ctrl-D), or reading it failed
		}
		question := strings.TrimSpace(lines.Text())
		if question == "" {
			continue // an empty line: just ask again
		}

		reply, err := conversation.ask(ctx, question)
		if err != nil {
			fmt.Fprintf(stderr, "goagent: %v\n", err) // say so, keep chatting
			continue
		}
		fmt.Fprintln(stdout, reply.Text())
		fmt.Fprintf(stderr, "[stop: %s, tokens in: %d, out: %d]\n",
			reply.StopReason, reply.Usage.InputTokens, reply.Usage.OutputTokens)
	}

	fmt.Fprintln(stderr) // end the last "> " line
	return lines.Err()   // nil when the input simply ended
}
```

</details>

```sh
make check
```

```
/usr/local/go/bin/go vet ./...
/usr/local/go/bin/go test ./...
ok  	goagent	0.003s
ok  	goagent/claude	(cached)
```

## Step 4: Blocks we don't read must still go back

Time to talk to the real API. Build it and pipe in three questions.
`printf` prints its text with each `\n` turned into a line end, so
goagent reads three lines:

```sh
make build
```

```
/usr/local/go/bin/go build -o goagent .
```

```sh
printf 'What is the capital of France?\nAnd its population?\nWhat did I ask you first?\n' | ./goagent
```

```
> Paris.
[stop: end_turn, tokens in: 54, out: 6]
> Paris has about 2.1 million residents within city limits. The wider Paris metropolitan area has roughly 12 to 13 million.
[stop: end_turn, tokens in: 66, out: 99]
> goagent: API error 400 invalid_request_error: messages.3.content.0.thinking.thinking: Field required (request req_011ChKN3Z7BPzkCrurGBMwo1)
> 
exit 0
```

The memory works: "And its population?" got an answer about Paris. But
the third question failed with a 400, the API's "your request is wrong".
You may not see this failure on your first try. Run it a few times. Look
at the error, though. `messages.3.content.0` is the fourth message (the
API counts from 0), which is Claude's second reply, and its first block.
That block's type is `thinking`, and it's missing a field called
`thinking`. We never sent a thinking block, and our code doesn't know the
word.

### Claude thinks before it answers

Claude can *think* before it answers: work through the question, then
write the reply. On Haiku 5.5 this is on by default, and it's
*adaptive*: Claude decides each time whether a question needs it. "What
is the capital of France?" didn't (6 output tokens). The population
question did (99 output tokens for a two-sentence answer). When Claude
thinks, the reply starts with a block like this:

```json
{"type": "thinking", "thinking": "", "signature": "EosnCkYICxIMMb3L..."}
```

On Haiku 5.5 the thinking text itself is left out by default (`"thinking":
""`), but you still pay for it, in output tokens. The `signature` is a
seal the API checks when the block comes back. The docs are strict:
*"Thinking blocks must be passed back unmodified and in their original
order; a modified block results in a 400 `invalid_request_error`."*

And we modified it. Chapter 1 said that `encoding/json` skips any JSON
field our struct has no field for. That was handy for reading replies.
But now the reply goes *back*, and `ContentBlock` only has `Type` and
`Text`. So the block came in as `{"type":"thinking", "thinking":"",
"signature":"..."}` and went back out as `{"type":"thinking"}`.

We could add `Thinking` and `Signature` fields. But Claude has other block
kinds too (redacted thinking, and the tool blocks we meet in Chapter 6),
and the same problem would come back with each one. The general fix:
**a block that came from the API keeps its exact bytes, and goes back as
those bytes.** We still read `Type` and `Text` from it as before.

### Writing the test first

The bug only shows up sometimes, when Claude happens to think. A test can
make it happen every time. Decode some reply content holding a thinking
block, encode it back as part of a conversation, and compare:

> **Your turn.** Create `claude/types_test.go` (`package claude`) with
> `TestReplyBlocksGoBackUnchanged`:
>
> 1. A string `content` holding a JSON list of two blocks: `{"type":
>    "thinking", "thinking": "", "signature": "c2lnbmF0dXJl"}` and a text
>    block `"Paris."`.
> 2. `json.Unmarshal` it into a `[]ContentBlock`.
> 3. `json.Marshal` a `[]Message`: an assistant message holding those
>    blocks, then a user message built with `TextBlock`.
> 4. Check that `blocks[1].Text` is `"Paris."` (we still read what we
>    know), and use `assert.JSONEq` to check the encoded JSON holds the
>    thinking block with all three fields.

<details>
<summary>Show the solution</summary>

```go
package claude

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplyBlocksGoBackUnchanged(t *testing.T) {
	// A reply's content, with a block kind we don't read: Claude's
	// thinking, whose signature must come back exactly as it was sent.
	content := `[
		{"type": "thinking", "thinking": "", "signature": "c2lnbmF0dXJl"},
		{"type": "text", "text": "Paris."}
	]`
	var blocks []ContentBlock
	require.NoError(t, json.Unmarshal([]byte(content), &blocks))

	// Send them back as an assistant message, with a block we built.
	sent, err := json.Marshal([]Message{
		{Role: RoleAssistant, Content: blocks},
		{Role: RoleUser, Content: []ContentBlock{TextBlock("And its population?")}},
	})
	require.NoError(t, err)

	assert.Equal(t, "Paris.", blocks[1].Text) // we still read what we know
	assert.JSONEq(t, `[
		{"role": "assistant", "content": `+content+`},
		{"role": "user", "content": [{"type": "text", "text": "And its population?"}]}
	]`, string(sent))
}
```

The expected JSON is built by putting `content` into it with `+`, so the
blocks we check for are exactly the blocks we decoded.

</details>

```sh
make check
```

```
/usr/local/go/bin/go vet ./...
/usr/local/go/bin/go test ./...
ok  	goagent	0.003s
--- FAIL: TestReplyBlocksGoBackUnchanged (0.00s)
    types_test.go:29: 
        	Error Trace:	/home/rijojohn/goagent-book/.book/scratch/ch03-step3/claude/types_test.go:29
        	Error:      	Not equal: 
        	            	expected: []interface {}{map[string]interface {}{"content":[]interface {}{map[string]interface {}{"signature":"c2lnbmF0dXJl", "thinking":"", "type":"thinking"}, map[string]interface {}{"text":"Paris.", "type":"text"}}, "role":"assistant"}, map[string]interface {}{"content":[]interface {}{map[string]interface {}{"text":"And its population?", "type":"text"}}, "role":"user"}}
        	            	actual  : []interface {}{map[string]interface {}{"content":[]interface {}{map[string]interface {}{"type":"thinking"}, map[string]interface {}{"text":"Paris.", "type":"text"}}, "role":"assistant"}, map[string]interface {}{"content":[]interface {}{map[string]interface {}{"text":"And its population?", "type":"text"}}, "role":"user"}}
        	            	
        	            	Diff:
        	            	--- Expected
        	            	+++ Actual
        	            	@@ -3,5 +3,3 @@
        	            	   (string) (len=7) "content": ([]interface {}) (len=2) {
        	            	-   (map[string]interface {}) (len=3) {
        	            	-    (string) (len=9) "signature": (string) (len=12) "c2lnbmF0dXJl",
        	            	-    (string) (len=8) "thinking": (string) "",
        	            	+   (map[string]interface {}) (len=1) {
        	            	     (string) (len=4) "type": (string) (len=8) "thinking"
        	Test:       	TestReplyBlocksGoBackUnchanged
FAIL
FAIL	goagent/claude	0.005s
FAIL
make: *** [Makefile:10: check] Error 1
```

That's the bug, caught every time: the expected block has three fields,
and ours went back with just `type`.

### Making a type decide its own JSON

`encoding/json` checks every value it handles for two interfaces:

- **`json.Unmarshaler`**: a method `UnmarshalJSON(data []byte) error`.
  When decoding into a type that has it, `json.Unmarshal` hands it the
  raw JSON for that value and lets it do the work.
- **`json.Marshaler`**: a method `MarshalJSON() ([]byte, error)`. When
  encoding a type that has it, `json.Marshal` uses the bytes it returns.

It's the same idea as `messageSender`: `encoding/json` says what it needs
as an interface, and any type with the right method fits.

So `ContentBlock` gets:

- **A private field `raw json.RawMessage`.** `json.RawMessage` is just
  `[]byte` that holds JSON. The field starts with a lower-case letter, so
  `encoding/json` ignores it (just like other packages can't see it), and
  code outside `claude` can't touch it. It's empty for blocks we build
  with `TextBlock`.
- **`UnmarshalJSON`** decodes `Type` and `Text` the normal way, then keeps
  a copy of `data` in `raw`. It must be a copy: the `encoding/json` docs
  say *"UnmarshalJSON must copy the JSON data if it wishes to retain the
  data after returning."* The decoder may reuse those bytes, the same
  trap as Step 2's slices. `slices.Clone` makes the copy.
- **`MarshalJSON`** returns `raw` if it's set, and otherwise encodes the
  block the normal way.

"The normal way" has one trick to it. Inside `UnmarshalJSON`, calling
`json.Unmarshal(data, b)` would see that `b` has an `UnmarshalJSON`
method and call it, which would call `json.Unmarshal`, which would call
it again, forever. The way out is a second type with the same fields but
none of the methods:

```go
type plain ContentBlock           // shape only: same fields, no methods
json.Unmarshal(data, (*plain)(b)) // b seen as a *plain: normal decoding
```

A new type declared from another (`type plain ContentBlock`) gets the
fields but not the methods. And because the two have the same fields, Go
lets you convert between them: `(*plain)(b)` is the same memory as `b`,
seen as the other type. Declare `plain` inside each method; nothing else
needs it.

Last detail: give `MarshalJSON` a *value* receiver, `(b ContentBlock)`,
not a pointer one. Messages hold blocks as values in a slice, and a value
receiver's method works for values and pointers both.

> **Your turn.** In `claude/types.go`:
>
> 1. Add the `raw json.RawMessage` field to `ContentBlock`.
> 2. Add `func (b *ContentBlock) UnmarshalJSON(data []byte) error`.
> 3. Add `func (b ContentBlock) MarshalJSON() ([]byte, error)`.
>
> Import `encoding/json` and `slices`.

<details>
<summary>Show the solution</summary>

```go
// ContentBlock is one piece of a message. Type says which kind it is,
// and which of the other fields are filled in. We only read "text"
// blocks, but Claude also sends kinds we don't read (like "thinking").
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"` // only for "text" blocks

	// raw is the block exactly as the API sent it, fields we don't read
	// included. Empty for blocks we build ourselves.
	raw json.RawMessage
}

// UnmarshalJSON makes *ContentBlock satisfy json.Unmarshaler, so
// encoding/json calls it to decode a block. It reads the fields we know,
// and keeps the whole block as it came.
func (b *ContentBlock) UnmarshalJSON(data []byte) error {
	// plain has ContentBlock's fields but not its methods, so decoding
	// into it uses encoding/json's normal rules, not this method again.
	type plain ContentBlock
	if err := json.Unmarshal(data, (*plain)(b)); err != nil {
		return err
	}
	b.raw = slices.Clone(data) // data isn't ours to keep: copy it
	return nil
}

// MarshalJSON makes ContentBlock satisfy json.Marshaler. A block that
// came from the API goes back byte for byte, as the API requires;
// a block we built is encoded the normal way.
func (b ContentBlock) MarshalJSON() ([]byte, error) {
	if b.raw != nil {
		return b.raw, nil
	}
	type plain ContentBlock
	return json.Marshal(plain(b))
}
```

</details>

```sh
make check
```

```
/usr/local/go/bin/go vet ./...
/usr/local/go/bin/go test ./...
ok  	goagent	0.003s
ok  	goagent/claude	0.005s
```

The other tests didn't need to change. `TestRunRemembersTheConversation`
still compares messages built by hand, and those have no `raw`, so they
compare equal as before.

## Try it

```sh
make build
```

```sh
printf 'What is the capital of France?\nAnd its population?\nWhat did I ask you first?\n' | ./goagent
```

```
> The capital of France is Paris.
[stop: end_turn, tokens in: 54, out: 12]
> Paris has about 2.1 million residents within the city proper (2023 estimate). The wider Paris metropolitan area has roughly 13 million.
[stop: end_turn, tokens in: 72, out: 100]
> You asked, "What is the capital of France?"
[stop: end_turn, tokens in: 184, out: 16]
> 
exit 0
```

All three answered, and Claude remembered the first question. The answers
are plain text now, with no `**`: that's the system prompt. The `> ` and
the answers share lines here because piped questions aren't shown. Only a
terminal shows what you type.

Look at `tokens in`. It grows with every turn, from 54 to 72 to 184,
because every request carries the whole conversation. Most of the jump to
184 is the second answer: its 100 output tokens, thinking included, are
now input, sent again with every request after it. A long chat gets slower and costlier
with every turn, and in the end it hits the model's size limit. Chapter
13 deals with that.

Now run it on its own and type. Press Ctrl-D on an empty line to finish.
Here's mine:

```sh
./goagent
```

```
> Name one moon of Jupiter.
Europa.
[stop: end_turn, tokens in: 58, out: 6]
> How big is it?
Europa has a diameter of about 3,122 km (1,940 miles), making it slightly smaller than Earth's Moon.
[stop: end_turn, tokens in: 71, out: 43]
> 
```

"It" in the second question is the moon from the first answer.

Since the prompts and token counts go to stderr, `2>/dev/null` throws them
away and leaves only the answers:

```sh
printf 'Name the three primary colours, comma separated.\n' | ./goagent 2>/dev/null
```

```
Red, yellow, blue
exit 0
```

## What this chapter doesn't do

- **The conversation is lost when goagent exits.** It lives in memory
  only. Saving it to a file is a small job, but nothing later needs it.
- **The history grows without limit.** Every turn costs more input
  tokens, and a long enough chat would hit the model's limit and fail.
  Chapter 13 trims it.
- **Errors aren't retried.** If the API is busy, you see the error and
  type the question again. Chapter 4 retries for you.
- **Ctrl-C kills goagent at once,** even halfway through a request.
  Chapter 12 makes it cancel cleanly.
- **One line is one question.** You can't paste a question with line
  breaks in it. And a line longer than 64 KB stops the scanner with an
  error (`bufio.Scanner`'s default limit).

## Commit

```sh
git add -A
git commit -m "Chapter 3: chat loop, system prompt, raw content blocks"
```

## What you should now be able to answer

**1. Why does every request carry the whole conversation?**

<details>
<summary>Answer</summary>

The API keeps nothing between requests. To Claude, each request is the
whole story, so anything it should remember has to be in it. That's also
why `tokens in` grows every turn.

</details>

**2. Where does the system prompt go, and how is it different from a
message?**

<details>
<summary>Answer</summary>

In the top-level `system` field, not in `messages`. It isn't a turn in the
conversation. It's standing instructions that apply to every answer, so
it's sent the same way with every request.

</details>

**3. Why does `ask` take a failed question back out of the history?**

<details>
<summary>Answer</summary>

Otherwise the next request would have two user turns in a row: the
unanswered question and the new one. The API would merge them into one
turn, and Claude would answer both at once. Taking it out keeps the
history going user, assistant, user.

</details>

**4. The fake kept `req.Messages`, and later found a different question
in it. Why?**

<details>
<summary>Answer</summary>

A slice points at an array, so the fake's slice and `run`'s `history`
shared one array. Shortening `history` didn't erase anything, and the next
`append` wrote the new question into the same spot. The fake keeps a
`slices.Clone` now, its own array.

</details>

**5. What would go wrong if `ContentBlock` had no `raw` field?**

<details>
<summary>Answer</summary>

Fields we have no Go field for would be dropped on the way back in. A
thinking block would lose its `thinking` and `signature`, and the API
rejects a modified thinking block with a 400. Keeping the raw bytes sends
every block back exactly as it came, including kinds we've never heard
of.

</details>

**6. Why does `UnmarshalJSON` decode into `(*plain)(b)` and not `b`?**

<details>
<summary>Answer</summary>

`json.Unmarshal(data, b)` would see that `b` has an `UnmarshalJSON` method
and call it: our own method, calling itself forever. `plain` has the same
fields and none of the methods, so decoding into it uses
`encoding/json`'s normal rules.

</details>

**7. The tests didn't change when we moved the history into `chat`. Why
is that a good sign?**

<details>
<summary>Answer</summary>

They test `run` from outside: text in, text out, requests sent. A split
that changes only the inside shouldn't change any of that, and the
unchanged tests passing shows it didn't.

</details>

## Next chapter

Run goagent enough and you'll see an answer like `API error 529
overloaded_error`, or `429 rate_limit_error` if you ask too fast. Today
goagent prints the error, and you type the question again. That's fine
for a chat, but in Chapter 6 goagent will run on its own for many turns,
and one busy moment would stop the whole job. In Chapter 4 goagent learns
which errors are worth trying again. It waits a little longer after each
failed try, and respects the `retry-after` header when the API sends it.
The `APIError` type from Chapter 2, with its `StatusCode`, is what makes
those decisions possible. And `httptest` lets us test all of this without
waiting for the real API to be busy.
