# Comments Plugin

Threaded comments on any organization resource: members comment and reply, authors edit their own
comments, and authors or organization owners and admins delete them. Threads are one level deep and
deleted comments keep their place with the text erased. Mentioned members receive a notification
when the `notification` starter is enabled; without it, mentions are stored silently.

```bash
cd api && go run ./cmd/fastbuild plugin:add comments
```

It installs the `comments` starter (API module, migration, contract) and needs the `organization`
starter. The plugin ships no commentable resource types: each module registers its own, deciding
who may read and comment (see `INTEGRATION.md`). The full contract ships as `contracts/COMMENTS.md`.
