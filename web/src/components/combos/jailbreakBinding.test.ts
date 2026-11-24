import { describe, expect, test } from 'bun:test'
import { jailbreakChoiceFor, planJailbreakChange } from './jailbreakBinding'

describe('jailbreakChoiceFor', () => {
  test('is off when there is no binding, or the binding is switched off', () => {
    expect(jailbreakChoiceFor({}, 'main')).toBe('')
    expect(jailbreakChoiceFor({ main: { enabled: false, presetId: 'p' } }, 'main')).toBe('')
  })

  test('reads the preset, and the default document when none is named', () => {
    expect(jailbreakChoiceFor({ main: { enabled: true, presetId: 'p' } }, 'main')).toBe('p')
    expect(jailbreakChoiceFor({ main: { enabled: true, presetId: '' } }, 'main')).toBe('default')
  })
})

describe('planJailbreakChange', () => {
  test('writes nothing when nothing changed', () => {
    expect(planJailbreakChange('main', 'main', { initial: '', choice: '' })).toEqual([])
    expect(planJailbreakChange('main', 'main', { initial: 'p', choice: 'p' })).toEqual([])
    expect(planJailbreakChange(null, 'new', { initial: '', choice: '' })).toEqual([])
  })

  test('turning it on, switching preset and turning it off', () => {
    expect(planJailbreakChange('main', 'main', { initial: '', choice: 'p' })).toEqual([
      { op: 'set', id: 'main', presetId: 'p' },
    ])
    expect(planJailbreakChange('main', 'main', { initial: 'p', choice: 'q' })).toEqual([
      { op: 'set', id: 'main', presetId: 'q' },
    ])
    expect(planJailbreakChange('main', 'main', { initial: 'p', choice: '' })).toEqual([
      { op: 'remove', id: 'main' },
    ])
  })

  test('a new combo only writes when a preset was chosen', () => {
    expect(planJailbreakChange(null, 'new', { initial: '', choice: 'p' })).toEqual([
      { op: 'set', id: 'new', presetId: 'p' },
    ])
  })

  test('a renamed combo moves its binding instead of leaving one behind', () => {
    expect(planJailbreakChange('old', 'new', { initial: 'p', choice: 'p' })).toEqual([
      { op: 'remove', id: 'old' },
      { op: 'set', id: 'new', presetId: 'p' },
    ])
    // Renamed and switched off: only the old binding goes.
    expect(planJailbreakChange('old', 'new', { initial: 'p', choice: '' })).toEqual([
      { op: 'remove', id: 'old' },
    ])
    // Renamed with nothing armed: nothing to move.
    expect(planJailbreakChange('old', 'new', { initial: '', choice: '' })).toEqual([])
  })
})
