# Workflows

Prefer MCP when `wolt_*` tools are available (match by suffix). CLI snippets
are the fallback.

## 1) Authenticate and Validate Session

**HTTP MCP:** do not run `wolt login`. Probe with `wolt_account_status`. If
the call fails for missing or expired credentials, ask the user to check
the mounted config and provide `wtoken` + `wrtoken` (or cookies). Retry
after they confirm.

**CLI / local stdio:**

```bash
# Host-side. Browser-driven (managed Chrome at 127.0.0.1:9222).
wolt login

# Manual tokens (no browser).
wolt login --wtoken "<token>" --wrtoken "<refresh-token>"

wolt status --format json --verbose
wolt account --format json
```

If an auth-gated CLI call fails (`WOLT_AUTH_REQUIRED`), ask the user to
re-run `wolt login`, then retry. Never `docker exec` login.

## 1a) Quickest "What Should I Eat?" Loop

MCP: `wolt_top` (limit 10), `wolt_feed` (optionally with a query).

```bash
wolt top 10                         # single ranked table, no jq
wolt feed --summary                 # one line per section overview
wolt feed --query "burger"          # filter the feed (matches brand carousels too)
```

## 2) Find Venue, Inspect Item, Add to Cart, Preview Checkout

MCP: `wolt_search_venues` / `wolt_search_items` → `wolt_resolve_venue` or
`wolt_venue_detail` → `wolt_venue_menu` / `wolt_venue_item` → confirm →
`wolt_cart_add` (plain items only; options need CLI) → `wolt_cart_show` →
`wolt_checkout_preview`.

```bash
# Discover/search
wolt venues  --query "burger" --limit 10 --format json
wolt items --query "whopper" --limit 20 --format json
wolt venue <venue-slug>  --format json

# Resolve item and options
wolt venue menu <venue-slug> --query "whopper" --include-options --limit 10  --format json
wolt venue item <venue-slug> <item-id>  --format json

# Mutation (confirm with user first)
wolt cart add <venue-id> <item-id> --venue-slug <venue-slug> --option "<group-id>=<value-id>"  --format json

# Validate basket and pricing preview
wolt cart --venue-id <venue-id> --details  --format json
wolt checkout --venue-id <venue-id> --delivery-mode standard  --format json
```

## 3) Large Marketplace Venue Strategy (Partial Assortments)

Use this path when `wolt_venue_menu` / `venue menu` is incomplete or returns
partial-assortment guidance.

MCP: `wolt_venue_menu` with `category`, or `wolt_venue_search_items`.

```bash
wolt venue categories <venue-slug>  --format json
wolt venue menu <venue-slug> --query "milk"  --format json
wolt venue menu <venue-slug> --category <category-slug> --include-options  --format json
```

Use `--full-catalog` only when explicitly needed; it can be slow.

## 4) Orders and Payment/Profile Inspection

MCP: `wolt_account_orders`, `wolt_account_order`, `wolt_account_payments`,
`wolt_favorites_list`.

```bash
wolt account orders  --limit 20 --format json
wolt account order <purchase-id>  --format json
wolt account payments  --mask-sensitive --format json
wolt account favorites  --format json
```

## 5) Address Book Operations (Mutating)

Confirm intent before add/update/remove/use. MCP `wolt_account_addresses` is
read-only; writes are CLI-only.

```bash
wolt account addresses  --format json
wolt account addresses add  --address "<formatted>" --lat <lat> --lon <lon> --label home --format json
wolt account addresses links  --format json
wolt account addresses use <address-id>  --format json
```

## 6) Location Override Rules

MCP: `address` **or** both `lat` + `lon`. CLI:

- Valid: `--address "Kamppi, Helsinki"`
- Valid: `--lat 60.1699 --lon 24.9384`
- Invalid: `--address ... --lat ... --lon ...` together
- Invalid: only one coordinate flag

On invalid combinations, expect `WOLT_INVALID_ARGUMENT` (CLI) or a tool error
(MCP).
