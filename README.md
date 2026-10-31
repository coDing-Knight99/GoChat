# GoChat

GoChat is a chat application backend written in Go. It uses HTTP handlers for account and conversation operations, WebSockets for live message exchange, MongoDB for user and message documents, and Redis for queued message persistence. The hub also recognizes the reserved recipient `GoIntellisense` and sends those messages to the Google GenAI client configured in the code.

This README describes behavior present in this repository. The React frontend source was supplied separately in the conversation; frontend details below are limited to that supplied `ChatPage` component.

## Architecture

```text
React ChatPage
   | HTTP requests with cookies       | WebSocket /ws with accessToken cookie
   v                                  v
Go HTTP mux -> CORS middleware -> JWT middleware (protected routes)
   |                                  |
   +--> user / conversation handlers  +--> WebSocket upgrader -> Hub <--> Clients
                 |                                      |
                 v                                      +--> Redis list `chat_msgs`
             MongoDB                                               |
                                                                    v
                                                           DB worker -> MongoDB
```

### Components

- `main.go` loads `.env` with `godotenv.Load`, connects to MongoDB and Redis, creates a hub and database worker, and registers HTTP routes. The MongoDB database name is `test`; the collections are `Users` and `Message`.
- `controllers/` contains account, WebSocket, conversation-summary, and message-history handlers.
- `middleware/verifyJWT.go` reads the `accessToken` cookie, validates it, and puts the token's user ID in the request context for protected handlers.
- `middleware/cors.go` adds CORS response headers using `FrontEnd_URL` and returns HTTP 200 for `OPTIONS` requests.
- `auth/token.go` creates and validates HMAC JWT access and refresh tokens using `AccessTokenSecret` and `refreshTokenSecret`.
- `models/` defines users, messages, connected WebSocket clients, and the in-memory hub.
- `workers/dbworker.go` moves serialized messages from a Redis queue into MongoDB in batches.
- `db/` contains MongoDB and Redis connection setup.
- `docker-compose.yaml` defines a Redis service with append-only persistence and a named Docker volume. It does not define the Go application or MongoDB service.

## Configuration

The code reads these environment variables:

| Variable | Use in code |
| --- | --- |
| `PORT` | Address passed to `http.ListenAndServe`. |
| `MONGODB_URI` | MongoDB connection URI. |
| `Environment` | Selects production Redis URL behavior and cookie attributes. Exact value comparisons are used. |
| `Redis_url` | Redis URL when `Environment` equals `Production`. |
| `Redis_Client` | Redis address when not in production mode. |
| `Redis_Username` | Redis username in non-production mode. |
| `Redis_Password` | Redis password in non-production mode. |
| `FrontEnd_URL` | Exact allowed WebSocket origin and CORS `Access-Control-Allow-Origin` value. |
| `AccessTokenSecret` | Signing and validation secret for access JWTs. |
| `refreshTokenSecret` | Signing and validation secret for refresh JWTs. |
| `GEMINI_API_KEY` | API key passed to the Google GenAI client for `GoIntellisense` requests. |

`main.go` calls `godotenv.Load()` without checking its returned error. Environment values may also be supplied by the process environment. The JWT secret byte slices are initialized at package load in `auth/token.go`, before `main()` calls `godotenv.Load()`; therefore, values supplied only by that `.env` load are not read into those already initialized variables.

The repository's `.gitignore` contains `.env`. Do not put real credentials in tracked documentation or source files.

## Application workflows

### Startup

1. `main()` loads the `.env` file if available.
2. `db.ConnectDB()` reads `MONGODB_URI`, connects, and pings MongoDB. If it returns `nil`, the application logs a connection failure and returns without registering routes or starting the HTTP server.
3. The application selects MongoDB database `test` and the `Message` and `Users` collections.
4. `db.ConnectRedis()` creates a Redis client. When `Environment` is exactly `Production`, it parses `Redis_url` and pings Redis; otherwise it uses `Redis_Client`, `Redis_Username`, and `Redis_Password`.
5. The application starts `Hub.StartHub()` and `DBWorker.Start()` as goroutines, registers routes on an `http.ServeMux`, wraps the mux with CORS middleware, and calls `http.ListenAndServe` with `PORT`.

### Registration and login

- `POST /register` decodes a `models.User` JSON body containing `username`, `email`, and `password`. It looks up the username in `Users`, hashes the password with bcrypt, and inserts the user document. It returns conflict when the lookup finds an existing username, created after insertion, and internal server error on the handled hash or insert errors.
- `POST /login` decodes `username` and `password`, looks up the user, compares the password with the stored bcrypt hash, and creates an access token and refresh token. It sets both as HttpOnly cookies. In `Development`, cookies use `SameSite=Lax`; otherwise they use `SameSite=None`. The `Secure` attribute is true only when `Environment` is exactly `Production`.
- `GET /LoginStatus` checks the access-token cookie and returns JSON with `isLogin` set to true or false.
- `GET /Logout` is protected by access-token middleware and writes expired/cleared cookie values. The handler sets `MaxAge: -1` for `accessToken`; the `refreshToken` cookie is written with a positive `MaxAge` of one day.

The route handlers are registered with `HandleFunc`; the code shown does not add explicit HTTP-method checks inside the mux registration.

### Protected HTTP routes

`/ws`, `/conversations`, `/messages`, `/users/search`, and `/Logout` are wrapped with `VerifyToken`. The middleware requires a valid `accessToken` cookie. It places the token's `userId` claim into the request context for the handler.

- `POST /checkavailable` decodes a username and queries `Users`; it returns an `available` string value of `true` or `false` when the lookup resolves normally.
- `POST /users/search` decodes a username prefix, escapes it for a case-insensitive MongoDB regular expression, limits results to ten, and returns user IDs and usernames.
- `GET /conversations` aggregates the authenticated user's messages by the other participant and returns each conversation's latest message, timestamp, and sender ID.
- `GET /messages?contact=<id>&page=<number>` retrieves messages exchanged between the authenticated user and the contact. It sorts newest first for retrieval, uses pages of 50 records, checks whether another record exists, then reverses the returned page into chronological order. Invalid or missing page values default to page 1.

### WebSocket connection and direct messages

1. The supplied React component reads `VITE_API_BASE_URL`, changes an initial `http` prefix to `ws`, and connects to `<converted-base-url>/ws`.
2. The browser sends the `accessToken` cookie with the WebSocket handshake when cookie policy permits. The `/ws` route first validates that cookie with `VerifyToken`.
3. The Gorilla WebSocket upgrader accepts the handshake only when the request's `Origin` header exactly equals `FrontEnd_URL`.
4. After upgrade, the handler creates a client with the authenticated user ID, registers it with the hub, starts its write pump, and reads JSON messages in the request goroutine.
5. For each received message, `ReadPump` sets `sender_id` from the authenticated client and sets the current timestamp, then sends the message to the hub's broadcast channel. The incoming `receiver_id` is taken from the message payload.
6. For a normal recipient, the hub sends the message to that recipient's `Send` channel if the recipient has an active connection. The hub queues the message for persistence whether or not the recipient is online.
7. The connected recipient's write pump serializes queued messages as JSON over the WebSocket. The supplied React component parses incoming JSON and updates its contact list and displayed message cache.

The hub's client map is keyed by user ID, so the code stores one active client entry per user ID. The provided code does not show offline WebSocket delivery; queued Redis messages are for database persistence.

### Message persistence

- `Hub.SaveToDB` JSON-encodes a message and pushes it to the Redis list `chat_msgs`.
- `DBWorker.Start` claims items by moving them from `chat_msgs` to `chat_msgs:processing` using `LMOVE`.
- The worker collects up to 100 messages or flushes its in-memory buffer on its four-second ticker. A batch is inserted into the MongoDB `Message` collection.
- After a successful insert, the worker removes each corresponding serialized message from `chat_msgs:processing`.
- At worker startup, it moves items left in `chat_msgs:processing` back to `chat_msgs` for retry. Invalid JSON is moved to `chat_msgs:dead_letter`.

MongoDB history and conversation endpoints read persisted documents. The supplied frontend merges fetched history with its local per-contact message cache, which its comments describe as retaining live messages that have not yet appeared in database history.

### GoIntellisense flow

When a message's `receiver_id` is `GoIntellisense`, the hub starts `APICall` and queues the original message for persistence. `APICall` creates a Google GenAI client using `GEMINI_API_KEY`, calls model `gemini-3.5-flash-lite` with the message text and the appended instruction `Answer in only text format`, then broadcasts a response whose `sender_id` is `GoIntellisense` and whose `receiver_id` is the original sender. The frontend component includes a temporary waiting message for this contact and replaces it when it receives a reply.

## HTTP routes

| Route | Authentication in mux | Behavior |
| --- | --- | --- |
| `/register` | No | Register a user. |
| `/login` | No | Verify credentials and set token cookies. |
| `/checkavailable` | No | Check username availability. |
| `/LoginStatus` | No | Report access-token login status. |
| `/Logout` | Yes | Clear token cookies (see workflow details above). |
| `/ws` | Yes, then WebSocket origin check | Upgrade connection and handle live messages. |
| `/conversations` | Yes | Return conversation summaries for the authenticated user. |
| `/messages` | Yes | Return a page of history for a contact. |
| `/users/search` | Yes | Search usernames by prefix. |

## Running the backend

The module declares Go version `1.26.4` in `go.mod`. Provide the environment variables listed above, then run:

```sh
go run .
```

To start the Redis service defined by the repository's Compose file:

```sh
docker compose up -d redis
```

The Compose file exposes Redis port `6379` and stores its data in the named volume `redisGoChat_data`. MongoDB is not defined in that Compose file; the application connects to the URI supplied through `MONGODB_URI`.
