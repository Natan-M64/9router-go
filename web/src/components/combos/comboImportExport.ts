import { PROVIDER_ID_TO_ALIAS } from '../../lib/models'
import type { Combo, SmartRule } from '../../api/client'
import { DEFAULT_SMART_RULE_MAX_MEMBERS, getComboModels } from './types'

export type ImportFormat = 'omniroute' | '9router'

/**
 * Normalizes a combo's matchRules from an imported file into a shape the API
 * accepts. The backend (combos/smart DecodeRules) takes a JSON string of rule
 * objects or an array of rule objects. A bare keyword string is not a rule
 * object, so it becomes a keyword include rule with the default member cap.
 */
// normalizeRuleEntries turns each entry into a SmartRule: a bare string becomes a
// keyword include rule, an object with a string pattern is kept as-is, anything
// else is dropped.
function normalizeRuleEntries(entries: unknown[]): SmartRule[] {
  const rules: SmartRule[] = []
  entries.forEach((entry: unknown, i: number) => {
    if (typeof entry === 'string') {
      const pattern = entry.trim()
      if (!pattern) return
      rules.push({
        id: `r${i + 1}`,
        pattern,
        mode: 'keyword',
        effect: 'include',
        enabled: true,
        maxMembers: DEFAULT_SMART_RULE_MAX_MEMBERS,
        allKinds: false,
      })
      return
    }
    if (entry && typeof entry === 'object' && typeof (entry as { pattern?: unknown }).pattern === 'string') {
      rules.push(entry as SmartRule)
    }
  })
  return rules
}

// importMatchRules accepts the matchRules value of a combo in an import file. The
// backend needs either a JSON array of rule objects (the form a stored combo
// carries) or an array of rules; a bare keyword string is also accepted from
// foreign files. A JSON array that is already rule objects is returned as the
// original string, so the stored form round-trips unchanged. Anything the
// backend could not read is normalized here or dropped, never passed through raw.
export function importMatchRules(raw: unknown): string | SmartRule[] | undefined {
  if (typeof raw === 'string') {
    const text = raw.trim()
    if (!text) return undefined
    let parsed: unknown
    try {
      parsed = JSON.parse(text)
    } catch {
      parsed = undefined
    }
    if (!parsed || typeof parsed !== 'object') {
      // Not JSON, or a JSON scalar such as "123": a bare keyword such as "coding".
      const rules = normalizeRuleEntries([text])
      return rules.length ? rules : undefined
    }
    const entries: unknown[] = Array.isArray(parsed) ? parsed : [parsed]
    const allRuleObjects =
      entries.length > 0 &&
      entries.every((e) => !!e && typeof e === 'object' && typeof (e as { pattern?: unknown }).pattern === 'string')
    if (allRuleObjects) return text
    const rules = normalizeRuleEntries(entries)
    return rules.length ? rules : undefined
  }
  if (!Array.isArray(raw)) return undefined
  const rules = normalizeRuleEntries(raw)
  return rules.length ? rules : undefined
}

export interface ProviderMappingEntry {
  rawProvider: string
  resolvedAlias: string
  status: 'exact' | 'suggested' | 'unresolved'
  candidates: string[]
}

export interface ParsedImportStep {
  kind: 'model' | 'combo-ref'
  rawProvider?: string
  modelId: string
}

export interface ParsedImportCombo {
  name: string
  steps: ParsedImportStep[]
  strategy: string
  customPrompt?: string
  matchRules?: string | SmartRule[]
}

export interface ParsedCombosImport {
  format: ImportFormat
  combos: ParsedImportCombo[]
  providerMappings: ProviderMappingEntry[]
}

// Explicit OmniRoute -> 9Router-Go provider ID differences where the short
// prefix or registry name differs between the two projects.
const OMNIROUTE_TO_9ROUTER_ALIASES: Record<string, string> = {
  agy: 'ag',
  antigravity: 'ag',
  claude: 'cc',
  codex: 'cx',
  'kilo-gateway': 'kgw',
  kilocode: 'kgw',
  'opencode-go': 'ocg',
  'opencode-zen': 'oc',
  opencode: 'oc',
  'gemini-cli': 'gc',
  github: 'gh',
  commandcode: 'cmc',
  opendesign: 'od',
}

export function getKnownProviderOptions(): Array<{ alias: string; label: string }> {
  const seen = new Map<string, string>()
  for (const [id, alias] of Object.entries(PROVIDER_ID_TO_ALIAS)) {
    const existing = seen.get(alias)
    if (!existing || id.length > existing.length) {
      seen.set(alias, id === alias ? alias : `${id} (${alias})`)
    }
  }
  return Array.from(seen.entries())
    .map(([alias, label]) => ({ alias, label }))
    .sort((a, b) => a.label.localeCompare(b.label))
}

function similarityScore(a: string, b: string): number {
  const x = a.toLowerCase()
  const y = b.toLowerCase()
  if (x === y) return 100
  if (y.startsWith(x) || x.startsWith(y)) return 80 - Math.abs(x.length - y.length)
  if (y.includes(x) || x.includes(y)) return 60 - Math.abs(x.length - y.length)

  // Character overlap (Dice coefficient on bigrams)
  if (x.length < 2 || y.length < 2) return 0
  const bigramsX = new Set<string>()
  for (let i = 0; i < x.length - 1; i++) bigramsX.add(x.slice(i, i + 2))
  let overlap = 0
  for (let i = 0; i < y.length - 1; i++) {
    if (bigramsX.has(y.slice(i, i + 2))) overlap++
  }
  return Math.round((2 * overlap * 50) / (x.length + y.length - 2))
}

export function resolveImportProvider(rawProvider: string): ProviderMappingEntry {
  const raw = rawProvider.trim()
  const lower = raw.toLowerCase()

  const knownAliases = new Set(Object.values(PROVIDER_ID_TO_ALIAS))

  if (PROVIDER_ID_TO_ALIAS[lower]) {
    return {
      rawProvider: raw,
      resolvedAlias: PROVIDER_ID_TO_ALIAS[lower],
      status: 'exact',
      candidates: [PROVIDER_ID_TO_ALIAS[lower]],
    }
  }

  if (knownAliases.has(lower)) {
    return {
      rawProvider: raw,
      resolvedAlias: lower,
      status: 'exact',
      candidates: [lower],
    }
  }

  if (OMNIROUTE_TO_9ROUTER_ALIASES[lower]) {
    const mapped = OMNIROUTE_TO_9ROUTER_ALIASES[lower]
    return {
      rawProvider: raw,
      resolvedAlias: mapped,
      status: 'exact',
      candidates: [mapped],
    }
  }

  // Fuzzy match against all known provider IDs and aliases
  const scored = new Map<string, number>()
  for (const [id, alias] of Object.entries(PROVIDER_ID_TO_ALIAS)) {
    const score = Math.max(similarityScore(lower, id), similarityScore(lower, alias))
    if (score >= 25) {
      const prev = scored.get(alias) ?? 0
      if (score > prev) scored.set(alias, score)
    }
  }

  const ranked = Array.from(scored.entries())
    .sort((a, b) => b[1] - a[1])
    .map(([alias]) => alias)
    .slice(0, 5)

  if (ranked.length > 0) {
    return {
      rawProvider: raw,
      resolvedAlias: ranked[0],
      status: 'suggested',
      candidates: ranked,
    }
  }

  return {
    rawProvider: raw,
    resolvedAlias: '',
    status: 'unresolved',
    candidates: [],
  }
}

function mapImportStrategy(raw: unknown): string {
  if (typeof raw !== 'string' || !raw.trim()) return 'fallback'
  const s = raw.trim().toLowerCase()
  if (s === 'priority') return 'fallback'
  return s
}

export function parseCombosImport(rawJson: string): ParsedCombosImport {
  const parsed = JSON.parse(rawJson) as unknown
  const root =
    Array.isArray(parsed)
      ? { combos: parsed }
      : parsed && typeof parsed === 'object'
        ? (parsed as Record<string, unknown>)
        : null

  if (!root || !Array.isArray(root.combos)) {
    throw new Error('Invalid JSON: expected an array or an object with a "combos" array.')
  }

  const meta = root._meta && typeof root._meta === 'object' ? (root._meta as Record<string, unknown>) : null
  const isOmniRoute =
    typeof meta?.version === 'string' && meta.version.toLowerCase().includes('omniroute')

  const rawProviders = new Set<string>()
  const combos: ParsedImportCombo[] = []

  for (const item of root.combos) {
    if (!item || typeof item !== 'object') continue
    const obj = item as Record<string, unknown>
    const name = typeof obj.name === 'string' ? obj.name.trim() : ''
    if (!name) continue

    const steps: ParsedImportStep[] = []
    const rawModels = Array.isArray(obj.models) ? obj.models : []

    for (const m of rawModels) {
      if (typeof m === 'string') {
        const s = m.trim()
        if (!s) continue
        const slash = s.indexOf('/')
        if (slash === -1) {
          steps.push({ kind: 'combo-ref', modelId: s })
        } else {
          const prov = s.slice(0, slash)
          rawProviders.add(prov)
          steps.push({ kind: 'model', rawProvider: prov, modelId: s.slice(slash + 1) })
        }
        continue
      }

      if (m && typeof m === 'object') {
        const mObj = m as Record<string, unknown>
        if (mObj.kind === 'combo-ref' && typeof mObj.comboName === 'string' && mObj.comboName.trim()) {
          steps.push({ kind: 'combo-ref', modelId: mObj.comboName.trim() })
          continue
        }
        const rawModel = typeof mObj.model === 'string' ? mObj.model.trim() : ''
        if (!rawModel) continue
        const explicitProv = typeof mObj.providerId === 'string' ? mObj.providerId.trim() : ''
        if (explicitProv) {
          rawProviders.add(explicitProv)
          const prefix = `${explicitProv}/`
          const modelId = rawModel.startsWith(prefix) ? rawModel.slice(prefix.length) : rawModel
          steps.push({ kind: 'model', rawProvider: explicitProv, modelId })
        } else {
          const slash = rawModel.indexOf('/')
          if (slash === -1) {
            steps.push({ kind: 'combo-ref', modelId: rawModel })
          } else {
            const prov = rawModel.slice(0, slash)
            rawProviders.add(prov)
            steps.push({ kind: 'model', rawProvider: prov, modelId: rawModel.slice(slash + 1) })
          }
        }
      }
    }

    const customPrompt =
      typeof obj.customPrompt === 'string' && obj.customPrompt.trim()
        ? obj.customPrompt
        : typeof obj.system_message === 'string' && obj.system_message.trim()
          ? obj.system_message
          : undefined

    combos.push({
      name,
      steps,
      strategy: mapImportStrategy(obj.strategy),
      customPrompt,
      matchRules: importMatchRules(obj.matchRules),
    })
  }

  const providerMappings = Array.from(rawProviders)
    .sort()
    .map((p) => resolveImportProvider(p))

  return {
    format: isOmniRoute ? 'omniroute' : '9router',
    combos,
    providerMappings,
  }
}

export function buildCombosFromImport(
  parsed: ParsedCombosImport,
  providerOverrides: Record<string, string>,
): Array<{
  name: string
  models: string[]
  strategy: string
  matchRules?: string | SmartRule[]
  customPrompt?: string
}> {
  const mapping = new Map<string, string>()
  for (const entry of parsed.providerMappings) {
    const chosen = providerOverrides[entry.rawProvider] ?? entry.resolvedAlias
    if (chosen) mapping.set(entry.rawProvider, chosen)
  }

  return parsed.combos.map((c) => {
    const models = c.steps.map((step) => {
      if (step.kind === 'combo-ref' || !step.rawProvider) {
        return step.modelId
      }
      const targetProvider = mapping.get(step.rawProvider) || step.rawProvider
      return `${targetProvider}/${step.modelId}`
    })
    return {
      name: c.name,
      models,
      strategy: c.strategy,
      ...(c.matchRules ? { matchRules: c.matchRules } : {}),
      ...(c.customPrompt ? { customPrompt: c.customPrompt } : {}),
    }
  })
}

export function exportCombosToJson(combos: Combo[]): string {
  const payload = {
    _meta: {
      version: '9router-combos-v1',
      exportedAt: new Date().toISOString(),
    },
    combos: combos.map((c) => ({
      name: c.name,
      models: getComboModels(c),
      strategy: c.strategy || 'fallback',
      ...(c.matchRules?.length ? { matchRules: c.matchRules } : {}),
      ...(c.customPrompt ? { customPrompt: c.customPrompt } : {}),
    })),
  }
  return JSON.stringify(payload, null, 2)
}
