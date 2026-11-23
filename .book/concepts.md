# Concepts ledger

The reader writes all code (AGENTS.md). "First explained" = where the
concept got its full explanation in a "Your turn" box. Later uses get a
one-line reminder.

| Concept | First explained | Later uses |
|---|---|---|
| Messages API request (url, headers, model/max_tokens/messages) | Ch1 by hand + Step 2–3 | |
| Reply shape: content blocks list, stop_reason, usage | Ch1 by hand + Step 4 | |
| API error shape (`type: error`, `error.type`, message) | Ch1 "When it goes wrong" | |
| JSON struct tags, json.Marshal / Unmarshal, unknown fields skipped | Ch1 Step 2, Step 4 | |
| Anonymous (in-place) struct field | Ch1 Step 4 | |
| os.Args, strings.Join | Ch1 Step 2 | |
| stdout vs stderr, exit codes, os.Exit | Ch1 Step 2 | |
| variadic `args ...any` and `args...` | Ch1 Step 2 (fail) | |
| os.Getenv, env vars, keys out of code | Ch1 Step 3 | |
| http.NewRequestWithContext, Header.Set, Client{Timeout}, Do | Ch1 Step 3 | |
| context.Background (brief) | Ch1 Step 3 | Ch12 full |
| defer resp.Body.Close | Ch1 Step 3 | |
| Client.Do: no error on non-2xx, check StatusCode | Ch1 Step 3 + Try it | |
| Makefile check/build, .PHONY, GO := $(shell which go) | Ch1 Step 1 | |
| DRY (named at fail) | Ch1 Step 2 | |
| zsh glob on unquoted `?` | Ch1 Step 2 | |
| Go packages, import path = module + folder, exported = capital letter | Ch2 Step 1 | |
| Content as list of blocks both directions (string = shorthand) | Ch2 Step 1 | Ch3, Ch6 |
| `omitempty` JSON tag | Ch2 Step 1 | |
| Methods, pointer receivers | Ch2 Step 1 (+ fakeSender note) | |
| error interface, own error type (APIError), errors.As | Ch2 Step 2, Q4 | Ch4 |
| Anonymous struct as a local variable type | Ch2 Step 2 | |
| %d vs %s, %!s(int=...) | Ch2 Step 2 | |
| Dependency injection (NewClient params) | Ch2 Step 3 | |
| fmt.Errorf %w, packages return errors not exit | Ch2 Step 3 | |
| httptest.NewServer / URL / Client / Close, HandlerFunc | Ch2 Step 4 | |
| _test.go, TestXxx, testify require vs assert, JSONEq, ErrorAs, EqualError | Ch2 Step 4/5 | |
| assert only (not require) inside handler goroutine | Ch2 Step 4 | |
| t.Cleanup, test helper (DRY), arrange-act-assert | Ch2 Step 4 | |
| go mod tidy (after writing imports), go.sum | Ch2 Step 4 | |
| Break a test on purpose to see it fail | Ch2 Step 4 | |
| Interfaces satisfied implicitly; dependency inversion (interface lives with user) | Ch2 Step 5 | Ch6, Ch7, Ch15 |
| io.Writer, bytes.Buffer for capturing output | Ch2 Step 5 | |
| run(...) error pattern; main only wires + exits | Ch2 Step 5 | |
| API is stateless: send the whole history every request; tokens in grow | Ch3 intro + Try it | Ch6, Ch13 |
| System prompt (top-level `system`, not a message) | Ch3 Step 1 | |
| stdin, io.Reader, strings.NewReader, bufio.Scanner (Scan/Text/Err), Ctrl-D | Ch3 Step 2 | |
| Roll back a failed turn; consecutive same-role turns get merged | Ch3 Step 2 | Ch6 |
| Prompts/stats on stderr, answers on stdout (2>/dev/null) | Ch3 Step 2 + Try it | |
| Slices share arrays (length/capacity); slices.Clone when keeping one | Ch3 Step 2 | Ch3 Step 4 |
| Fake with scripted answers + recorded requests | Ch3 Step 2 | Ch4, Ch6 |
| Single responsibility (named at chat vs run split); tests survive the split | Ch3 Step 3 | |
| Thinking blocks (adaptive, on by default, display omitted, signature) | Ch3 Step 4 | Ch6 |
| json.Marshaler / Unmarshaler, json.RawMessage, `type plain T` trick, copy data | Ch3 Step 4 | |
| Write the failing test first for a flaky bug | Ch3 Step 4 | |
