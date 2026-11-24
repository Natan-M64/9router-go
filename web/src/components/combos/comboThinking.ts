const SUFFIX_RE = /\(([^)]+)\)$/

export const COMBO_THINKING_OPTIONS = [
  { value: '', label: 'Thinking: Auto' },
  { value: 'off', label: 'Thinking: Off' },
  { value: 'minimal', label: 'Thinking: Minimal' },
  { value: 'low', label: 'Thinking: Low' },
  { value: 'medium', label: 'Thinking: Medium' },
  { value: 'high', label: 'Thinking: High' },
  { value: 'xhigh', label: 'Thinking: XHigh' },
  { value: 'max', label: 'Thinking: Max' },
] as const

export function parseComboModelThinking(model: string): { base: string; level: string } {
  const m = model.match(SUFFIX_RE)
  if (!m) {
    return { base: model, level: '' }
  }
  const raw = m[1].trim().toLowerCase()
  const base = model.slice(0, m.index)
  if (raw === 'none') {
    return { base, level: 'off' }
  }
  return { base, level: raw }
}

export function applyComboModelThinking(model: string, level: string): string {
  const { base } = parseComboModelThinking(model)
  const clean = level.trim().toLowerCase()
  if (!clean || clean === 'auto') {
    return base
  }
  return `${base}(${clean})`
}
