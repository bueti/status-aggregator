<script lang="ts">
	import StatusBadge from './StatusBadge.svelte';
	import type { ProviderSummary } from '$lib/api/types';

	let { provider }: { provider: ProviderSummary } = $props();
</script>

<a
	href="/providers/{provider.id}"
	class="group flex flex-col gap-3 rounded-xl border border-white/10 bg-white/5 p-4 transition hover:bg-white/10"
>
	<div class="flex items-start justify-between gap-2">
		<div class="min-w-0">
			<div class="truncate font-medium">{provider.name}</div>
			{#if provider.url}
				<div class="truncate text-xs text-white/40">{provider.url}</div>
			{/if}
		</div>
		<StatusBadge indicator={provider.indicator} size="sm" />
	</div>
	<div class="text-sm text-white/70">
		{provider.description || '—'}
	</div>
	<div class="mt-auto flex items-center justify-between text-xs text-white/40">
		<div class="flex items-center gap-2">
			{#if provider.active_incidents > 0}
				<span class="rounded bg-white/10 px-1.5 py-0.5">
					{provider.active_incidents} active
				</span>
			{/if}
			{#if provider.stale}
				<span class="rounded bg-unknown/20 px-1.5 py-0.5 text-unknown">stale</span>
			{/if}
		</div>
		<span class="text-white/30 transition group-hover:text-white/60">→</span>
	</div>
</a>
