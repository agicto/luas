# Knowledge Base Plugin

Organization knowledge articles: owners and admins write Markdown articles, every member reads and
searches them. Search ranks full-text matches and also finds substrings, so Chinese and other
languages without word spacing work.

```bash
cd api && go run ./cmd/fastbuild plugin:add knowledge-base
```

It installs, and `plugin:remove` takes away again:

- the `knowledge` API starter: module, migration, contract (`contracts/KNOWLEDGE.md` and the
  OpenAPI fragment); it needs the `organization` starter;
- the `knowledge` Web feature: a Knowledge console page (`/console/knowledge`) with cursor-paged
  list, search, article view, and owner/admin create, edit, and delete; browser routes that use
  the Go API adapter or the mock BFF; en-US and zh-Hans messages; and its tests. It needs the
  `organization` Web feature and is wired into the console navigation, routes, i18n, and API error
  codes through slots.

Follow `INTEGRATION.md` after installing. There is no Admin Console view: operators do not read
tenant content.
