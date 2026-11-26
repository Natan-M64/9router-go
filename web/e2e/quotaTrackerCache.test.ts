/**
 * Reopening the Quota Tracker must show the last known quotas immediately.
 *
 * The view is mounted under an `{#if}` in App.svelte, so switching tabs
 * unmounts it and coming back re-runs the whole fan-out. It also writes every
 * answer to localStorage (`quotaCacheData`) but never read it back, so the
 * visited data was there while the page still painted a spinner until every
 * gated upstream read finished — with the gateway pacing quota reads 250 ms
 * apart, that is seconds for a page of accounts.
 *
 * The regression is invisible to `tsc`, `vite build`, the svelte-check ratchet
 * and `go test`: the JSON is correct and the component compiles. The assertion
 * is therefore the rendered DOM on a second visit whose network read is held
 * for longer than the assertion timeout — the row can only come from the cache.
 */
import { afterAll, beforeAll, describe, test } from 'bun:test'
import type { Dashboard } from './harness'
import { startDashboard } from './harness'

const SLOW = 30_000
const CACHED_WAIT = 2_500
const DELAYED_RESPONSE_MS = 6_000

const QUOTA_FIXTURE = {
  plan: 'Venice AI',
  quotas: {
    diem: {
      name: 'DIEM (daily epoch)',
      displayName: 'DIEM (daily epoch)',
      used: 9.5,
      total: 100,
      remainingPercentage: 90.5,
      resetAt: '2099-01-01T00:00:00Z',
      recurring: true,
    },
    usd: {
      name: 'USD credits',
      displayName: 'USD credits',
      used: 0,
      total: 25,
      remainingPercentage: 100,
      isCreditBalance: true,
      currency: 'USD',
    },
  },
}

let app: Dashboard

beforeAll(async () => {
  app = await startDashboard(20461)
  const created = await app.page.request.post(`${app.baseURL}/api/connections`, {
    data: { provider: 'venice', authType: 'apikey', name: 'E2E Venice', apiKey: 'e2e-key' },
  })
  if (!created.ok()) {
    throw new Error(`create connection failed: ${created.status()} ${await created.text()}`)
  }
  let calls = 0
  // Only the per-connection quota read is intercepted; `/api/usage/stats` and
  // friends are untouched. The first answer is immediate so the first visit
  // populates the cache; every later one is slow.
  await app.page.route(
    (url) => /^\/api\/usage\/[0-9a-f-]{36}$/.test(url.pathname),
    async (route) => {
      calls += 1
      if (calls > 1) await new Promise((resolve) => setTimeout(resolve, DELAYED_RESPONSE_MS))
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(QUOTA_FIXTURE) })
    },
  )
}, SLOW)

afterAll(async () => {
  await app?.stop()
})

describe('quota tracker cache', () => {
  test(
    'a revisit renders the last known quotas before the network answers',
    async () => {
      const page = app.page
      await page.goto(`${app.baseURL}/dashboard/quota`)
      await page.getByText('DIEM (daily epoch)').first().waitFor({ timeout: SLOW })
      await page.getByText('Credit: 25.00 USD').first().waitFor({ timeout: SLOW })

      // A full reload, not a tab switch: it also proves the snapshot survives
      // the SPA booting again.
      await page.goto(`${app.baseURL}/dashboard/quota`)
      await page.getByText('DIEM (daily epoch)').first().waitFor({ timeout: CACHED_WAIT })
      await page.getByText('Credit: 25.00 USD').first().waitFor({ timeout: CACHED_WAIT })
    },
    SLOW,
  )
})
