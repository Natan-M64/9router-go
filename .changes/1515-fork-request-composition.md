### feat(usage): a request record shows what the client sent, by role and by tool

- Each successful request record now has `composition`: the message count, the bytes per role (system, user, assistant, tool), the bytes per tool name (the largest twelve, the rest under `other`), and two fingerprints of the prefix, one for the system messages and one for the tools definition.
- No text is stored. The fingerprints let two turns of one conversation be compared: when they match, the prompt cache can reuse the prefix; when they differ, the record shows which part moved.
