### fix(antigravity): a Google ToS ban switches the account off and keeps the reason

- A `403 TOS_VIOLATION` from Antigravity now disables the connection, so the selector stops choosing it for every model. Before, the refusal only locked the one model, and the account stayed active.
- The reason is stored on the connection (`lastError` and `accountDisabledReason`) and shows on the provider card. It is a fixed sentence of ours: the upstream body, and any appeal link in it, is never copied.
- Other providers are not affected: the rule matches the Antigravity providers only.
