# Permission Retention

Read this reference only when the downstream app retains the optional
`permission` starter.

- Retain `organization` in both API and Web optional selections.
- Extend the code-owned dotted permission catalog at assembly time.
- Authorize only from the typed active organization context.
- Keep membership roles separate from access roles and API key scopes.
- Product modules continue to own resource-instance policies.
- Never keep permission-management UI without API enforcement or replace exact
  authorization checks with hidden buttons.
