# Auto Tagging Rules

This document defines the rules for automatic tagging of services when a tag
declares `auto: true`.

## Core Rules

- `auto: true` is only valid on explicitly declared tags.
- With `auto: true`, `element_type` is required. If missing, generation fails.
- With `auto: true`, `element_type` must be an interface type. Non-interface
  types cause a generation error.
- Auto tagging runs after `DecoratorPass`, so it sees the final decorated
  service graph.
- Only real services are candidates. Aliases and `.inner` services are excluded.
- For decorator chains, only the outermost decorator can be auto-tagged.
- A service is auto-tagged if its final type is `AssignableTo` the tag's
  `element_type`.
- Explicit tags remain; auto-tagging only adds missing services (deduped by
  service ID).
- Result ordering is deterministic (stable order by service ID).
- `auto: true` cannot be combined with `sort_by` and must be rejected.
- An auto-tag may end up empty; this is allowed.

## Implementation Note (Temporary)

Auto-tagging currently excludes decorator inner services by checking the
`.inner` suffix. This is a temporary rule until decorator metadata is available
in IR to avoid name-based filtering.

## Decorator Example

Given:

```yaml
tags:
  handlers:
    element_type: "github.com/acme/app.Handler"
    auto: true

services:
  base:
    constructor:
      func: "github.com/acme/app.NewBaseHandler"
  decorated:
    constructor:
      func: "github.com/acme/app.NewDecoratedHandler"
      args:
        - "@.inner"
    decorates: "base"
    decoration_priority: 10
```

Only `decorated` is auto-tagged (if its final type implements `Handler`).
`base` is not auto-tagged, and `.inner` services never participate.
