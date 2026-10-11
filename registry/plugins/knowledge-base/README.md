# Knowledge Base Plugin

Organization knowledge articles: owners and admins write Markdown articles, every member reads and
searches them. Search ranks full-text matches and also finds substrings, so Chinese and other
languages without word spacing work.

```bash
cd api && go run ./cmd/fastbuild plugin:add knowledge-base
```

It installs the `knowledge` starter (API module, migration, contract) and needs the `organization`
starter. Follow `INTEGRATION.md` after installing. The full contract ships as
`contracts/KNOWLEDGE.md`.
