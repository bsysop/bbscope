# Changes

## Program Brief Storage

Added support for fetching and persisting program briefs (rules of engagement / descriptions) across all platforms.

### Database

- Added `brief TEXT NOT NULL DEFAULT ''` column to the `programs` table (`pkg/storage/storage.go`)
- `UpsertProgramEntries` and `getOrCreateProgram` now accept and persist `brief`
- Upsert logic preserves an existing brief if the new value is empty (`CASE WHEN excluded.brief != '' THEN excluded.brief ELSE programs.brief END`)

### Data model

- Added `Brief string` field to `scope.ProgramData` (`pkg/scope/scope.go`)
- Added `SkipBrief bool` to `platforms.PollOptions` (`pkg/platforms/platform.go`) — when `true`, brief fetching is skipped (used for daily polls; pass `--brief` to update)

### Platform extractors

| Platform | Source field | Notes |
|----------|-------------|-------|
| **HackerOne** | `attributes.policy` | Extra `GET /v1/hackers/programs/{handle}` call, gated by `SkipBrief` |
| **Bugcrowd** | `data.brief.description` / `data.brief.tagline` | From engagement brief version document; scope group `descriptionHtml` appended |
| **Intigriti** | `rulesOfEngagement.content.description` | Single nested path in program detail response |
| **YesWeHack** | `rules` (fallback: `description`, `policy`, `text`, `content`) | Root-level field in program detail response |
| **Immunefi** | `programOverview` RSC reference | Resolved via Next.js RSC text chunk format: `REFID:T<byteLen>,<text>` |

### CLI flag

- `--brief` flag added to `bbscope poll` — brief fetching is off by default; pass this flag when running monthly updates

---

## Program Filter Flag

Added `--program <string>` flag to `bbscope poll` to narrow a full platform poll to a single program.

- Filters handles containing the given string after `ListProgramHandles` returns
- Works in both `--db` and non-DB modes (`cmd/poll.go`, `pkg/polling/polling.go`)
- Safety check (abort if 0 handles returned) is bypassed when `--program` is active

---

## Bugcrowd Login Fix

Fixed login flow to handle accounts without MFA enabled (`pkg/platforms/bugcrowd/bugcrowd.go`).

Previously, the code assumed MFA was always required and returned an error when `needsMfa` was `false`. Now it follows the `redirect_to` URL and extracts the session cookie directly, matching the MFA-less login flow.

---

## DB Connection: Password Injection Support

Added `db_password` config key (and `DB_PASSWORD` env var) to allow injecting a database password separately from the connection URL (`cmd/root.go`).

- Useful when the password contains characters that break URL percent-encoding (`%`, `@`, `#`, etc.)
- Implemented via `buildKVDSN`: parses the postgres URL with a regex (bypassing `url.Parse`) and emits a libpq `keyword=value` DSN with the password single-quote-escaped
- `storage.go` `createDatabase` updated to support both URL and keyword=value DSN formats when initialising the database

---

## Bugcrowd Engagement View Fix

Fixed `getEngagementBriefVersionDocument` return signature (now returns 3 values to align with callers expecting an additional string return) (`pkg/platforms/bugcrowd/bugcrowd.go`).

---

## Paused Program Detection

Paused/suspended programs are now detected across all platforms and marked `disabled` in the `programs` table, so they are excluded from active scope while their scope data is retained.

### Motivation

Previously bbscope only distinguished "present in the platform's program list" (`disabled=0`) from "removed from the list" (`disabled=1`). Paused programs are still returned by the platform APIs, so they stayed active and their assets kept flowing into downstream tooling even though the program was not accepting testing.

### Data model

- Added `Paused bool` to `scope.ProgramData` (`pkg/scope/scope.go`)
- `UpsertProgramEntries` / `getOrCreateProgram` take a `paused` flag and set `disabled = paused` on upsert (`pkg/storage/storage.go`). Paused programs stay in the polled list, so `SyncPlatformPrograms` does not delete their targets — their scope is **retained** (`disabled` now means removed **or** paused)

### Platform paused signals

| Platform | Signal |
|----------|--------|
| **Intigriti** | `status.id == 4` ("Suspended") — the ⏸ paused badge |
| **Bugcrowd** | brief `data.engagement.state == "in_progress_paused"` (or non-empty `pausedReason`) |
| **HackerOne** | `attributes.submission_state == "paused"` (kept + flagged instead of dropped; fully closed/`disabled` still skipped) |
| **YesWeHack** | list item `disabled == true` (kept + flagged instead of dropped) |

### Asset extraction

- `ListEntries` (`bbscope db get urls/wildcards/domains/ips` and the website `/api` target feeds) now filters `AND p.disabled = 0`, so paused programs' assets are excluded from active output.

### Website

- Program detail page distinguishes **paused** (disabled but live scope → amber "Program Paused" notice) from **removed** (disabled, scope reconstructed from history → red "Program Removed" banner) (`website/pkg/core/program.go`)

### Operational note

`bbscope poll --program <x>` must **not** be used to roll this out: a filtered poll passes only the matching handles to `SyncPlatformPrograms`, which marks every *other* program on that platform `disabled=1` and deletes their targets. Use a full per-platform poll.
