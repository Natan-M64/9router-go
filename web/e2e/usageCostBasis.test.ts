/**
 * The Est. Cost card must separate billed spend from list-price cost on a plan.
 *
 * `cost` is a list-price estimate for every provider. Summing it unlabelled
 * overstates the bill by the price of every subscription and free-quota request:
 * on this install Codex and Antigravity alone were $26.85 of a $30.33 "spend"
 * total that was never invoiced. The server now classifies each provider and
 * reports `totalCostMetered` / `totalCostIncluded` / `totalCostUnclassified`
 * apart from `totalCost`, plus a `billing` field per provider row.
 *
 * Nothing below the browser sees this: the JSON carries the numbers, `tsc`
 * checks the types, and the page renders. Only a browser can prove the split
 * reaches the card and the provider table instead of staying a field nobody
 * draws. The fixture is fixed, so the assertion is about the DOM.
 */
import { afterAll, beforeAll, describe, expect, test } from 'bun:test'
import type { Dashboard } from './harness'
import { startDashboard } from './harness'

const SLOW = 20_000

const STATS_FIXTURE = {
  totalRequests: 8,
  totalPromptTokens: 10_300,
  totalCompletionTokens: 330,
  totalCachedTokens: 0,
  totalCost: 1.3,
  totalCostMetered: 0.07,
  totalCostIncluded: 1.2,
  totalCostUnclassified: 0.03,
  byProvider: {
    deepseek: {
      requests: 5,
      promptTokens: 1000,
      completionTokens: 100,
      cachedTokens: 0,
      cost: 0.07,
      billing: 'metered',
    },
    codex: {
      requests: 2,
      promptTokens: 9000,
      completionTokens: 200,
      cachedTokens: 0,
      cost: 1.2,
      billing: 'included',
    },
    'openai-compatible-local': {
      requests: 1,
      promptTokens: 300,
      completionTokens: 30,
      cachedTokens: 0,
      cost: 0.03,
      billing: 'unknown',
    },
  },
  byModel: {},
  byAccount: {},
  byApiKey: {},
  byEndpoint: {},
  activeRequests: [],
  recentRequests: [],
  errorProvider: '',
  pending: {},
}

let app: Dashboard

beforeAll(async () => {
  app = await startDashboard(20455)
  await app.page.route('**/api/usage/stats**', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(STATS_FIXTURE),
    }),
  )
  await app.page.route('**/api/usage/dashboard', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ daily: [], nodes: [], models: [], lifetime: null }),
    }),
  )
  await app.page.route('**/api/usage/request-details**', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ details: [], total: 0, limit: 20, offset: 0 }),
    }),
  )
  await app.page.route('**/api/usage/chart**', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ points: [] }),
    }),
  )
}, SLOW)

afterAll(async () => {
  await app.stop()
})

describe('usage cost basis', () => {
  test(
    'the Est. Cost card splits billed from included',
    async () => {
      const page = app.page
      await page.goto(`${app.baseURL}/dashboard/usage`)
      await page.getByText('Usage & Analytics').waitFor({ timeout: SLOW })
      await page.getByText(/\$0\.07 billed/).first().waitFor({ timeout: SLOW })
      const subtext = await page.getByText(/\$0\.07 billed/).first().textContent()
      expect(subtext).toContain('$1.20 included')
      expect(subtext).toContain('$0.03 unclassified')
    },
    SLOW,
  )

  test(
    'each provider row carries its billing label',
    async () => {
      const page = app.page
      await page.goto(`${app.baseURL}/dashboard/usage`)
      await page.getByText('Usage & Analytics').waitFor({ timeout: SLOW })
      await page.selectOption('select:has(option[value="provider"])', 'provider')
      await page.getByText('Metered', { exact: true }).first().waitFor({ timeout: SLOW })
      await page.getByText('Included', { exact: true }).first().waitFor({ timeout: SLOW })
      await page.getByText('Unclassified', { exact: true }).first().waitFor({ timeout: SLOW })
    },
    SLOW,
  )
})
