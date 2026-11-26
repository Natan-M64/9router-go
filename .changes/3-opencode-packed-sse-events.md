### Streaming: packed SSE data events are split into separate events
- The `ocg/deepseek-v4.1-flash-free` stream packs the tool-call chunk and the
  terminal `finish_reason:"tool_calls"` chunk with a single newline between
  them. Per SSE those two `data:` lines are one event, so a strict client joined
  both JSON objects and failed with `JSON parsing failed` (seen in Kilo). The
  OpenAI passthrough now inserts a blank separator before a `data:` line that
  follows another one with no blank line since. Only a blank line resets the
  boundary, already-separated streams are unchanged, and the split is generic
  framing with no provider-specific branch.
- Pinned by `TestSSECopy_SplitsPackedDataEvents` (packed, separated, CRLF,
  comment-between, EOF and byte-at-a-time). Red on base, green with the fix.
