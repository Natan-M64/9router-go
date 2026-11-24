import { describe, expect, test } from 'bun:test'
import { applyComboModelThinking, parseComboModelThinking } from './comboThinking'

describe('comboThinking', () => {
  test('parses plain model and suffixed model', () => {
    expect(parseComboModelThinking('cx/gpt-6.2-sol')).toEqual({
      base: 'cx/gpt-6.2-sol',
      level: '',
    })
    expect(parseComboModelThinking('cx/gpt-6.2-sol(high)')).toEqual({
      base: 'cx/gpt-6.2-sol',
      level: 'high',
    })
    expect(parseComboModelThinking('cc/claude-opus-5-5(none)')).toEqual({
      base: 'cc/claude-opus-5-5',
      level: 'off',
    })
  })

  test('applies and clears thinking suffix cleanly', () => {
    expect(applyComboModelThinking('cx/gpt-6.2-sol', 'xhigh')).toBe('cx/gpt-6.2-sol(xhigh)')
    expect(applyComboModelThinking('cx/gpt-6.2-sol(high)', 'low')).toBe('cx/gpt-6.2-sol(low)')
    expect(applyComboModelThinking('cx/gpt-6.2-sol(high)', '')).toBe('cx/gpt-6.2-sol')
    expect(applyComboModelThinking('cx/gpt-6.2-sol(high)', 'auto')).toBe('cx/gpt-6.2-sol')
  })
})
