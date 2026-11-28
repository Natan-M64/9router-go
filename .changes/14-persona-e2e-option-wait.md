### test(e2e): wait for the key option before counting it
- `e2e/personaKeyScope.test.ts` counted the API-key `<option>`s immediately after
  the picker mounted, racing the key fetch; `main` failed on it (run
  `46151325057`, `Expected: 1, Received: 0`) while the same tree passed on a PR.
- The test now waits for the option to be `attached` before asserting the count.
  `attached`, not the default `visible`, because Playwright never reports an
  `<option>` as visible until the native dropdown opens.
- The assertion itself is unchanged: exactly one option carries the key name.
  Verified locally with `bun test e2e/personaKeyScope.test.ts` (4 pass, 1.8 s).
