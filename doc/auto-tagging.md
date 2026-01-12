# Auto Tagging Rules

This document defines the rules for automatic tagging of services when a tag
declares `autoconfigure: true`.

## Core Rules

- `autoconfigure: true` is only valid on explicitly declared tags.
- With `autoconfigure: true`, `element_type` is required. If missing, generation fails.
- With `autoconfigure: true`, `element_type` must be an interface type. Non-interface
  types cause a generation error.
- Auto tagging runs after `DecoratorPass`, so it sees the final decorated
  service graph.
- Only real services are candidates. Aliases are excluded.
- Decorator inner services are created with `autoconfigure: false` and do not participate.
- A service is auto-tagged if its final type is `AssignableTo` the tag's
  `element_type`.
- Explicit tags remain; auto-tagging only adds missing services (deduped by
  service ID).
- Result ordering is deterministic (stable order by service ID).
- `autoconfigure: true` cannot be combined with `sort_by` and must be rejected.
- Services can opt out via `autoconfigure: false` (or in `_default`).
- An auto-tag may end up empty; this is allowed.

## Decorator Example

Given:

```yaml
tags:
  handlers:
    element_type: "github.com/acme/app.Handler"
    autoconfigure: true

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
