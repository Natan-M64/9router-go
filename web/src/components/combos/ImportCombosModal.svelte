<script lang="ts">
  import { api, type Combo } from '../../api/client'
  import Button from '../../lib/ui/Button.svelte'
  import Modal from '../../lib/ui/Modal.svelte'
  import {
    buildCombosFromImport,
    getKnownProviderOptions,
    parseCombosImport,
    type ParsedCombosImport,
  } from './comboImportExport'

  interface Props {
    isOpen: boolean
    existingCombos: Combo[]
    onClose: () => void
    onImported: () => Promise<void> | void
  }

  let { isOpen, existingCombos, onClose, onImported }: Props = $props()

  let rawText = $state('')
  let parsed = $state<ParsedCombosImport | null>(null)
  let parseError = $state('')
  let providerOverrides = $state<Record<string, string>>({})
  let skipExisting = $state(true)
  let isImporting = $state(false)
  let importResult = $state('')

  const providerOptions = getKnownProviderOptions()
  let existingNames = $derived(new Set(existingCombos.map((c) => c.name)))
  let hasUnresolvedProviders = $derived(
    parsed ? parsed.providerMappings.some((m) => !(providerOverrides[m.rawProvider] || m.resolvedAlias)) : false,
  )

  function handleParse(text: string) {
    rawText = text
    parseError = ''
    importResult = ''
    if (!text.trim()) {
      parsed = null
      providerOverrides = {}
      return
    }
    try {
      const next = parseCombosImport(text)
      parsed = next
      const overrides: Record<string, string> = {}
      for (const m of next.providerMappings) {
        overrides[m.rawProvider] = m.resolvedAlias
      }
      providerOverrides = overrides
    } catch (err) {
      parsed = null
      parseError = err instanceof Error ? err.message : 'Invalid JSON'
    }
  }

  async function handleFileChange(e: Event) {
    const input = e.currentTarget as HTMLInputElement
    const file = input.files?.[0]
    if (!file) return
    const text = await file.text()
    handleParse(text)
    input.value = ''
  }

  async function handleImport() {
    if (!parsed || hasUnresolvedProviders || isImporting) return
    isImporting = true
    parseError = ''
    importResult = ''

    let created = 0
    let skipped = 0
    let failed = 0

    try {
      const payloads = buildCombosFromImport(parsed, providerOverrides)
      for (const payload of payloads) {
        if (skipExisting && existingNames.has(payload.name)) {
          skipped++
          continue
        }
        try {
          await api.createCombo(payload)
          created++
        } catch {
          failed++
        }
      }
      importResult = `Imported ${created}, skipped ${skipped}${failed > 0 ? `, failed ${failed}` : ''}.`
      await onImported()
      if (failed === 0 && created > 0) {
        onClose()
      }
    } finally {
      isImporting = false
    }
  }

  function resetAndClose() {
    rawText = ''
    parsed = null
    parseError = ''
    importResult = ''
    providerOverrides = {}
    onClose()
  }
</script>

<Modal isOpen={isOpen} onClose={resetAndClose} title="Import Combos (JSON / OmniRoute)" size="lg">
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <p class="text-xs text-text-muted">
        Upload or paste a <code class="font-mono">9router-combos-v1</code> or
        <code class="font-mono">omniroute-combos-v1</code> JSON file.
      </p>
      <label class="inline-flex cursor-pointer items-center gap-1.5 rounded-lg border border-border bg-surface-2 px-3 py-1.5 text-xs font-medium text-text-main hover:bg-surface-3">
        <span class="material-symbols-outlined text-[16px]">upload_file</span>
        <span>Choose JSON file</span>
        <input type="file" accept=".json,application/json" class="hidden" onchange={handleFileChange} />
      </label>
    </div>

    <textarea
      value={rawText}
      oninput={(e) => handleParse(e.currentTarget.value)}
      rows="7"
      placeholder='Paste JSON here...'
      class="w-full rounded-lg border border-border bg-surface-2 p-3 font-mono text-xs text-text-main focus:border-brand-500 focus:outline-none"
    ></textarea>

    {#if parseError}
      <p class="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-xs text-red-500">
        {parseError}
      </p>
    {/if}

    {#if parsed}
      <div class="flex flex-col gap-3 rounded-lg border border-border bg-surface-2/60 p-3">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <div class="flex items-center gap-2 text-xs">
            <span class="rounded bg-brand-500/15 px-2 py-0.5 font-semibold text-brand-600 dark:text-brand-400">
              {parsed.format === 'omniroute' ? 'OmniRoute' : '9Router-Go'}
            </span>
            <span class="text-text-main font-medium">{parsed.combos.length} combos detected</span>
          </div>
          <label class="flex cursor-pointer items-center gap-2 text-xs text-text-muted">
            <input type="checkbox" bind:checked={skipExisting} class="accent-brand-500" />
            <span>Skip existing combos</span>
          </label>
        </div>

        {#if parsed.providerMappings.length > 0}
          <div class="flex flex-col gap-2 border-t border-border pt-2">
            <p class="text-[11px] font-medium text-text-muted">
              Provider mapping (review or override any provider alias):
            </p>
            <div class="grid gap-2 sm:grid-cols-2">
              {#each parsed.providerMappings as entry (entry.rawProvider)}
                <div class="flex items-center justify-between gap-2 rounded border px-2.5 py-1.5 text-xs {entry.status === 'exact' ? 'border-border bg-background' : 'border-amber-500/40 bg-amber-500/10'}">
                  <span class="font-mono text-text-main truncate" title={entry.rawProvider}>
                    {entry.rawProvider}
                  </span>
                  <select
                    aria-label="Map provider {entry.rawProvider}"
                    value={providerOverrides[entry.rawProvider] ?? ''}
                    onchange={(e) => {
                      providerOverrides = { ...providerOverrides, [entry.rawProvider]: e.currentTarget.value }
                    }}
                    class="max-w-[170px] rounded border border-border bg-background px-2 py-1 text-xs text-text-main focus:border-brand-500 focus:outline-none"
                  >
                    <option value="">Select provider…</option>
                    {#each providerOptions as opt (opt.alias)}
                      <option value={opt.alias}>{opt.label}</option>
                    {/each}
                  </select>
                </div>
              {/each}
            </div>
          </div>
        {/if}
      </div>
    {/if}

    {#if importResult}
      <p class="rounded-lg border border-emerald-500/30 bg-emerald-500/10 px-3 py-2 text-xs text-emerald-600 dark:text-emerald-400">
        {importResult}
      </p>
    {/if}

    <div class="flex justify-end gap-2 border-t border-border pt-3">
      <Button variant="secondary" size="sm" onclick={resetAndClose}>Cancel</Button>
      <Button
        size="sm"
        disabled={!parsed || parsed.combos.length === 0 || hasUnresolvedProviders || isImporting}
        onclick={handleImport}
      >
        {isImporting ? 'Importing…' : `Import ${parsed?.combos.length ?? 0} Combos`}
      </Button>
    </div>
  </div>
</Modal>
