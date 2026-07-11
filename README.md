# x-archive-parser

`x-archive-parser` expands an X (formerly Twitter) archive into per-month
files that coding agents can investigate.

- Takes the archive zip downloaded from X, or a directory it was extracted to
- Expands **every post** — including reposts and replies — with t.co URLs resolved
- Writes one Markdown file (for reading) and one JSONL file (for grep/jq) per month
- Generates an `index.md` entry point with per-month counts and usage notes
- Runs fully offline, using only the Go standard library

## Install

```sh
go install github.com/syumai/x-archive-parser@latest
```

Requires Go 1.26+.

## Usage

```
x-archive-parser [flags] [archive]

  archive is the path to the archive zip file or extracted directory; it
  defaults to the current directory if omitted. Flags must be given before
  the archive argument.

Flags:
  -o string         output directory (default "./posts")
  -tz string        IANA timezone name (e.g. Asia/Tokyo, UTC) used for month
                     boundaries and displayed times; defaults to the system
                     local timezone
```

Examples:

```sh
# Simplest: run inside an extracted archive directory; writes to ./posts.
x-archive-parser

# From the zip file downloaded from X.
x-archive-parser ./twitter-2023-01-01-xxxxxxxx.zip

# Choose the output directory and timezone explicitly.
x-archive-parser -o ./posts -tz Asia/Tokyo ./twitter-2023-01-01-xxxxxxxx
```

Re-running the tool with the same `-o` directory is idempotent: it overwrites
its own previous output. It refuses to write into a non-empty directory that
wasn't produced by a previous run (i.e. one without an `index.md`), so it
won't clobber an unrelated directory by mistake.

### Output layout

```
posts/
├── index.md                # entry point: account info, date range, per-month table
├── 2020/
│   ├── 2020-01.md
│   ├── 2020-01.jsonl
│   └── ...
├── ...
└── 2023/
    ├── 2023-12.md
    └── 2023-12.jsonl
```

Posts are split into one Markdown file and one JSONL file per calendar
month, grouped under a year directory, so a month can be located from its
file name alone. Within each month, posts are ordered oldest to newest.

`index.md` looks like this:

```markdown
# X post archive of @example

- Range: 2020-01-05 to 2023-12-28 (Asia/Tokyo)
- Total: 12,345 posts (post 6,789 / repost 2,345 / reply 3,211)

## How to use this archive
...

## Posts per month

| Month | Total | post | repost | reply |
|---|---|---|---|---|
| [2020-01](2020/2020-01.md) | 42 | 30 | 7 | 5 |
| ... |
```

A monthly Markdown file (`2023/2023-06.md`) groups posts under date
headings, with each post as a short, timestamped bullet:

```markdown
# 2023-06 (@example)

98 posts (post 60 / repost 18 / reply 20)

## 2023-06-01 (Thu)

- **09:12 post** https://twitter.com/example/status/1000000000000000001
  Trying out the new WebAssembly runtime. https://example.com/article
- **12:45 repost** (RT of @someone) https://twitter.com/example/status/1000000000000000002
  A new release of the toolchain is out…
```

The matching JSONL file (`2023/2023-06.jsonl`) has one post per line, so a
single `grep` hit already carries the timestamp, type, URL, and text:

```json
{"id":"1000000000000000001","url":"https://twitter.com/example/status/1000000000000000001","created_at":"2023-06-01T09:12:34+09:00","type":"post","text":"Trying out the new WebAssembly runtime. https://example.com/article","lang":"en","favorite_count":3,"retweet_count":1}
```

## Use with coding agents

The output is designed to be investigated by coding agents: an agent can be
asked something like "summarize what I posted about WebAssembly between
March and April" and answer it by browsing the index, grepping a handful of
JSONL files, and reading the matching Markdown for context — without needing
API access, a database, or the original archive UI.

Point an agent at the output directory and tell it to start from
`index.md`, which explains the layout and lists per-month counts. From
there the agent can grep the JSONL files to find candidate months, then
read those months' Markdown files for surrounding context.

Example prompt:

> Look at `./posts/index.md` to understand the archive layout, then
> summarize what I posted about WebAssembly between March and April 2023,
> including reposts. Use the JSONL files to search and the Markdown files
> to read context around any hits.

Example `grep` an agent might run to find candidate months before reading
the Markdown:

```sh
grep -il wasm posts/*/[0-9]*.jsonl
```

## Notes and limitations

- **Truncated reposts**: the archive does not store the full text of
  retweets — only up to roughly the first 140 characters, ending in `…`.
  This is a limitation of the source data, not of this tool.
- **t.co expansion**: `t.co` short URLs in post text are replaced with
  their expanded form whenever the archive records the mapping
  (`entities.urls[].expanded_url`); URLs the archive doesn't have a mapping
  for are left as-is.
- **Long-form (Note Tweet) posts**: the archive stores long-form post text
  separately from the truncated version in the main tweet data; it is
  matched back to the corresponding post and restored on a best-effort
  basis.
- **Timestamps**: all displayed times and month/day boundaries use the
  selected timezone — the system's local timezone by default, or the zone
  passed via `-tz`. The timezone actually used is recorded in `index.md` so
  an agent doesn't need to guess.
- **Offline**: this tool never makes network requests; it only reads the
  archive on disk.

## Acknowledgements

Inspired by [timhutton/twitter-archive-parser](https://github.com/timhutton/twitter-archive-parser).
Unlike that project, `x-archive-parser` doesn't aim to produce a
human-browsable HTML archive — it targets structured, greppable output for
coding agents.

## License

MIT — see [LICENSE](LICENSE).
