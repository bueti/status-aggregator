<script lang="ts">
	import ProviderCard from '$lib/components/ProviderCard.svelte';
	import StatusBadge from '$lib/components/StatusBadge.svelte';
	import RefreshIndicator from '$lib/components/RefreshIndicator.svelte';
	import { createOverview } from '$lib/api/overview.svelte';

	const overview = createOverview();
</script>

<div class="flex flex-wrap items-center justify-between gap-4">
	<div class="flex items-center gap-3">
		{#if overview.data}
			<StatusBadge indicator={overview.data.worst_indicator} />
		{:else}
			<StatusBadge indicator="unknown" label="Loading..." />
		{/if}
		<h1 class="text-lg font-semibold">
			{#if overview.data && overview.data.worst_indicator === 'operational'}
				All systems operational
			{:else if overview.data}
				Some providers report issues
			{:else}
				Status Aggregator
			{/if}
		</h1>
	</div>
	<RefreshIndicator
		loading={overview.loading}
		lastUpdated={overview.lastUpdated}
		onrefresh={() => overview.refresh()}
	/>
</div>

{#if overview.error}
	<div
		class="mt-6 rounded-lg border border-critical/30 bg-critical/10 p-4 text-sm text-critical"
	>
		Failed to load overview: {overview.error.message}
	</div>
{/if}

{#if overview.data}
	<div class="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
		{#each overview.data.providers ?? [] as p (p.id)}
			<ProviderCard provider={p} />
		{/each}
	</div>
	{#if (overview.data.providers ?? []).length === 0}
		<div
			class="mt-6 rounded-lg border border-white/10 bg-white/5 p-8 text-center text-white/60"
		>
			<p class="mb-3">No providers configured.</p>
			<a class="text-white underline" href="/settings">Add one in settings →</a>
		</div>
	{/if}
{:else if !overview.error}
	<div class="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
		{#each Array(3) as _, i (i)}
			<div class="h-32 animate-pulse rounded-xl border border-white/10 bg-white/5"></div>
		{/each}
	</div>
{/if}
