# MCP tools and CLI map

Canonical protocol names are `wolt_*`. MCP hosts (Cursor, Claude Desktop, …)
prefix them with the client server key — for example Cursor exposes
`mcp_wolt_wolt_feed` when the key is `wolt`. Match available tools by
**suffix** (`…wolt_feed`). Never hardcode a host prefix, server key, or
hostname. Stdio and Streamable HTTP (including Docker) share this catalog.

Do not call `/mcp` or `/healthz`. Do not `docker exec` to run these tools.

## Intent map

| Intent | Canonical MCP tool | CLI fallback |
|---|---|---|
| Home-page feed | `wolt_feed` | `wolt feed` (`--summary` for one line per section) |
| Ranked "eat now" list | `wolt_top` | `wolt top [N]` |
| Filtered venue search | `wolt_search_venues` | `wolt venues` |
| Location category slugs | `wolt_venue_categories` | `wolt venues categories` |
| Global item search | `wolt_search_items` | `wolt items --query <text>` |
| Geocode an address | `wolt_resolve_address` | `--address` on a location-aware command |
| Exact venue (name/slug/id/URL) | `wolt_resolve_venue` | pass the identifier to `wolt venue` |
| Venue page | `wolt_venue_detail` | `wolt venue <venue>` |
| Menu / category / search | `wolt_venue_menu` | `wolt venue menu <venue>` |
| In-venue item search | `wolt_venue_search_items` | `wolt venue menu <venue> --query <text>` |
| Hours | `wolt_venue_hours` | `wolt venue hours <venue>` |
| One item + options | `wolt_venue_item` | `wolt venue item <venue> <item>` |
| Profile / session | `wolt_account_status` | `wolt status` / `wolt account` |
| Orders | `wolt_account_orders`, `wolt_account_order` | `wolt account orders`, `wolt account order` |
| Addresses (read) | `wolt_account_addresses` | `wolt account addresses` |
| Payments | `wolt_account_payments` | `wolt account payments` |
| Favorites | `wolt_favorites_list`, `wolt_favorites_add`, `wolt_favorites_remove` | `wolt account favorites …` |
| Cart | `wolt_cart_show`, `wolt_cart_count`, `wolt_cart_add`, `wolt_cart_remove`, `wolt_cart_clear` | `wolt cart …` |
| Checkout preview | `wolt_checkout_preview` | `wolt checkout` |

⚠ = mutates user state: `wolt_cart_add`, `wolt_cart_remove`, `wolt_cart_clear`,
`wolt_favorites_add`, `wolt_favorites_remove`. Confirm with the user first.

No tool places an order.

## Gaps vs CLI (do not invent)

- **Login / logout** — no MCP tools. On **HTTP MCP**, do not run
  `wolt login`; ask the user to check and provide credentials. On CLI /
  local stdio, host-side `wolt login` / `wolt logout`.
- **Stats dashboard** — `wolt stats` only.
- **Address book writes** — CLI `account addresses add/update/remove/use` only.
  MCP `wolt_account_addresses` is read-only.
- **`wolt_cart_add` options** — the MCP tool does not accept option
  selections; it fails closed when an item requires one. Use CLI
  `wolt cart add … --option "Drink=Cola"` (or IDs) for configurable items.
- **CLI-only cart helpers** — `--query` name resolve on `cart add`,
  `--option` name resolution, `--details` table flags.

Operator wiring (stdio vs Docker HTTP, tokens, ports): repo `README.md` Path B
and [`docs/mcp.md`](../../../docs/mcp.md).

## MCP result shape

Successful calls put the typed payload in `structuredContent`. `content` is a
short summary unless the client requested JSON duplication.

Errors: short message in `content`; stable shape in `_meta.wolt_error`:

```json
{
  "code": "RATE_LIMITED",
  "message": "wolt is rate-limiting requests; retry after 2s",
  "retryable": true,
  "retry_after_ms": 2000
}
```

Auth failures use `AUTH_REQUIRED` / `AUTH_EXPIRED` / `SESSION_REFRESH_FAILED`.
On HTTP MCP, ask the user to check and provide credentials; do not run
`wolt login`. On CLI / local stdio, they may run `wolt login` on the host.

CLI envelope parsing (`meta` / `data` / `warnings` / `error`):
`output-and-errors.md`.
