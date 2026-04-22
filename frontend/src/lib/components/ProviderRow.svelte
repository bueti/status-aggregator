<script lang="ts">
	import StatusBadge from './StatusBadge.svelte';
	import type { ProviderSummary } from '$lib/api/types';

	let { provider }: { provider: ProviderSummary } = $props();
</script>

<a
	href="/providers/{provider.id}"
	class="group flex items-center justify-between gap-4 rounded-lg border border-border bg-surface px-4 py-3 transition hover:bg-surface-hover"
>
	<div class="flex min-w-0 flex-1 items-center gap-3">
		<StatusBadge indicator={provider.indicator} size="sm" />
		<div class="min-w-0 flex-1">
			<div class="flex items-baseline gap-2">
				<span class="truncate font-medium">{provider.name}</span>
				{#if provider.stale}
					<span class="rounded bg-unknown/20 px-1.5 py-0.5 text-xs text-unknown">stale</span>
				{/if}
			</div>
			<div class="truncate text-xs text-fg-muted">
				{provider.description || '—'}
			</div>
		</div>
	</div>
	<div class="flex items-center gap-3 text-xs text-fg-subtle">
		{#if provider.active_incidents > 0}
			<span class="rounded bg-surface-hover px-1.5 py-0.5">
				{provider.active_incidents} active
			</span>
		{/if}
		<span class="transition group-hover:text-fg-muted">→</span>
	</div>
</a>
