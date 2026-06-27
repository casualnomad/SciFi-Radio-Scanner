# SciFi-Radio-Scanner

A tiny Go web app that simulates scanning a sci-fi radio. You pick a channel —
**Civilian**, **Military**, or **Anomaly** — and a locally-hosted LLM generates a
short, in-character transmission. Each channel keeps its own conversation
history, so the chatter stays coherent across repeated scans.

This README is written to be the single source of truth for getting back into
the project cold. It covers how the pieces fit together, how to run it, the
known rough edges, and a prioritized list of next steps.

---

## Table of contents

- [What it does](#what-it-does)
- [Tech stack](#tech-stack)
- [Project layout](#project-layout)
- [How it's wired together](#how-its-wired-together)
  - [Request flow, end to end](#request-flow-end-to-end)
  - [Package responsibilities](#package-responsibilities)
  - [Data shapes](#data-shapes)
- [Prerequisites](#prerequisites)
- [Running it](#running-it)
- [Configuration](#configuration)
- [The LLM backend](#the-llm-backend)
- [API reference](#api-reference)
- [Known issues / rough edges](#known-issues--rough-edges)
- [Next steps (roadmap)](#next-steps-roadmap)
- [Glossary](#glossary)

---

## What it does

1. The browser loads a static page with three channel buttons.
2. Clicking a button calls `GET /next?channel=<name>`.
3. The server builds a prompt for that channel, sends the channel's full chat
   history to a local LLM, and gets back a generated transmission.
4. The transmission is appended to that channel's history (so context
   accumulates) and returned as JSON.
5. The page displays the text.

It's intentionally small — a working skeleton, not a finished product. See
[Next steps](#next-steps-roadmap) for what's missing.

---

## Tech stack

| Layer    | Choice                                                            |
| -------- | ----------------------------------------------------------------- |
| Language | Go **1.25.5** (see [go.mod](go.mod))                              |
| HTTP     | Standard library `net/http` only — no framework, no dependencies  |
| Frontend | Plain HTML + CSS + vanilla JS (`fetch`), no build step            |
| LLM      | Any **OpenAI-compatible** chat-completions server (local default) |

The module has **zero external dependencies** — `go.mod` lists only the module
name and Go version. Everything is standard library.

---

## Project layout

```
SciFi-Radio-Scanner/
├── main.go              # Entry point: wires deps together, starts HTTP server
├── go.mod               # Module definition (module "Radio-Scanner", go 1.25.5)
├── Radio-Scanner        # Pre-built binary (committed — see Known issues)
├── .gitignore
├── README.md            # You are here
│
├── llm/
│   └── client.go        # OpenAI-compatible chat client (the only thing that talks to the model)
│
├── radio/
│   ├── state.go         # WorldState, Channel type, ChatMessage, NewWorld()
│   └── engine.go        # Prompt construction: InitialPrompt() + BuildPrompt()
│
└── web/
    ├── server.go        # Server struct, /next handler, route registration
    └── static/
        ├── index.html   # UI markup: three buttons + output div
        ├── app.js       # Click handlers → fetch /next → render text
        └── style.css    # Green-on-black terminal aesthetic
```

---

## How it's wired together

### Request flow, end to end

```
Browser (app.js)
   │  click "Civilian" button
   │  fetch("/next?channel=civilian")
   ▼
web/server.go  →  Server.HandleNext
   │  1. read ?channel= query param
   │  2. if this channel has no history yet, seed it with a system prompt
   │        (radio.InitialPrompt)
   │  3. build a user prompt for the current world state
   │        (radio.BuildPrompt)  and append it to history
   │  4. convert radio.ChatMessage[] → llm.ChatMessage[]  (toLLMMessages)
   ▼
llm/client.go  →  Client.Chat
   │  POST {Url}/v1/chat/completions   (JSON body: model, messages, max_tokens)
   ▼
Local LLM server (e.g. http://localhost:2276)
   │  returns OpenAI-style { choices: [ { message: { content } } ] }
   ▼
web/server.go  (back in HandleNext)
   │  5. append the assistant reply to the channel's history
   │  6. respond with JSON  { "channel": "...", "text": "..." }
   ▼
Browser (app.js)
      renders  "[civilian] <text>"  into #output
```

### Package responsibilities

**`main` ([main.go](main.go))** — Composition root. It constructs the three
collaborators and starts the server. This is the one place where concrete
dependencies are wired:

```go
client := llm.New("http://localhost:2276", "local-model") // LLM client
world  := radio.NewWorld()                                // in-memory world state
server := web.New(client, world)                          // HTTP server holds both
server.Routes()                                           // register handlers
http.ListenAndServe(":3000", nil)                         // serve
```

Note the LLM URL, model name, and listen port are **hard-coded here** — that's
the single spot to change them today (see [Configuration](#configuration)).

**`llm` ([llm/client.go](llm/client.go))** — The only package that knows how to
talk to a model. It speaks the OpenAI `/v1/chat/completions` protocol:

- `New(url, model)` → builds a `Client`.
- `Chat(messages, maxTokens)` → marshals a `ChatRequest`, POSTs it, decodes the
  `ChatResponse`, and returns `choices[0].message.content`.

It is deliberately ignorant of channels, prompts, and world state. Swap this
package's internals and nothing else needs to change as long as `Chat` keeps its
signature.

**`radio` ([radio/state.go](radio/state.go), [radio/engine.go](radio/engine.go))**
— The domain / "game" logic. No HTTP, no networking.

- `state.go` defines the `Channel` enum, the `WorldState` (the simulated
  universe), and `ChatMessage`. `NewWorld()` returns a seeded starting state.
- `engine.go` turns a channel + world state into prompts:
  - `InitialPrompt(ch)` → the **system** message that sets the channel's persona.
  - `BuildPrompt(ch, state)` → the **user** message describing the current world,
    asking for the next transmission.

**`web` ([web/server.go](web/server.go))** — Transport layer. Holds the `Server`
struct (`Llm` + `State`), exposes:

- `Routes()` — mounts the static file server at `/` and `HandleNext` at `/next`.
- `HandleNext` — orchestrates the per-request flow described above.
- `toLLMMessages` — adapter that converts `radio.ChatMessage` →
  `llm.ChatMessage` (the two types are identical in shape but live in different
  packages to keep them decoupled).

### Data shapes

**`radio.WorldState`** — the in-memory simulated universe. Lives for the
lifetime of the process; **not persisted** anywhere.

```go
type WorldState struct {
    CivilianShips []string                  // e.g. ["Silver Needle", "Wren Courier-9"]
    MilitaryUnits []string                  // e.g. ["New World Patrol"]
    AnomaliesSeen int                        // counter
    History       map[Channel][]ChatMessage // per-channel conversation log
}
```

`History` is the important one: each channel accumulates `system` → `user` →
`assistant` → `user` → `assistant` ... and the **entire** slice is resent to the
LLM on every `/next` call. That's what gives each channel memory — and also why
the request payload grows unbounded over a long session (see Known issues).

**`llm.ChatRequest` / `ChatResponse`** — minimal subset of the OpenAI schema:

```go
ChatRequest{  Model, Messages []ChatMessage, MaxTokens }
ChatResponse{ Choices: [ { Message: { Content } } ] }
```

---

## Prerequisites

- **Go 1.25.5+** installed and on your `PATH` (`go version` to check).
- **A running OpenAI-compatible LLM server** reachable at the configured URL.
  By default that's `http://localhost:2276` exposing `POST /v1/chat/completions`.
  See [The LLM backend](#the-llm-backend) for options.

You do **not** need to fetch any Go dependencies — there are none.

---

## Running it

From the project root:

```bash
# 1. Make sure your local LLM server is up (see "The LLM backend" below).

# 2. Run directly from source:
go run .

# …or build a binary and run it:
go build -o Radio-Scanner .
./Radio-Scanner
```

You should see:

```
Scanner Running on Http://localhost:3000
```

Then open **http://localhost:3000** in a browser and click a channel.

> ⚠️ **Working directory matters.** The static files are served via
> `http.Dir("./web/static")`, a path **relative to where you launch the
> binary**. Run it from the project root, or static assets will 404. (Making
> this robust is on the roadmap.)

Quick smoke test without the browser:

```bash
curl "http://localhost:3000/next?channel=anomaly"
# → {"channel":"anomaly","text":"..."}
```

---

## Configuration

There is **no config file or env-var support yet**. All knobs are hard-coded in
[main.go](main.go):

| Setting       | Current value                | Where                          |
| ------------- | ---------------------------- | ------------------------------ |
| LLM base URL  | `http://localhost:2276`      | `main.go` → `llm.New(...)`     |
| Model name    | `local-model`                | `main.go` → `llm.New(...)`     |
| Listen port   | `:3000`                      | `main.go` → `http.ListenAndServe` |
| Max tokens    | `200`                        | `web/server.go` → `HandleNext` |
| Static dir    | `./web/static`               | `web/server.go` → `Routes`     |

Wiring these up to environment variables / flags is the first roadmap item.

---

## The LLM backend

The app expects an **OpenAI-compatible** server — anything that accepts
`POST /v1/chat/completions` with `{ model, messages, max_tokens }` and returns
`{ choices: [ { message: { content } } ] }`.

Common ways to get one locally on port `2276` (pick one, then point `main.go`
at it):

- **[LM Studio](https://lmstudio.ai/)** — start its local server; note the port
  it serves on and the model id it advertises.
- **[Ollama](https://ollama.com/)** — exposes an OpenAI-compatible endpoint at
  `/v1/chat/completions` (default port `11434`).
- **[llama.cpp server](https://github.com/ggml-org/llama.cpp)** —
  `llama-server` with `--port 2276`.
- **[vLLM](https://github.com/vllm-project/vllm)** or any other
  OpenAI-compatible host.

Whatever you choose, make sure the **URL, port, and model name in `main.go`
match** what your server actually exposes. The `model` string is passed through
verbatim; some servers ignore it, others require an exact match.

If the LLM server is down or the URL is wrong, `/next` will currently return a
`500` with the underlying error text (and, due to a bug, also try to write a
JSON body afterward — see below).

---

## API reference

### `GET /next?channel=<channel>`

Generates the next transmission for a channel.

- **Query param** `channel` — one of `civilian`, `military`, `anomaly`.
  - Any other value is rejected with `400 Bad Request`.
- **Response** `200 OK`, `application/json`:
  ```json
  { "channel": "civilian", "text": "<generated transmission>" }
  ```
- **Errors**: `400` for an unknown channel; `500` with the error text if the
  LLM call fails.

### `GET /` (and any other path)

Serves static files from `./web/static` (`index.html`, `app.js`, `style.css`).

---

## Known issues / rough edges

These are the things most likely to bite you (or future-you):

1. **The world never actually changes.** `AnomaliesSeen` stays `0`, and
   `CivilianShips` / `MilitaryUnits` never grow. `BuildPrompt` reads these, but
   nothing ever mutates them, so the "simulation" is static — only the chat
   history evolves.

2. **Unbounded history growth.** Every `/next` resends the channel's *entire*
   history to the LLM. Over a long session this blows past the context window
   and inflates latency/cost. Needs a sliding window or summarization.

3. **Config is hard-coded.** No env vars / flags (see Configuration).

4. **Relative static path.** `./web/static` breaks if you launch from anywhere
   but the project root. Consider `embed.FS` to bundle assets into the binary.

5. **The compiled `Radio-Scanner` binary is committed** (~8.9 MB). It's listed
   in `.gitignore` but was committed before being ignored. Consider
   `git rm --cached Radio-Scanner` so it stops riding along in the repo.

6. **No tests, no logging, no graceful shutdown.** `http.ListenAndServe`'s
   return value is also ignored, so a bind failure is silent.

> **Recently fixed:** the missing `return` after `http.Error` in `HandleNext`,
> the unguarded `WorldState` map (now behind `WorldState.Mu`), and unknown-channel
> validation (now `400`).

---

## Next steps (roadmap)

Roughly in priority order — each is a self-contained chunk of work:

**Make it correct**
- [x] Add `return` after `http.Error` in `HandleNext`.
- [x] Guard `WorldState` with a mutex.
- [x] Validate the `channel` param and 400 on unknown channels.

**Make it real (the world should evolve)**
- [ ] Mutate `WorldState` per transmission: increment `AnomaliesSeen` on the
      anomaly channel, occasionally add/retire ships and units. This is the
      feature that turns it from "chatbot with three personas" into a "scanner
      of a living world."
- [ ] Optionally parse structured info back out of the LLM reply (new ship
      names, events) to feed the world state.

**Make it usable**
- [ ] Move URL / model / port / max-tokens to env vars or flags (issue #5).
- [ ] Embed static assets with `embed.FS` so the binary runs from anywhere
      (issue #6).
- [ ] Trim/summarize history to bound context size (issue #4).
- [ ] Add a "scan" / auto-cycle mode that polls channels on a timer, and/or
      stream tokens to the UI (SSE or WebSocket) for a live-radio feel.

**Make it maintainable**
- [ ] `git rm --cached Radio-Scanner` (issue #7).
- [ ] Add unit tests for `BuildPrompt`/`InitialPrompt` and an httptest-based
      test for `HandleNext` with a fake LLM.
- [ ] Add structured logging and check `ListenAndServe`'s error.

---

## Glossary

- **Channel** — one of the three radio bands (`civilian` / `military` /
  `anomaly`), each with its own persona and conversation history.
- **WorldState** — the in-memory model of the simulated universe (ships, units,
  anomaly count, per-channel history). Not persisted; resets on restart.
- **Transmission** — a single LLM-generated message returned by `/next`.
- **System / user / assistant** — standard chat roles. The system prompt sets a
  channel's persona; user prompts request the next transmission; assistant
  messages are the model's replies, kept in history for continuity.
</content>
