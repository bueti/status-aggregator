<script lang="ts">
	import StatusBadge from './StatusBadge.svelte';
	import type { ProviderSummary } from '$lib/api/types';

	let { provider }: { provider: ProviderSummary } = $props();
</script>

<a
	href="/providers/{provider.id}"
	class="group flex flex-col gap-3 rounded-xl border border-border bg-surface p-4 transition hover:bg-surface-hover"
>
	<div class="flex items-start justify-between gap-2">
		<div class="min-w-0">
			<div class="truncate font-medium">{provider.name}</div>
			{#if provider.url}
				<div class="truncate text-xs text-fg-subtle">{provider.url}</div>
			{/if}
		</div>
		<StatusBadge indicator={provider.indicator} size="sm" />
	</div>
	<div class="text-sm text-fg-muted">
		{provider.description || '—'}
	</div>
	<div class="mt-auto flex items-center justify-between text-xs text-fg-subtle">
		<div class="flex items-center gap-2">
			{#if provider.active_incidents > 0}
				<span class="rounded bg-surface-hover px-1.5 py-0.5">
					{provider.active_incidents} active
				</span>
			{/if}
			{#if provider.stale}
				<span class="rounded bg-unknown/20 px-1.5 py-0.5 text-unknown">stale</span>
			{/if}
		</div>
		<span class="transition group-hover:text-fg-muted">→</span>
	</div>
</a>
