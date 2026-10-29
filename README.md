# AND Search Engine

**English** · [Türkçe](README.tr.md)

![Go](https://img.shields.io/badge/Go-1.26%2B-00ADD8?logo=go&logoColor=white)
![License](https://img.shields.io/badge/license-AGPL--3.0-blue)
![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-lightgrey)

A program that searches the content of the **[AND](https://github.com/AND-community/AND)**
ecosystem (P2P forum + chat + personal sites) and runs **completely independently of AND**. ("Arama
motoru" is Turkish for "search engine".)
Even while AND itself is closed, this program joins AND's P2P network directly as a peer with its
own network identity and its own libp2p connection, collects the content and makes it searchable in a
local full-text index — without importing AND's internal packages and without running AND as a
subprocess.

---

## Contents

- [Overview](#overview)
- [Features](#features)
- [Why a standalone program and not a "plugin"?](#why-a-standalone-program-and-not-a-plugin)
- [Architecture](#architecture)
- [Requirements](#requirements)
- [Installation and running](#installation-and-running)
- [Usage](#usage)
- [Command-line options](#command-line-options)
- [Sharing site pages](#sharing-site-pages)
- [Advanced query syntax](#advanced-query-syntax)
- [Theme](#theme)
- [Languages](#languages)
- [Data directory files](#data-directory-files)
- [Scope and limits](#scope-and-limits)
- [Ecosystem](#ecosystem)
- [Contributing](#contributing)
- [Security](#security)
- [License](#license)

---

## Overview

The AND Search Engine is a terminal application (TUI) — and at the same time a scriptable
command-line tool (CLI) — that joins AND's P2P forum/chat/personal-site network with its own
independent identity, indexes the content travelling on the network and lets you search it with a
Google/Yandex-like experience (full-text search, relevance ranking, spelling suggestions, paging).

It doesn't use AND's internal packages and doesn't run AND as a subprocess — it re-implements the
protocols (signature format, sync flow) independently; only the topic names and protocol IDs come
from the `protokol` package shared with AND, and that the signature formats stay identical to AND's
is tested with sample messages produced by AND (`../AND/protokol/testdata/`, see
`protokol_altin_test.go`). It joins the network as a **real peer of its own**. That is why you can
see and search the content on the network even when AND isn't running at all.

---

## Features

| Feature | Description |
|---------|----------|
| Fully standalone | Needs no connection to AND and no environment variables; runs with its own identity and network connection |
| Full-text search | SQLite FTS5 based, accent-insensitive for Turkish, ranked by `bm25()` + a freshness boost |
| Multi-hop network search | Also searches the indexes of other nodes on the network with its own P2P protocol (`/and-arama-motoru/1.0.0`) |
| Advanced query syntax | `"exact phrase"`, `-exclude` and `term1 OR term2` are supported |
| Spelling suggestions | A "did you mean?" suggestion when there are zero results (no extra dependency) |
| Search by author | List all content by a result's author with one key |
| Scriptable CLI | Search the local index with `-search`/`-site` without opening the TUI or the network; machine-readable output with `-json` |
| Personal site share codes | Share a site page directly with an `authorKey/slug` code or (on Windows and Linux) with an `and-site://` link that opens with one click |
| Clickable links | `http(s)://` links in results and `and-site://` links inside pages are really clickable in the terminal |
| Visual identity | The same design language as AND (palette, logo, list rows, hint line); customisable theme |
| Panic shield | A parsing error in untrusted network content doesn't crash the program |
| Crash-safe storage | The identity/peer list files are protected with atomic writes |
| Sort mode | Sort results by Relevance (bm25+freshness) or only by Date (Newest) (`ctrl+r`) |
| Saved searches / "follow" | Follow a search; get notified with an "N new" badge as newly matching content arrives from the network |
| Targeted P2P query | Query only the up-to-date index of the peer a network result came from |
| Peer trust indicator | On the Status screen, see the ratio of validly/invalidly signed results of the peers queried directly |
| Site map and code search that "falls back to the network" | List ALL site pages of an author with just their author key; a code that isn't available locally is searched on the network automatically |
| In-page cross links and QR codes | Follow `and-site://` links in a site page's body; show a share code as a QR code in the terminal |
| Edit sync | The EDITED version of a site page is also caught retroactively through sync — not only pages that were never seen before |

---

## Why a standalone program and not a "plugin"?

This folder is **outside** AND's Go module: under Go's package visibility rules it can't import the
`AND/internal/...` packages (network, forum, authentication logic). That is why it **re-implements
AND's protocols independently** (listed below) — the same GossipSub topics, the same signature
format, the same sync protocol, but its own code. The result: a real peer that can connect to the
network on its own and see the same content, even when AND isn't running.

## Architecture

1. **Its own identity** (`identity.go`) — with the same derivation scheme AND uses (BIP39 mnemonic → ed25519 key) but completely independent of the user's actual AND identity; generated on the first run and saved to `~/.and-arama-motoru/kimlik.json`.
2. **Network layer** (`network.go`) — its own libp2p host; the same peer discovery mechanism as AND: a DHT built on the public IPFS bootstrap peers (rendezvous: `and-community/1.0.0`) + mDNS on the local network (`and-community-mdns`).
3. **Content intake** (`forum.go`, `chat.go`, `site.go`) — subscribes to AND's forum (the `github.com/lucian95511/and/forum` topic), chat (the `.../chat` topic) and personal site (the `.../site/1.0.0` topic) GossipSub broadcasts; it verifies the ed25519 signature of every post/reply/message/page with **exactly the same payload format** as AND itself — nothing unsigned or tampered with is indexed. With every newly connected peer it also collects the history retroactively by speaking AND's sync protocols (`/and/sync/1.0.0`, `/and/chatsync/1.0.0`, `/and/sitesync/1.0.0`). UNLIKE forum posts and chat messages, site pages CAN CHANGE (they change completely when the same author/slug is published again) — so instead of the write-once `Ekle`, `Depo.EkleYaGuncelle` with "newest wins" upsert semantics is used.
4. **Local index** (`index.go`) — SQLite FTS5, accent-insensitive search for Turkish; its own separate database file (`~/.and-arama-motoru/arama_motoru.db`), it never touches AND's `forum.db`. Ranking combines a column-weighted `bm25()` (a title match is worth much more than a body match) with a freshness boost — among equally relevant results, newer content moves slightly ahead. Results come with an excerpt (snippet) that highlights the matching term; a reply result also shows the title/category of the topic it belongs to (if the topic has already been synced).
5. **Its own P2P search protocol — MULTI-HOP** (`search.go`) — a SEPARATE libp2p protocol of this program (`/and-arama-motoru/1.0.0`): you can send a query directly over a stream to peers that run another copy of this program and get results from their own indexes ("search the network", `ctrl+n`). Unlike AND's DM channel there is no size limit (the full body is carried), and the sender's identity is known from the connection itself (in a way that can't be faked). A query isn't limited to DIRECTLY connected peers: unless its own TTL (3 hops by default) is used up, every node also forwards the query to its own peers, collects the answers and returns a single combined result (the classic "flood & aggregate" P2P search pattern) — so content in the index of a node a few hops away on the network can be found too. The same query circulating endlessly or being repeated needlessly is prevented by processing a shared query ID only once on every node (see `sorguOnbellek`). Every result is labelled with the peer ID of the node that ACTUALLY indexed it (carried unchanged by intermediate nodes) — you know the real source regardless of how many hops away it came from.
6. **Known peer list** (`peers.go`) — the addresses of the peers you are currently connected to are periodically written to `~/.and-arama-motoru/bilinen_peerler.json`; on the next start it tries to connect to them directly without waiting for DHT/mDNS discovery, which makes reconnecting after a restart noticeably faster.
7. **Levelled, size-limited diagnostic log** (`log.go`) — without affecting the TUI at all (not to stdout), it writes lifecycle events and errors with the levels `[bilgi]`/`[uyarı]`/`[hata]` (info/warning/error) to `~/.and-arama-motoru/gunluk.log` — to diagnose problems such as "why are there no results at all". When the file exceeds 5 MB it is rotated to `.old` automatically and starts from scratch, so it doesn't grow without bounds in a long-running process. The libraries' own logs (the standard `log` package and libp2p's go-log) go to this file instead of the terminal as well. On Windows, the "Failed to set multicast interface" warnings that mDNS repeated for virtual network interfaces used to be written over the full-screen interface and break the screen; this harmless warning is now written to the log only once.
8. **Ordered, correct shutdown** (`main.go`) — Ctrl+C/a shutdown signal first tells all background goroutines to stop and waits for them to REALLY stop (including saving the known peer list one last time), and only THEN closes the network/database resources — doing it in the reverse order could make a still-running operation try to use a closed resource, or make the save at shutdown fail entirely.
9. **Opening links** (`links.go`) — `http(s)://` links in a result's details are found and numbered automatically and made **really clickable with the mouse** with the OSC 8 terminal hyperlink sequence (supported by Windows Terminal, VS Code's integrated terminal, iTerm2 and most modern Linux terminals — when clicked, the terminal opens the url itself). For terminals that don't support this, `o` (the first one) or the `1`-`9` keys open them in the operating system's default browser (the same pattern as `klasoruAc` in AND's DeltaX plugin). Only http(s) schemes are recognised — other schemes such as `file://` or `javascript:` are deliberately rejected; since the content comes from untrusted peers, this is an important security boundary.
10. **Scriptable CLI mode** (`cli.go`) — the `-search "query"` flag searches the local index WITHOUT opening the TUI or the network at all, writes the results to stdout and exits (machine-readable output with `-json`, a type filter with `-tip`, the number of results with `-limit`, paging with `-sayfa`). If there are zero results and a "did you mean?" candidate is found, it is written to `stderr` so that `stdout` (e.g. `| jq`) stays intact.
11. **Panic shield** — because content comes from UNTRUSTED peers on the network (see the security principle above), an unexpected edge case in the parsing/processing logic no longer crashes THE WHOLE PROGRAM: both the TUI's `Update` loop and all background goroutines that process content/queries (`recoverGoroutine`) are wrapped in a panic shield — if a panic happens, it is logged and ignored, and the program keeps running.
12. **Adjustable log level and TTL** (`main.go`) — the detail level of the diagnostic log (`hata`/`uyari`/`bilgi`, i.e. error/warning/info) can be set with `-log-level`, and the default number of hops for "search the network" (1-5) with `-ttl`, at run time.
13. **Search by author — "this author's other content"** (`index.go`, `search.go`) — in a result's details, the `a` key switches to a search filtered by the selected result's author key (the real identity, not the spoofable YazarAd/author name); this filter DOESN'T NEED a keyword — if left empty, all of that author's content is listed from newest to oldest (the "author page" behaviour of real search engines). It works both in the local index and in "search the network" (the network protocol carries the author filter too); `ctrl+y` clears it. Its CLI counterpart is `-yazar <key>` (usable on its own, even without `-search`).
14. **Performance: WAL mode and connection pool** (`index.go`) — SQLite runs in WAL journal mode, pinned to a single connection (`SetMaxOpenConns(1)`), for the same reason as AND's own `internal/storage`: it reduces lock contention between the constant gossip writes and search queries. An index on `bilinen(tip)` was added — the "known IDs" list sent to every new peer during sync no longer does a full table scan.
15. **Crash-safe file writes** (`fsutil.go`) — `kimlik.json` and `bilinen_peerler.json` are written with a temporary file + atomic rename (`atomicYaz`) instead of a direct write that could be left half-written/corrupt; if the process is interrupted right in the middle of writing (power cut, kill), the file stays either completely old or completely new.
16. **The same visual language as AND and a real theme system** (`view.go`, `tema.go`) — AND's shared design system `internal/ui` is repeated here independently (this program is a separate Go module, so it can't import the package): the same ANSI 256 code points as AND's default "Klasik" (Classic) theme, the same block-letter "AND" logo, a full-width background and a `▌` accent bar on the selected row, `━`/`─` fill bars and one-line `✓`/`✗` notices. The line at the bottom of every screen is no longer the old one-line status bar but, as in AND, a hint line made of "key description" pairs; a pair that doesn't fit moves to the next line without being split. Long lines (excerpts, links, share codes) are cut or wrapped to the screen width, and the frame never overflows at any width. Like AND, it **uses the whole terminal**: every screen is inside a full-screen frame (the `AND` badge, the theme name and the version on the top edge), the hint line and notices are always at the bottom, and the splash screen sits in the middle of the frame. Result bodies, cards and input boxes stay at most 100 columns wide; the cards on the Status screen are laid out in two columns on wide terminals. The colours are not fixed: at startup the program reads **the theme selected in AND**, and its own `~/.and-arama-motoru/tema.json` takes priority if it exists (see [Theme](#theme)).
17. **Personal site pages are searchable too** (`site.go`) — an independent re-implementation of AND's personal site feature (`internal/site`): it indexes pages with the SAME security discipline as forum/chat (ed25519 signature, real identity, not a pseudonym), but it knows that UNLIKE the "write-once" nature of forum posts/replies, this is a content type that CAN CHANGE: when the same author/slug is published again, the old version in the index is overwritten (not deleted and re-added), and an old version that arrives late (replay) never overwrites a newer one we already have.
18. **Google/Yandex-style advanced query syntax** (`ftsSorguOlustur` in `index.go`) — `"exact phrase"` (adjacent match), `-exclude` (exclude from the results) and `term1 OR term2` (either one) are supported; if none of them is used, the output is byte for byte the same as the old behaviour (backward compatible). Injection into FTS5's own mini query language is not possible — every term is escaped and wrapped in its own quoted literal.
19. **"Did you mean?" spelling suggestions** (`yazimonerisi.go`) — needs NO new dependency: it reads all unique terms in the index with SQLite FTS5's `fts5vocab` extension and finds the closest known term with a hand-written Levenshtein (edit distance) function. It is triggered only when a search returns ZERO results; in the TUI you can apply the suggestion directly with `ctrl+d`.
20. **Paging** (`index.go`, `model.go`) — results can be browsed page by page both in the local search (`pgup`/`pgdown`) and in the CLI (`-sayfa`); to tell whether there is a next page without running an extra `COUNT(*)` query, every page fetches one row more than requested and drops the last row.
21. **Search console statistics** (`TipDagilimiAl`/`EnAktifYazarlarAl`/`GunlukIcerikTrendiAl` in `index.go`) — like Google Search Console/Yandex Webmaster, the Status screen shows the content type breakdown, the most active authors and the daily content trend of the last 14 days. These queries run only once WHEN the Status screen is ENTERED (NOT in the existing 2-second status refresh) — so as not to put needless load on a database pinned to a single connection with `SetMaxOpenConns(1)`.
22. **Share codes for personal site pages** (`SiteSayfasiBul` in `index.go`, the "Open by Code" screen) — the ID that a site page already has for personal sites (`authorKey/slug`) is now shown in the Result Details as a **selectable code** (you can copy it and send it to someone by message/mail). When the recipient pastes this code into the "Open by Code" screen on the main menu and presses `enter` (or uses `-site <code>` in the CLI), they can open the page directly — WITHOUT WAITING to "come across" that author in the forum/chat first. The `-site` flag in the CLI stays a LOCAL query (it doesn't go to the network, so that scripting stays predictable/fast); the "Open by Code" screen in the TUI has the network fallback described in item 27. **Important limit:** this works only between people who run AND or the AND Search Engine and are connected to the same P2P network — it is NOT a real web address/browser link (a P2P architecture would need a separate, larger HTTP bridge for that, which is out of scope).
23. **Sort mode: Relevance ⇄ Newest** (`SiralamaModu`/`AraGelismisSirali` in `index.go`, `ctrl+r` on the Search screen) — the default (Relevance) ranking keeps using the column-weighted `bm25()` + freshness boost; `ctrl+r` switches to a mode that IGNORES it and sorts results purely chronologically (newest to oldest). The external signature of `AraGelismis` (current callers: the CLI, the tests) is KEPT — the new `AraGelismisSirali` is a superset that adds an extra parameter on top of it.
24. **Saved searches / "follow"** (the "Kayıtlı arama takibi" section in `index.go`, `ctrl+t` on the Search screen, the new "Followed" screen) — saves the current query/type/author combination in a LOCAL ledger; from then on, whenever a **really new** record from forum/chat/site gossip matches this follow (the same `ftsSorguOlustur`/FTS5 MATCH logic is REUSED for the question "does this record match query X"), a counter goes up; it shows on the Main Menu as a "Followed (N)" badge. It is a completely local ledger — nothing is published to the network, and the read-only peer principle is not broken.
25. **Targeted P2P query** (`aramaTekEsIstegi` in `search.go`, the `t` key in the Result Details) — queries the `KaynakPeer` (source peer) of a network result as a SINGLE target (TTL=1, no forwarding), unlike `ctrl+n`, which spreads to ALL peers; to check "what else is in this peer's current index". It uses the same `/and-arama-motoru/1.0.0` protocol and the signature verification at the SINGLE choke point (`aramaIstekGonderRaw`), and NEEDS no separate protocol.
26. **Peer trust indicator** (`esGuvenDefteri` in `search.go`, the Status screen) — tracks, in a process-lifetime in-memory ledger, how many validly/invalidly signed results every peer queried DIRECTLY by this node returned, and lists the 5 peers interacted with most on the Status screen. This is NOT a BLOCKING mechanism (security is already provided by `dogrulaAgSonucu` itself) — it is only a visual hint about "which peers behave suspiciously"; it is not written to disk and resets on every restart.
27. **"Open by Code" falls back to the network** (`agdaSiteAraCmd` in `model.go`) — when a share code is not found in the LOCAL index (and there are connected peers), the TUI automatically triggers a network query (the SAME search protocol as in item 5, `Metin=""` + `Tip=TipSite` + `YazarKey`) targeting the code's author key and looks for an exact `KaynakID` match — a page that has NOT yet been synced to this node but still exists on the network can be found this way too. The page found (having already passed signature verification) is safely persisted to the local index as well.
28. **Site map** (the "Open by Code" screen in `model.go`) — if you paste only an `authorKey` (an input without any `/`) instead of `authorKey/slug` into "Open by Code", ALL site pages of that author (`yazarinTumIcerigi` in `index.go` is REUSED) are listed on the Search screen with the TipSite filter instead of a SINGLE page; from there you can search the network for the same author's other pages with `ctrl+n`.
29. **In-page cross links** (`siteBaglantilariBul` in `links.go`, the `g` key in the Result Details) — `and-site://` links in the body of a site page (see item 22) are now listed separately like `http(s)://` links and made clickable with OSC 8; the `g` key goes to the first internal link from INSIDE the application, WITHOUT needing the protocol registration (with the SAME network fallback as in item 27 if it isn't available locally). It lets you move from page to page like a personal "wiki".
30. **QR code for share codes** (`qr.go`, the `k` key in the Result Details) — pressing `k` turns a site page's `and-site://` link into a scannable QR code drawn in the terminal with half-block characters (▀▄█) (`skip2/go-qrcode`, pure Go — no CGO needed); to scan the code with a phone instead of copy-pasting it to a device.
31. **The `and-site://` protocol registration now exists on Linux too** (`protokol_linux.go`) — besides Windows (`protokol_windows.go`, the HKCU registry), `-protokol-kaydet`/`-protokol-kaldir` now also work on Linux desktop environments through the `xdg-mime`/`.desktop` file mechanism (only under `$HOME/.local/share/applications`, NO root rights needed). This is still missing for macOS — Launch Services (`lsregister`) accepts as the unit of registration not a bare binary but a real `.app` bundle that declares `CFBundleURLTypes`; that needs a separate packaging step beyond `go build` producing a single binary, and was left out of scope.
32. **Edited site pages arrive through sync too** (`siteSyncIstek.KnownVersions` in `site.go`, `SiteSurumleriAl` in `index.go`) — the site sync request now carries not only the known slugs but also the version (`UpdatedAt`, with nanosecond precision) of every page we have; AND's version-aware responder (`internal/site/sync.go`) still sends a known slug if its version is NEWER. Previously the edited version of a page never reached this node through sync — it was indexed only if we happened to see the live gossip at that moment. An older AND version that doesn't know this field ignores it (the old behaviour); if the request grows large enough to exceed AND's 1 MB request limit, the versions are dropped and it is sent in the old form (so that new pages aren't lost).
33. **Text from the network is sanitised before it reaches the terminal; the search protocol is limited against amplification** (`guvenlik.go`, `search.go`) — signed content only proves WHO wrote it, not that what they wrote is harmless. Titles, bodies, names, categories and share codes are cleaned of terminal escape sequences (OSC 52, which writes to the clipboard; OSC 8, which redirects links; CSI, which moves the cursor), C0/C1 control characters and text-direction characters before they are printed on the screen (and in the `-search` command-line output); if a clickable link's address contains a control character, the link is not created at all. The cleaning is done on the screen, not in storage — records are passed on to other peers with their signatures, so changing the text would break signature verification. In the search protocol: queries are sent only to peers that speak this protocol (most connected peers are generic DHT nodes), another peer's query is forwarded to at most 8 peers per hop, at most 16 incoming queries are processed at once, the query text is limited to 256 characters, the deadline announced by the sender is limited to this node's own timeout, and requests that carry an invalid author key or content type are rejected. The search server is now registered BEFORE connecting to any peer, so that peers learn that this node is a search engine at the very first contact.

## Requirements

- **Go 1.26.4 or later** (to build from source — the version in `go.mod`; Go 1.21 and later download the required toolchain on their own)
- **Operating system:** Windows, Linux or macOS (libp2p and `modernc.org/sqlite` are all pure Go/CGO-free, so it works without problems, cross-compilation included)
- **Network access:** an internet connection that allows outbound connections — needed for the DHT bootstrap and peer discovery; if you only want to talk to peers on your local network, mDNS alone is enough
- **Disk:** only as much as the content you index (an SQLite file of a few tens of MB in typical use)
- CGO/an external compiler is **not needed** — all dependencies are pure Go

## Installation and running

To build it, AND's source folder must sit in the neighbouring folder (`../AND`): this project takes
the small `protokol` and `dil` packages it shares with AND from there (the `replace` in `go.mod`).
Clone the two repositories side by side into the same folder:

```bash
git clone https://github.com/AND-community/AND.git
git clone https://github.com/AND-community/and-arama-motoru.git
cd and-arama-motoru

go build -o and-arama-motoru .
./and-arama-motoru

# Windows
go build -o and-arama-motoru.exe .
.\and-arama-motoru.exe
```

While it runs, it needs no connection to AND and no environment variables — it runs independently.

### Command-line options

```
-version         print version information and exit
-veri-dir <path> data directory (default: $ARAMA_MOTORU_VERI_DIR or ~/.and-arama-motoru)
-log-level <s>   log detail level: hata|uyari|bilgi (error|warning|info; default: bilgi)
-ttl <n>         default number of hops (TTL) for "search the network", 1-5 (default: 3)

-search <query>  search the local index, print the results and exit (no TUI/network)
-tip <t>         with -search: limit results to a type (konu|yanit|sohbet|site =
                 topic|reply|chat|site)
-yazar <key>     only results of this author key; even with an empty -search it
                 lists all of that author's content on its own
-limit <n>       with -search: at most this many results (default: 20)
-sayfa <n>       with -search: page number, starting at 1 (default: 1)
-site <code>     look up a site share code (authorKey/slug) in the local index,
                 print the page and exit (no TUI/network) — see "Sharing site
                 pages" below
-json            with -search/-site: print the result as JSON
-protokol-kaydet  register this program with the operating system to open
                  and-site:// links (Windows and Linux) and exit
-protokol-kaldir  remove the and-site:// link registration (Windows and Linux)
                  and exit
```

Flag names and values stay Turkish in every interface language (scripts depend on them).

The version can be overridden at build time with `-ldflags "-X main.Version=vX.Y.Z"` (default
`v0.1.0`); optionally `-X main.BuildCommit=...` and `-X main.BuildDate=...` can be filled in too, and
if set they are shown in the `-version` output (together with the Go version/OS/architecture). The
`-h`/`-help` output is grouped into General/Search/Site sharing/Network categories.

Example — scriptable search:

```bash
./and-arama-motoru -search "merhaba" -json -limit 5 | jq '.[].baslik'
```

### Sharing site pages

Personal site pages have no real web address (there is no such concept on a P2P network) — but
every page already has an ID in the form `authorKey/slug`, which can be used as a **share code**. In
a page's Result Details screen this code is shown with the label "Code"; you can select and copy it
with the mouse, or press `c` to copy it straight to the system clipboard (with the OSC 52 terminal
protocol — it copies to your LOCAL clipboard even over SSH; terminals that don't support it silently
ignore it). The person who receives the code:

```bash
# CLI:
./and-arama-motoru -site "a3f9…92c1/hakkimda"

# or in the TUI: Main Menu → "Open by Code" → paste the code → enter
```

**Limit:** this works only for people who run AND/the AND Search Engine — it is not a real web link
that can be opened in a browser. The recipient DOESN'T NEED to have the page in their local index
already (through the forum/chat/sync): if the code isn't found locally and there is a connected
peer, the "Open by Code" screen automatically searches the network too ("Open by Code falls back to
the network" — see item 27 of the Architecture); the `-site` flag in the CLI stays with the LOCAL
search so that scripting stays fast/predictable.

If you paste only an `authorKey` (an input without any `/`) instead of a page code, "Open by Code"
lists ALL site pages of that author (the "site map") — see item 28 of the Architecture.

#### As a clickable link (and-site://)

The share code is also wrapped in a custom URI scheme of the form `and-site://<authorKey>/<slug>` —
it is shown clickable with OSC 8 on the "Link" row of the Result Details screen (it can be clicked/
selected with the mouse in terminals that support it). For the operating system to recognise it as
a REALLY clickable link (e.g. in a chat app or in a browser's address bar), you need to register it
once:

```bash
# Windows/Linux — only for the current user (no administrator/root rights needed):
./and-arama-motoru -protokol-kaydet

# To remove the registration:
./and-arama-motoru -protokol-kaldir
```

After registering, clicking an `and-site://…` link starts this program with that code (exactly like
the `-site` flag): no TUI/network is opened, the page is printed directly. **On Windows** it is
registered under `HKEY_CURRENT_USER`, **on Linux** with a `.desktop` file under
`$HOME/.local/share/applications` + `xdg-mime`. **Not implemented for macOS yet** — Launch Services
(`lsregister`) accepts as the unit of registration not a bare command-line binary but a real `.app`
bundle that declares `CFBundleURLTypes`; that needs a separate packaging step and was left out of
scope. On that platform the `-site` flag, the "Open by Code" screen and the `g` key on in-page
`and-site://` links (see item 29 of the Architecture) keep working as they are; only the convenience
of "opening automatically when the link is clicked" is missing.

This DOESN'T REMOVE the basic limit above: it still works only for someone who has AND/the AND
Search Engine installed AND has registered the protocol AND already has that content in their own
index — it doesn't rely on a real web server/DNS.

### Advanced query syntax

The search box (both in the TUI and in `-search`) understands three extra syntaxes similar to
Google/Yandex — if none of them is used, the behaviour is the same as before:

```bash
./and-arama-motoru -search '"exact phrase"'   # adjacent match, not a prefix
./and-arama-motoru -search 'apple -pear'      # results containing "pear" are excluded
./and-arama-motoru -search 'cat OR dog'       # either one (upper-case OR)
```

### Theme

The program **follows the theme selected in AND by itself**: if it has no `tema.json` of its own, it
reads AND's `theme.json` at startup (in `AND_DATA_DIR` if that is set, otherwise in AND's default
data directory) and uses the same colours. If you changed the theme in AND in Settings → Themes, the
Search Engine opens with the same theme on its next start. If AND isn't installed at all or no theme
was selected, the default colours match AND's own "Klasik" (Classic) theme exactly.

If you want to give the Search Engine a different theme from AND, create
`~/.and-arama-motoru/tema.json` by hand. If this file exists, it always takes priority. Write only
the fields you want to change; the rest keep their defaults:

```json
{
  "tema_adi": "Gece Mavisi",
  "accent": "39",
  "accent2": "86",
  "ok": "42",
  "warn": "220",
  "err": "203",
  "muted": "241",
  "sel_bg": "24",
  "sel_fg": "255",
  "rol": "33",
  "panel_bg": "236"
}
```

The default values (AND's Classic theme): `accent` 63, `accent2` 86, `ok` 78, `warn` 214, `err` 203,
`muted` 245, `sel_bg` 57, `sel_fg` 255, `rol` 33, `panel_bg` 236.

The values of AND's built-in themes are below; if you want to use a built-in theme other than the
one selected in AND only in the Search Engine, writing them to `tema.json` is enough:

| AND theme | `accent` | `sel_bg` | `sel_fg` | `accent2` | `rol` |
|---|---|---|---|---|---|
| Turkuaz (Turquoise) | 80 | 23 | 255 | 216 | 141 |
| Gece Moru (Night Purple) | 99 | 55 | 255 | 86 | 75 |
| Kehribar (Amber) | 214 | 94 | 230 | 180 | 117 |
| Orman (Forest) | 114 | 22 | 255 | 223 | 117 |
| Mono | 255 | 240 | 255 | 252 | 250 |

The values can be any colour string that lipgloss accepts: an ANSI 256 code point (`"63"`) or a
`"#RRGGBB"` true-colour hex. If `tema_adi` (theme name) is filled in, it appears on the top edge of
the frame next to the version number. The program reads this file on every start; if the file is
corrupt, it silently falls back to the default theme (without an error).

### Languages

The Search Engine opens in AND's six languages: Turkish, English, German, Spanish, French and
Russian. It doesn't choose a language itself but **follows the language selected in AND**: first the
`AND_DIL` environment variable, then `dil.json` in AND's data directory (Settings → Language in AND,
or `F2` on the login screen). If neither exists, it opens in Turkish. The language is read at
startup; if you changed the language in AND, the Search Engine opens with the new language on its
next start. To try a specific language: `AND_DIL=en and-arama-motoru`.

The translations cover the TUI, the `-h` help, the `-search`/`-site` output and the error messages.
These are not translated:

- Flag names and values (`-tip konu`, `-log-level bilgi`), the fields of the `-json` output and `gunluk.log`: scripts and debugging read them.
- Forum categories travel over the network in Turkish and are shown in the active language on the screen.

The translations use AND's `dil` package (through `replace ../AND`). Texts that are already in AND's
own catalog ("Durum", "Peer ID", category and theme names) come from there; those specific to the
Search Engine are in `ceviri.json`. The tests check two things:

- `TestCeviriKapsami`: that every text used has a translation in every language and that the format verbs (`%s`, `%d`, `%w`) are kept. `DIL_ISKELET=<file> go test -run TestCeviriKapsami` writes the missing ones as a skeleton.
- `TestEkranlarHerDilde`: that every screen is drawn in every language at 80×24 and 120×40, that no untranslated Turkish text is left and that no line overflows or is cut off. `ARAMA_EKRAN_DOKUM=<directory>` writes the screens as plain text for visual review.

## Usage

Pressing `f1` on any screen opens a help screen that shows ALL shortcuts in this table and the
advanced query syntax INSIDE the application (`esc` returns to the screen you were on, and the
query/code you were typing isn't lost). `f1` is deliberately a key that doesn't produce text: while
you type in the Search/Open by Code boxes, the shortcut itself never ends up in the query.

| Screen | Description |
|-------|----------|
| Search | Searches the local index instantly with every keystroke; `tab` filters by content type (All/Topic/Reply/Chat/Site); `ctrl+n` asks the network (multi-hop); `ctrl+r` changes the sorting (Relevance ⇄ Newest); `ctrl+t` follows the current search; `ctrl+y` clears the active author filter; `pgup`/`pgdown` move between pages; if there are zero results and there is a "did you mean?" suggestion, `ctrl+d` applies it |
| Open by Code | Paste a site share code (`authorKey/slug`) you received from someone and go directly to that page with `enter` (if it isn't available locally and there is a connected peer, it searches the network automatically too); if you paste only an `authorKey`, it lists all site pages of that author (the site map) — see [Sharing site pages](#sharing-site-pages) |
| Followed | Lists the searches saved with "follow"; every row shows the number of unread new matches; `enter` loads a search and switches to the Search screen (also resetting the badge), `d` deletes it |
| Result Details | `a` switches to a search filtered by the selected result's author ("this author's other content"); on network results `t` targets that peer directly; on site pages the "Code" row shows the share code (`c` copies it to the clipboard, `k` shows/hides its QR code), and the "Link" row shows the clickable `and-site://` link of the same code; `g` follows the internal `and-site://` links in the body — see [Sharing site pages](#sharing-site-pages) |
| Status | Your own peer ID, the number of connected peers, the number of records in the index; plus a "search console" panel that shows the content type breakdown, the most active authors, the daily content trend of the last 14 days and the trust indicator of the peers queried directly |

## Data directory files

| File | Contents |
|-------|--------|
| `kimlik.json` | This node's BIP39 mnemonic (network identity) |
| `arama_motoru.db` | SQLite FTS5 search index |
| `bilinen_peerler.json` | Last known peer addresses for fast reconnecting |
| `tema.json` | *(optional)* Custom colour theme — see [Theme](#theme) |
| `gunluk.log` | Diagnostic log |

## Scope and limits

- **It is a read-only peer** — it doesn't create posts/messages/site pages, doesn't apply the moderation chain (founder/moderator certificates, `mod_delete`) and doesn't answer other peers' forum/chat/site sync requests (it answers only its own search protocol).
- **"Search the network" reaches at most `aramaVarsayilanTTL` (3 by default) hops away** — it doesn't spread forever (a deliberate limit: both to keep the traffic reasonable and to stop a query from circulating on the network forever), and only peers that run another copy of this SAME program can answer (AND itself doesn't speak this protocol).
- **NAT/relay/hole-punching support is more limited than AND's** (see the doc comment of `network.go`) — if you are behind a strict NAT, outgoing connections still work, but being reachable for incoming "search the network" queries may not be as reliable as with AND.
- **The signature verification logic was transcribed by hand from AND's source code** (because it can't be imported) — the doc comments of `forum.go`/`chat.go` say so explicitly; the tests prove its internal consistency (sign→verify, tamper detection).
- **Kademlia DHT discovery is exposed to a known weakness shared by all Kademlia-based systems (including IPFS)** ([GO-2024-3218](https://pkg.go.dev/vuln/GO-2024-3218) / CVE-2023-26248): malicious nodes can manipulate the DHT and make specific content harder to find ("content censorship"). This is a PROTOCOL-LEVEL limit of the `go-libp2p-kad-dht` library — there is no fix yet. In this program its impact is partly reduced: discovery doesn't rely only on the DHT; mDNS (local network) and the known peer list (`peers.go`) provide independent discovery paths; and since AND itself uses the SAME library, this risk is shared by the whole ecosystem and isn't specific to this program.
- **The targeted P2P query (the `t` key) needs the target peer's address to be in this node's peerstore** — if a peer that was seen only INDIRECTLY in a multi-hop chain (never connected to directly) is targeted, dialling may fail; in that case a normal "stream couldn't be opened" error is shown.
- **The peer trust indicator (Status screen) lives as long as the process** — it is not written to disk, resets on every restart and is NOT a blocking mechanism, only a visual hint (the actual security is provided by the signature verification in `dogrulaAgSonucu`).
- **Saved-search follows ("follow") evaluate only the content this node SEES** — it doesn't scan the network proactively in the background; a match is counted only when the relevant content reaches this node normally through gossip/sync.

## Ecosystem

The AND Search Engine is one of the three independent parts of the AND ecosystem — all three are
separate programs that speak **the same network** (the same GossipSub topics, the same Ed25519
signature format, the same sync protocols) but run independently of each other:

| Project | What it does |
|-------|--------------|
| **[AND](https://github.com/AND-community/AND)** | The actual P2P terminal forum/chat application that forms the network itself — content can't be published on the network without a node |
| **AND Search Engine** *(this project)* | A read-only peer that indexes and searches the content on the network independently |
| **[and-vscode](https://github.com/AND-community/AND/tree/main/and-vscode)** | An editor extension that offers a live Forum/DM/Chat/Network/Site panel inside VS Code through AND's plugin API — personal site pages are edited and published in the editor, and share codes are fully compatible with "Open by Code" here |

What shows that they belong to the same family: the same ANSI 256-colour palette and the same "AND"
block-letter logo, the same BIP-39/Ed25519 identity model, the same GossipSub/sync protocols. For
details, see the [Ecosystem](https://github.com/AND-community/AND#ecosystem) section of the main
project's README.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) (in Turkish) for bug reports, feature suggestions and code
contributions.

## Security

If you found a security vulnerability, please **don't** open a public issue — follow the reporting
process in [SECURITY.md](SECURITY.md) (in Turkish).

## License

This project is licensed under the [GNU Affero General Public License v3.0](LICENSE) — the same
license as AND itself.
