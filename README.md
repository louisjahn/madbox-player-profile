# Player Profile API — Home Assignment Scaffold

Minimal starter for the Madbox Senior Backend home assignment. Only the
truly boilerplate wiring is provided; everything else is part of what we
evaluate.

## What's wired for you

- `main.go` — dials Mongo via `internal/mongo.Connect` on startup and
  starts an HTTP server on `HTTP_ADDR`; registers a single example
  `GET /healthz` handler that pings Mongo
- `internal/mongo/client.go` — a small helper that dials Mongo and pings
- `main_test.go` — a `TestMain` that boots a real Mongo via
  `testcontainers-go`, connects, wires `newMux` into an `httptest` server,
  and exposes `testMongo` and `testServer` as package globals for your
  tests. Includes a minimal `TestHealthz` to show the pattern; add your
  own `Test…` functions alongside it.

## What's up to you

Everything else — business handlers, schema, indexes, graceful shutdown,
middleware, structured logging, tracing, concurrency, Dockerfile/compose,
Makefile, lint config, and your broader test strategy — is part of the
assignment. There is no "right" layout; we evaluate the choices you make.

The `/healthz` endpoint and the `internal/mongo` package are reference
examples, not requirements — delete or rework either if your design wants
something else.

## Run

```bash
go run .
# or with a custom port:
HTTP_ADDR=:9000 go run .
```

The server dials Mongo on startup at `MONGO_URI` (default
`mongodb://localhost:27017`). Bring up Mongo however you like — a local
install, `docker run mongo:7`, a compose file you write, etc. A quick
option:

```bash
docker run -d --rm -p 27017:27017 --name scaffold-mongo mongo:7
```

## Test

```bash
go test ./...
```

Integration tests require Docker (testcontainers will pull `mongo:7` on
first run).

## Configuration

| Var | Default |
|---|---|
| `HTTP_ADDR` | `:8080` |
| `MONGO_URI` | `mongodb://localhost:27017` |

## Notes

- AI usage is encouraged. Keep brief notes on what you used it for and
  include them at the end of this README before submission.

## Implementation

| Method | Path | Purpose |
|---|---|---|
| `PUT` | `/v1/players/{id}` | Idempotent create. `201` on first play, `200` on returning player |
| `GET` | `/v1/players/{id}` | Fetch profile |
| `PATCH` | `/v1/players/{id}` | Update mutable fields (`display_name`, `country`) |
| `POST` | `/v1/players/{id}/events` | Append a gameplay event |
| `GET` | `/v1/players/{id}/events` | Recent events, newest first |
| `GET` | `/v1/players/{id}/session` | Session summary: profile + recent events + counts by type |

## Approach and design choices
PUT /players/{id} is an idempotent create. The ID comes from the client (device or auth ID), not the server, so there's no create-then-read round trip on a cold start. The status code carries the useful bit: 201 means first play, 200 means returning player. The game client can branch on that without a second call. A POST /players returning a server-generated ID would force the client to persist a mapping it doesn't need.
Events are append-only in their own collection. The obvious alternative is embedding them in the profile document. I didn't, for three reasons: unbounded array growth, the 16 MB document limit, and write contention on a single hot document during an active session. A separate collection also lets the event write path stay cheap — it's the highest-volume endpoint in the service.
One compound index, {player_id: 1, ts: -1}, serves both reads. The recent-events list is a bounded range scan in index order (no in-memory sort), and the windowed count aggregation matches the same prefix, so it's a range scan rather than a collection scan.
The session summary is a read model, not a stored document. It composes three reads issued in parallel via errgroup. The 404 is driven by the profile read alone: a player with no events is a valid new player and gets empty collections back, not an error.
Lists are wrapped, counts are a map. Events come back as {"events": [...]} rather than a bare array, so cursor pagination can be added later without a breaking change. Counts by type are a map\[string\]int64, so a new event type needs no schema or API change — and the repository returns an initialised empty map so the JSON is {} and never null.
Layering. Each feature package (player, event) owns its domain type, a repository interface, and a service. The Mongo implementation lives in a mongorepo subpackage and is the only code that imports the driver. The service depends on the interface, so the HTTP and business layers are testable without a database, and swapping the store doesn't ripple upward. Handlers do transport concerns only: decode, validate shape, call the service, map domain errors to status codes.
Timestamps. Event timestamps are supplied by the client, because games are played offline and events are flushed later. They're stored as given, normalised to UTC. Trusting client clocks is a deliberate tradeoff; see below.
Testing. Unit tests cover validation and error mapping with fakes. Integration tests hit a real MongoDB and only assert what a fake couldn't prove: index-backed ordering, limit behaviour, filter correctness, and the boundary of the 7-day count window. One end-to-end test walks the full session lifecycle through the HTTP surface.

## What I'd do with another day
Cursor pagination on the events list (?after=\<ts\>&limit=), which the wrapped response shape already accommodates.
A received_at server timestamp on every event alongside the client ts, so clock skew and offline flushes are diagnosable and analytics has a trustworthy ordering key.
Retention policy on events: a TTL index or a scheduled archival job to cold storage. Right now the collection grows forever.
Per-player rate limiting on the event endpoint — it's the one an abusive or buggy client can hammer.
OpenTelemetry tracing, with spans around the three parallel reads in the session summary. That's the endpoint whose latency is hardest to reason about from logs alone.
A schema registry per event type, validating payloads against a per-type schema instead of accepting arbitrary JSON. Free-form payloads are right for shipping fast and wrong for a year from now.
Move index creation into an explicit migration step. It currently runs at startup, which is fine at this size but doesn't belong in the serving path of a real deployment.
Structured request logging and a /healthz readiness probe that actually pings MongoDB.
Load-shedding and timeouts tuned per endpoint rather than one global server timeout.


## AI usage notes
I used Claude (Anthropic) extensively throughout this assignment, as the brief encourages. Specifically:

Architecture and terminology. My background is C++ game servers with custom internal networking; I've not built a Go HTTP service from scratch before. I asked Claude to explain the idiomatic Go layering vocabulary — domain package, repository interface, service layer — and to justify why each boundary exists rather than just asserting it. I then implemented the first endpoint (player creation) myself against that explanation, to make sure I understood the pattern before repeating it.
Code generation. Substantial portions of the repository implementations, the testServer HTTP test helpers, and this README were drafted by Claude and then reviewed and adjusted by me.
Debugging. I used it as a rubber duck on a 400 invalid body failure in the event-creation test; its suggestion to surface the underlying decode error rather than swallow it into a generic message is now reflected in the handler's error handling.
Review. I asked for PR-style critique of handler and test structure and acted on parts of it.

What I did myself: ran every test and query against a real MongoDB, chose the data model and index strategy after being presented with the tradeoffs, and made the final call on the API shape. Where Claude's suggestions conflicted with what the assignment asked for or with my own judgement, I overrode them.
I'd rather be specific here than claim more independence than is true — the assignment explicitly encourages AI use, and how someone works with these tools seems more interesting than whether they do.
