<script lang="ts">
  import { fmtTokens } from '../../lib/compactNumber'
  import { costIn } from '../../lib/currency.svelte'
  import { type StatsData } from './types'

  let { stats = {} }: { stats?: StatsData } = $props()

  function fmtSuccessRate(rate?: number | null): string {
    if (rate == null) return '—'
    return `${rate.toFixed(1)}%`
  }

  function rateColor(rate?: number | null): string {
    if (rate == null) return 'text-text-muted'
    if (rate >= 95) return 'text-success'
    if (rate >= 80) return 'text-warning'
    return 'text-red-500'
  }

  function fmtSuccessSubtext(s: StatsData): string {
    const total = s.totalRequests ?? 0
    if (total === 0) return 'No requests recorded'
    const errors = s.totalErrors ?? 0
    const successful = total - errors
    return `${successful.toLocaleString()}/${total.toLocaleString()} success`
  }
  // The server splits its list-price total into what a per-token bill would
  // charge and what ran on a subscription, plan quota or free tier. Until a
  // gateway that reports the split answers, fall back to the plain caveat.
  let costSplit = $derived.by(() => {
    const metered = stats.totalCostMetered
    const included = stats.totalCostIncluded
    if (metered == null && included == null) return ''
    const unclassified = stats.totalCostUnclassified ?? 0
    const parts = [`${costIn(metered ?? 0)} billed`, `${costIn(included ?? 0)} included`]
    if (unclassified > 0) parts.push(`${costIn(unclassified)} unclassified`)
    return parts.join(' · ')
  })
</script>

<div class="grid min-w-0 grid-cols-2 gap-3 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-6 sm:gap-4">
  <div class="bg-surface border border-border-subtle rounded-[14px] shadow-[var(--shadow-soft)] flex min-w-0 flex-col gap-1 px-4 py-3">
    <span class="text-text-muted text-sm uppercase font-semibold">Total Requests</span>
    <span class="truncate text-2xl font-bold text-text-main" title={fmtTokens(stats.totalRequests)}>{fmtTokens(stats.totalRequests)}</span>
  </div>
  <div class="bg-surface border border-border-subtle rounded-[14px] shadow-[var(--shadow-soft)] flex min-w-0 flex-col gap-1 px-4 py-3">
    <span class="text-text-muted text-sm uppercase font-semibold">Success Rate</span>
    <span class="truncate text-2xl font-bold {rateColor(stats.successRate)}" title={fmtSuccessRate(stats.successRate)}>
      {fmtSuccessRate(stats.successRate)}
    </span>
    <span class="text-[10px] text-text-muted truncate">{fmtSuccessSubtext(stats)}</span>
  </div>
  <div class="bg-surface border border-border-subtle rounded-[14px] shadow-[var(--shadow-soft)] flex min-w-0 flex-col gap-1 px-4 py-3">
    <span class="text-text-muted text-sm uppercase font-semibold">Total Input Tokens</span>
    <span class="truncate text-2xl font-bold text-primary" title={fmtTokens(stats.totalPromptTokens)}>{fmtTokens(stats.totalPromptTokens)}</span>
  </div>
  <div class="bg-surface border border-border-subtle rounded-[14px] shadow-[var(--shadow-soft)] flex min-w-0 flex-col gap-1 px-4 py-3">
    <span class="text-text-muted text-sm uppercase font-semibold">Cached Tokens</span>
    <span class="truncate text-2xl font-bold text-info" title={fmtTokens(stats.totalCachedTokens)}>{fmtTokens(stats.totalCachedTokens)}</span>
  </div>
  <div class="bg-surface border border-border-subtle rounded-[14px] shadow-[var(--shadow-soft)] flex min-w-0 flex-col gap-1 px-4 py-3">
    <span class="text-text-muted text-sm uppercase font-semibold">Output Tokens</span>
    <span class="truncate text-2xl font-bold text-success" title={fmtTokens(stats.totalCompletionTokens)}>{fmtTokens(stats.totalCompletionTokens)}</span>
  </div>
  <div class="bg-surface border border-border-subtle rounded-[14px] shadow-[var(--shadow-soft)] flex min-w-0 flex-col gap-1 px-4 py-3">
    <span class="text-text-muted text-sm uppercase font-semibold">Est. Cost</span>
    <span class="truncate text-2xl font-bold text-warning">~{costIn(stats.totalCost ?? 0)}</span>
    {#if costSplit}
      <span class="text-[10px] text-text-muted truncate" title="Billed is estimated per-token spend; included ran on a subscription, plan quota or free tier at list price.">{costSplit}</span>
    {:else}
      <span class="text-[10px] text-text-muted truncate">Estimated, not actual</span>
    {/if}
  </div>
</div>
