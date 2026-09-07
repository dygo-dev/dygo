# Server

`dygo serve` starts the local dygo HTTP server.

The server address comes from:

```txt
dygo.yml
```

The default address is:

```txt
127.0.0.1:6790
```

`dygo dev` loads the development database credentials by default and starts the local development experience:

```sh
dygo dev
```

In a source checkout with `apps/studio/ui/package.json`, `dygo dev` starts Studio's development asset server internally and proxies Studio pages through dygo. The browser-facing address stays `http://127.0.0.1:6790/`, so Studio and `/api/v1/...` share one origin during development.

Generated projects serve Studio from `.dygo/apps/studio/ui/dist` when that cache exists. Release builds also include bundled Studio assets, and `dygo new` / `dygo upgrade` refresh the generated-project cache when the running dygo binary has those assets.

`dygo serve` starts the runtime server and uses generated-project or bundled Studio assets. If no generated-project cache or bundled release assets are available, `dygo serve` exits before listening instead of serving an API-only site.

Use another encrypted environment with `--env`:

```sh
dygo serve --env staging
```

Use `dygo dev --studio-dev-url` only when the Studio asset server is already running somewhere else:

```sh
dygo dev --studio-dev-url http://127.0.0.1:6791
```

The server opens and pings PostgreSQL before it starts listening. It does not run `dygo db migrate` automatically; run metadata sync before serving runtime metadata.

## Production Processes

Production deployments that use Jobs or Schedules need two long-running dygo processes:

```txt
web: dygo serve
worker: dygo worker
```

`dygo serve` handles HTTP, Studio, auth, metadata, and Record APIs. It does not claim or run queued Job Executions.

`dygo worker` checks due Schedules, creates queued Job Executions, claims queued Job Executions from PostgreSQL, and runs compiled Job handlers. If the worker is not running, new Job Executions remain queued and due Schedules wait in the database until a worker starts.

Deployment tools own process supervision, restarts, scaling, and log collection. dygo only defines the commands those tools should run.

## Health

The first server surface is:

```txt
GET /health
```

It returns:

```txt
ok
```

This endpoint is intentionally small. It only confirms that the HTTP process is accepting requests.

## Auth API

Studio-oriented auth uses an HttpOnly `dygo_session` cookie:

```txt
POST /api/v1/auth/login
POST /api/v1/auth/logout
GET  /api/v1/auth/me
GET  /api/v1/boot
```

`POST /api/v1/auth/login` is public. Boot, metadata, and Record API routes require a valid session. Metadata and Record APIs use the same permission engine. Metadata list endpoints filter unreadable Entities; Record API routes require the relevant Entity action.

Authenticated Studio notification routes are:

```txt
GET  /api/v1/notifications?limit=20
GET  /api/v1/notifications/unread-count
POST /api/v1/notifications/{id}/read
GET  /api/v1/notifications/{id}/deep-link
```

These routes always scope data to the current Core User. Deep links are local Studio paths; external URLs are rejected.

## Studio State API

Authenticated Studio state routes use the current session's User:

```txt
GET    /api/v1/studio/preferences
PUT    /api/v1/studio/preferences/{key}
DELETE /api/v1/studio/preferences/{key}
GET    /api/v1/studio/saved-filters?entity=crm/contact
POST   /api/v1/studio/saved-filters
PATCH  /api/v1/studio/saved-filters/{id}
DELETE /api/v1/studio/saved-filters/{id}
```

Preference reads return a key/value map in the `data` envelope. Write one key with `{"value":...}`. Keys are dot-separated namespaces, such as `studio.theme` or `studio.records.crm.contact.hidden-columns`. URL-encode the key. Concurrent first writes use the unique User/key constraint and a bounded retry.

Create a saved filter with `{"entity":"crm/contact","label":"Active","filters":[{"field":"enabled","operator":"eq","value":"true"}]}`. Update its `label`, `filters`, or both. Responses contain `id`, canonical `entity`, `label`, and `filters`. Labels are unique within the User and target Entity. An incompatible saved filter includes `validationError` on list reads so its owner can replace or delete it.

Request bodies cannot set ownership. Saved-filter requests require target Entity read access and validate predicates against current metadata and Field access. Studio Preference and Saved Filter storage are private: generic Record routes cannot expose them, including through Administrator access. These state changes do not copy private values into the generic Activity feed. Trusted system SDK code remains privileged.

## Tree Record API

Tree Entities add authenticated GET routes under `/api/v1/records/{entity}/tree/`:

| Route | Result |
| --- | --- |
| `roots` | Root Records |
| `children?name=...` | Direct children |
| `descendants?name=...` | Descendants |
| `ancestors?name=...` | Root-to-parent chain |
| `path?name=...` | Root-to-Record chain |
| `search` | Filtered matches with readable ancestor context |

URL-encode anchor Record names. Paged routes accept existing Record filters, `limit`, `offset`, and `sort`. Search also accepts `exclude-subtree` with an anchor Record name to omit that node and its descendants.

Each `data` item contains `record`, `hasChildren`, `matched`, and `pathUnavailable`. Search returns paginated matches plus ordered display `context`, which can include matching Records to preserve sibling order. Deduplicate by Record name. All returned data respects Record and Field access. Complete paths fail when an ancestor is inaccessible. Tree reads do not grant access through ancestry.

Traversal rejects paths deeper than 2,500 Records or batches above 10,000 path steps. Narrow the search or reduce the page size when this limit is reached; paths are never silently truncated.

Use the existing Record create and update endpoints to set the parent Link. Use null to create a root. There is no separate HTTP mutation path for trees. Deleting a Record with children returns a conflict without exposing child details.

## Metadata API

The first runtime API is read-only and powered by persisted Core metadata records:

```txt
GET /api/v1/apps
GET /api/v1/apps/{app}
GET /api/v1/entities
GET /api/v1/entities/{entity}/meta
```

Responses use stable JSON envelopes:

```json
{"data":[]}
```

Errors use:

```json
{"error":{"code":"not_found","message":"entity not found","details":{"entity":"lead"}}}
```

These endpoints are generic. dygo does not create per-Entity routes such as `/api/users` or `/api/leads`.

`{entity}` in metadata and Record routes is the Entity slug.

Metadata visibility is permission-aware. `GET /api/v1/entities` returns only Entities the current user can read, while `GET /api/v1/entities/{entity}/meta` returns `403 forbidden` for a known Entity the user cannot read. App metadata is visible when the user can read Core `app` metadata or at least one Entity owned by that App.

## Record API

The first Record API is also generic and metadata-powered:

```txt
GET    /api/v1/records/{entity}?limit=50&offset=0&status:eq=Open&sort=-created-at,name
GET    /api/v1/records/{entity}/{id}
GET    /api/v1/records/{entity}/name/{name}
GET    /api/v1/records/{entity}/single
GET    /api/v1/records/{entity}/{id}/activity?limit=50&offset=0
GET    /api/v1/records/{entity}/export
POST   /api/v1/records/{entity}
POST   /api/v1/records/{entity}/actions/{action}
POST   /api/v1/records/{entity}/{id}/activity
PATCH  /api/v1/records/{entity}/{id}
PATCH  /api/v1/records/{entity}/single
DELETE /api/v1/records/{entity}/{id}
```

Record APIs read persisted Core metadata to map Entity slugs, Field names, and storage columns. `{entity}` is the slug, defaulting to the file-derived Entity key. Run `dygo db migrate` before serving Records so metadata tables and Entity storage tables are in sync.

`GET /api/v1/records/{entity}/name/{name}` returns one Record by system `name`; URL-encode `{name}` as a path segment.

For Single Entities, use `GET /api/v1/records/{entity}/single` and `PATCH /api/v1/records/{entity}/single`. Normal list, create, and delete operations are not valid for Single Entities.

Record request bodies use a `data` envelope:

```json
{"data":{"email":"a@example.com","full-name":"A User"}}
```

Record responses use dygo metadata names, including system fields:

```json
{"data":{"id":1,"name":"a@example.com","created-at":"2026-05-07T12:00:00Z","updated-at":"2026-05-07T12:00:00Z","email":"a@example.com"}}
```

Write-only fields such as `password` are accepted in create and update requests, but are not returned in responses.

List responses include pagination metadata:

```json
{"data":[],"meta":{"limit":50,"offset":0,"count":0}}
```

Record lists support direct `field:operator=value` params and multi-field sorting through `sort`. Empty checks use `field:empty` or `field:not-empty`; range checks use `start..end` as the value. `limit`, `offset`, and `sort` are reserved query params. Sorting uses `-field` for descending order and appends `id ASC` as a deterministic tie-breaker.

The runtime compiles conditional `when` policies into the list, direct Record,
mutation, action, Activity, file, and export queries. Denied fields do not
appear in Record or CSV output.

`PATCH` is the update operation and only changes fields provided in the request body. `DELETE` performs a hard delete in v1.

Scoped Record Activity is read through `GET /api/v1/records/{entity}/{id}/activity`. It returns newest-first Activity only when the actor can read the live target Record. Use `POST` on the same path to add an append-only comment when the actor can update the target.

Private files and durable CSV imports use these authenticated routes:

```txt
POST   /api/v1/files
GET    /api/v1/files/{id}
DELETE /api/v1/files/{id}
POST   /api/v1/imports
GET    /api/v1/imports/{id}
```

File requests include a target App, Entity, Record ID, and field. Import
requests accept CSV multipart data and return a Core Import ID for progress
and row-error polling.

Record API permissions are checked through the single internal permission engine:

```txt
GET list/detail -> read
GET activity -> read
POST -> create
PATCH -> update
DELETE -> delete
```

Administrator users are allowed by the engine before flat role permissions are checked.

## Shutdown

Stop the server with `Ctrl-C`.

The CLI listens for interrupt and termination signals, asks the HTTP server to shut down cleanly, and stops the auto-started Studio dev server when one was started by `dygo dev`.

## Boundaries

The current server includes health, session auth, metadata APIs, scoped Record CRUD and Entity actions, Activity, private files, notifications, CSV import/export, static Studio serving, and the development proxy. It does not include per-Entity controllers or arbitrary permission SQL.
