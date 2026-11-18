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
