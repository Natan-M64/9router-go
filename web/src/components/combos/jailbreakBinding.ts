// The combo editor's "Prompt injection" choice is stored as an exact jailbreak
// binding on the combo's name: binding a combo name arms every leg of that combo,
// and removing the binding turns it off again. Nothing is injected otherwise
// (keyword matching is off by default), so this choice is the only switch a
// user has to flip.

/** '' means off; any other value is a preset id ('default' is the default document). */
export type JailbreakChoice = string

export interface JailbreakChoiceChange {
  /** The choice the editor opened with. */
  initial: JailbreakChoice
  /** The choice the editor saves. */
  choice: JailbreakChoice
}

export type JailbreakOp =
  | { op: 'set'; id: string; presetId: string }
  | { op: 'remove'; id: string }

/**
 * jailbreakChoiceFor reads a combo's current choice from the bindings map the
 * API returns. A binding that exists but is switched off counts as off.
 */
export function jailbreakChoiceFor(
  bindings: Record<string, { enabled: boolean; presetId: string }>,
  comboName: string,
): JailbreakChoice {
  const cfg = bindings[comboName]
  if (!cfg || !cfg.enabled) return ''
  return cfg.presetId || 'default'
}

/**
 * planJailbreakChange turns an edit into the writes it needs, and no more: a
 * save that changes nothing writes nothing, and a renamed combo takes its
 * binding with it instead of leaving one under the old name.
 */
export function planJailbreakChange(
  previousName: string | null,
  name: string,
  change: JailbreakChoiceChange,
): JailbreakOp[] {
  const renamed = previousName !== null && previousName !== name
  if (!renamed && change.choice === change.initial) return []

  const ops: JailbreakOp[] = []
  if (renamed && change.initial !== '') {
    ops.push({ op: 'remove', id: previousName as string })
  }
  if (change.choice !== '') {
    ops.push({ op: 'set', id: name, presetId: change.choice })
  } else if (!renamed && change.initial !== '') {
    ops.push({ op: 'remove', id: name })
  }
  return ops
}
