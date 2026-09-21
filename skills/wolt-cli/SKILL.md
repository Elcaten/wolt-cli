---
name: wolt-cli
description: Browse Wolt venues, menus, items, and run profile, cart, and checkout-preview actions via the local `wolt` CLI or `wolt-mcp` tools (`wolt_feed`, `wolt_top`, `wolt_cart_add`, …), including dockerized Streamable HTTP. Trigger when asked to find food on Wolt, inspect venue catalogs, resolve item/option IDs, automate basket or profile tasks, or debug Wolt auth/location/output behavior.
---

# Wolt CLI + MCP

Tool repository: https://github.com/Elcaten/wolt-cli

Stdio MCP and HTTP MCP (Compose / `docker run`) expose the **same** tools.
Docker is an operator deployment, not a third API. Do not curl `/mcp` or
`/healthz`. Do not hardcode `mcp_<server>_…` names or a hostname.

## Session Startup

1. Scan available tools for names whose suffix is a canonical `wolt_*` tool
   (`wolt_feed`, `wolt_top`, `wolt_cart_add`, …). Hosts prefix them
   (Cursor: `mcp_<serverKey>_wolt_feed`). Match by suffix; ignore the server
   key. If any match, **use MCP** for the rest of the session. If that server
   is configured with a `url` (Streamable HTTP / Docker), you are on HTTP
   transport — see Auth.
2. Else if the `wolt` binary is on PATH, use the CLI:
   ```bash
   wolt <group> <command> [flags] --format json
   ```
   Parse `.data`; surface `.warnings` / `.error`.
3. Else stop: neither transport is wired. Point the user at the repo README
   (Path A = CLI, Path B = MCP stdio or Docker HTTP).

Canonical names, CLI equivalents, and MCP-only gaps:
`references/mcp-tools.md`.

## Safety Rules

- Start read-only by default.
- Request explicit confirmation before mutating:
  - MCP: `wolt_cart_add`, `wolt_cart_remove`, `wolt_cart_clear`,
    `wolt_favorites_add`, `wolt_favorites_remove`
  - CLI: `cart add/remove/clear`, `account favorites add/remove`,
    `account addresses add/update/remove/use`, and `login` (CLI/stdio
    only — never on HTTP MCP)
- Never describe checkout preview as order placement. Nothing here places
  a final order.

## Auth Workflow

Treat HTTP MCP and local CLI/stdio differently. Never `docker exec` login.

**HTTP MCP** (server configured with a `url`, including Docker Compose /
`docker run` on `/mcp`): do **not** run `wolt login`, open a browser, or
pass tokens into the container yourself. There is no login tool. If a call
fails with `AUTH_REQUIRED`, `AUTH_EXPIRED`, `SESSION_REFRESH_FAILED`, or
missing credentials, stop and ask the user to check the mounted
`~/.wolt/.wolt-config.json` (or `WOLT_CONFIG_PATH`) and to provide
credentials (`wtoken` + `wrtoken`, or `__wtoken` / `__wrtoken` cookies).
Retry only after they confirm the session is in place.

**CLI or local stdio MCP** (`wolt` / `wolt-mcp` as a local `command`): login
is a host-side operator action. Confirm before running it.

```bash
wolt login                                      # browser-driven (managed Chrome at 127.0.0.1:9222)
wolt login --wtoken "<jwt>" --wrtoken "<rt>"    # manual tokens
wolt status --format json --verbose             # CLI probe; MCP: wolt_account_status
```

`wolt login` (no flags) opens the Wolt login page in managed Chrome and polls
the Chrome DevTools cookie store every ~1.5s until a real `__wtoken` cookie
is set. Default timeout is 2 minutes.

When refresh credentials are available, missing, expired, or rejected access
tokens are refreshed automatically. The refreshed access token is saved only
when the complete persisted credential snapshot is unchanged. Any refresh
token returned by the rotation endpoint remains process-local; the saved
bootstrap refresh token and cookies stay pinned.

## Location Rules

Apply exactly:

- MCP: pass `address` **or** both `lat` + `lon`. CLI: `--address` **or**
  both `--lat` + `--lon`. Never combine address with coordinates.
- If no override is passed, current account location is used.
- CLI `venues` / `venue` use `--address` or account location (no direct
  `--lat/--lon` on those commands). MCP venue tools accept `lat`/`lon` or
  `address` per their schemas.
- `venue hours` / `wolt_venue_hours` reads venue-local static data and does
  not require a location.
- CLI `venues`, `cart`, `checkout`, and `account favorites` support
  `--lat/--lon`.

## Command Selection

Prefer MCP when present. Names below are canonical; call the host-prefixed
form if that is what you have.

- "What should I eat right now?" — `wolt_top` / `wolt top [N]`
- "What's on the home page?" — `wolt_feed` / `wolt feed --summary` for a
  one-line-per-section overview. Full `wolt_feed` / `wolt feed` for
  per-section detail.
- Flat list / filtered search: `wolt_search_venues` / `wolt venues`,
  `wolt_venue_categories` / `wolt venues categories`
- Product search across nearby venues: `wolt_search_items` / `wolt items
  --query <text>`. Preserve global rank; expand a venue with
  `wolt_venue_search_items` / `venue menu <venue> --query <text>`.
  Completeness is unknown (no continuation token or exact total).
- Exact venue from a name, slug, ID, or URL: `wolt_resolve_venue` (MCP).
  CLI: pass the slug/ID/URL straight to `wolt venue`.
- Inspect one venue: `wolt_venue_detail`, `wolt_venue_menu`,
  `wolt_venue_hours` / `wolt venue …`
- Resolve one item: `wolt_venue_item` / `wolt venue item`
- Basket and pricing: `wolt_cart_*` then `wolt_checkout_preview` /
  `wolt cart …` then `wolt checkout`
- Account and history: `wolt_account_*`, `wolt_favorites_*` /
  `wolt account …`, `wolt status`

Prefer `wolt_top` / `wolt top` over parsing the full feed for "I'm hungry"
queries. `wolt_search_venues` / `wolt venues` when the user already has a
search term. Discovery rows carry `tagline`, `top_offer`, `badges`, and
`menu_highlights`.

For large marketplace venues, prefer menu query or a category slug instead
of an unrestricted full-catalog crawl.

`wolt stats` and option-aware `cart add --option` are CLI-only. See
`references/mcp-tools.md` for gaps.

## Output and Diagnostics

- MCP: read `structuredContent`; on errors read `_meta.wolt_error` (and the
  short `content` message). Do not invent CLI envelope fields.
- CLI: `--format json|yaml` envelope keys `meta`, `data`, `warnings`,
  optional `error`. On upstream failures, rerun with `--verbose`.

## References

- Canonical MCP names, CLI map, host-prefix rule: `references/mcp-tools.md`
- Full CLI command and flag matrix: `references/command-reference.md`
- Reusable high-confidence workflows: `references/workflows.md`
- Envelope/error parsing: `references/output-and-errors.md`
- Operator setup (CLI, stdio MCP, Docker HTTP): repo `README.md` and `docs/mcp.md`
