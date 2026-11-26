import { describe, expect, test } from 'bun:test'
import { parseQuotaData } from './types'

describe('venice quota parsing', () => {
  test('rows keep a fixed order regardless of the payload key order', () => {
    const data = {
      plan: 'Venice AI',
      quotas: {
        earned: { name: 'Earned credits', displayName: 'Earned credits', total: 2, isCreditBalance: true, currency: 'USD' },
        usd: { name: 'USD credits', displayName: 'USD credits', total: 25, isCreditBalance: true, currency: 'USD' },
        diem: {
          name: 'DIEM (daily epoch)',
          displayName: 'DIEM (daily epoch)',
          used: 9.5,
          total: 100,
          remainingPercentage: 90.5,
          resetAt: '2026-11-26T00:00:00Z',
        },
        bundled: { name: 'Bundled credits', displayName: 'Bundled credits', total: 10, isCreditBalance: true, currency: 'USD' },
      },
    }
    const rows = parseQuotaData('venice', data)
    expect(rows.map((row) => row.name)).toEqual([
      'DIEM (daily epoch)',
      'USD credits',
      'Bundled credits',
      'Earned credits',
    ])
  })

  test('a credit balance keeps its amount and currency', () => {
    const rows = parseQuotaData('venice', {
      quotas: { usd: { name: 'USD credits', displayName: 'USD credits', total: 3.25, isCreditBalance: true, currency: 'USD' } },
    })
    expect(rows).toHaveLength(1)
    expect(rows[0].isCreditBalance).toBe(true)
    expect(rows[0].total).toBe(3.25)
    expect(rows[0].currency).toBe('USD')
  })

  test('the DIEM epoch row stays a percentage with a reset', () => {
    const rows = parseQuotaData('venice', {
      quotas: {
        diem: {
          name: 'DIEM (daily epoch)',
          displayName: 'DIEM (daily epoch)',
          used: 9.5,
          total: 100,
          remainingPercentage: 90.5,
          resetAt: '2026-11-26T00:00:00Z',
        },
      },
    })
    expect(rows[0].remainingPercentage).toBe(90.5)
    expect(rows[0].isCreditBalance).toBeUndefined()
    expect(rows[0].resetAt).toBe('2026-11-26T00:00:00Z')
  })
})
