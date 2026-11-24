### feat(combos): JSON and OmniRoute import/export, and a render crash fix

- Combos can be imported from `9router-combos-v1` and `omniroute-combos-v1` files, with automatic or assisted mapping of provider abbreviations, and exported back.
- `matchRules` from a foreign file are normalized before they reach the API: a bare keyword, or a JSON array of keywords, becomes a keyword include rule; a JSON array of rule objects is kept as the original string; a value with no usable rule is dropped instead of being sent.
- `/dashboard/combos` no longer crashes while rendering: the import modal passed a callback and a size prop under the wrong names.
