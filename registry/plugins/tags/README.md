# Tags Plugin

Organization tags: owners and admins create labels with a palette color and attach up to 20 of
them to any resource whose module registers a resource type; every member reads them. It is a
product-neutral building block that other modules and plugins tag their resources with.

```bash
cd api && go run ./cmd/fastbuild plugin:add tags
```

It installs the `tag` starter (API module, migration, contract) and needs the `organization`
starter. The starter ships with no resource types: follow `INTEGRATION.md` after installing, which
also shows how a module such as `knowledge` registers one. The full contract ships as
`contracts/TAGS.md`.
