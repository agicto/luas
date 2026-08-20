# Organization Retention

Read this reference only when the downstream app retains the optional
`organization` starter.

- Keep `organization` as the tenant/account term unless the product deliberately
  adds a separate child concept.
- Preserve member IDs as membership-resource IDs and keep email out of the
  member-directory response.
- Use the same `OPTIONAL_STARTERS` value in API replicas and migration jobs
  before attaching product resources or permission scopes.
- Select the Web feature with
  `NEXT_PUBLIC_OPTIONAL_FEATURES=organization`.
- Keep active organization selection request-scoped. Forward exactly one
  `Organization-Id` from the current browser tab or URL, apply the API's
  `organization_context` middleware after authentication, and authorize product
  data from the typed resolved context, never the raw header.
- Add `Organization-Id` to the production CORS allow-list when browsers call
  the API cross-origin.
- For a retained same-origin adapter, keep selection in the URL and extend only
  explicit Route Handlers. Never replace `src/server/api-adapter/` with a
  browser-controlled catch-all proxy.
