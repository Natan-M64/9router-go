import { describe, expect, test } from 'bun:test'
import {
  buildCombosFromImport,
  exportCombosToJson,
  importMatchRules,
  parseCombosImport,
  resolveImportProvider,
} from './comboImportExport'

describe('comboImportExport', () => {
  test('resolves exact and OmniRoute provider aliases and suggests closest for unknown', () => {
    expect(resolveImportProvider('agy')).toMatchObject({
      status: 'exact',
      resolvedAlias: 'ag',
    })
    expect(resolveImportProvider('codex')).toMatchObject({
      status: 'exact',
      resolvedAlias: 'cx',
    })
    expect(resolveImportProvider('kilo-gateway')).toMatchObject({
      status: 'exact',
      resolvedAlias: 'kgw',
    })
    expect(resolveImportProvider('opencode-go')).toMatchObject({
      status: 'exact',
      resolvedAlias: 'ocg',
    })

    const unknown = resolveImportProvider('deepsek')
    expect(unknown.status).toBe('suggested')
    expect(unknown.candidates).toContain('ds')
  })

  test('parses OmniRoute combos JSON including combo-ref and system_message', () => {
    const raw = JSON.stringify({
      _meta: { version: 'omniroute-combos-v1' },
      combos: [
        {
          name: 'context-compactor',
          strategy: 'priority',
          system_message: 'Summarize cleanly.',
          models: [
            { kind: 'model', model: 'agy/gemini-4-hydrogen-medium', providerId: 'agy' },
            { kind: 'model', model: 'codex/gpt-6-luna', providerId: 'codex' },
            { kind: 'combo-ref', comboName: 'scout-free' },
          ],
        },
      ],
    })

    const parsed = parseCombosImport(raw)
    expect(parsed.format).toBe('omniroute')
    expect(parsed.combos).toHaveLength(1)

    const built = buildCombosFromImport(parsed, {})
    expect(built[0]).toEqual({
      name: 'context-compactor',
      strategy: 'fallback',
      customPrompt: 'Summarize cleanly.',
      models: ['ag/gemini-4-hydrogen-medium', 'cx/gpt-6-luna', 'scout-free'],
    })
  })

  test('exports 9router combos and round-trips cleanly', () => {
    const json = exportCombosToJson([
      {
        id: '1',
        name: 'my-combo',
        models: ['cx/gpt-6.2-sol(high)', 'cc/claude-opus-5-5'],
        strategy: 'fallback',
        matchRules: '[{"id":"r1","pattern":"coding","mode":"keyword","effect":"include","enabled":true,"maxMembers":20,"allKinds":false}]',
      },
    ])
    const parsed = parseCombosImport(json)
    expect(parsed.format).toBe('9router')
    const built = buildCombosFromImport(parsed, {})
    expect(built[0].models).toEqual(['cx/gpt-6.2-sol(high)', 'cc/claude-opus-5-5'])
    // The stored JSON string survives the round trip unchanged; the API parses it.
    expect(built[0].matchRules).toBe(
      '[{"id":"r1","pattern":"coding","mode":"keyword","effect":"include","enabled":true,"maxMembers":20,"allKinds":false}]',
    )
  })

  test('importMatchRules turns bare keywords into keyword include rules', () => {
    const rules = importMatchRules(['coding', '  ', 'review'])
    expect(Array.isArray(rules)).toBe(true)
    const arr = rules as { pattern: string; mode: string; effect: string; maxMembers: number }[]
    expect(arr.map((r) => r.pattern)).toEqual(['coding', 'review'])
    expect(arr.every((r) => r.mode === 'keyword' && r.effect === 'include')).toBe(true)
    expect(arr[0].maxMembers).toBe(20)
  })

  test('importMatchRules passes a JSON string or rule objects through', () => {
    expect(importMatchRules('[{"pattern":"x"}]')).toBe('[{"pattern":"x"}]')
    expect(importMatchRules('   ')).toBeUndefined()
    expect(importMatchRules(undefined)).toBeUndefined()
    const obj = { id: 'r9', pattern: 'a*', mode: 'glob', effect: 'exclude', enabled: true, maxMembers: 0, allKinds: true }
    expect(importMatchRules([obj])).toEqual([obj])
  })
})
