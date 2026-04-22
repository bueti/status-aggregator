<script lang="ts">
	import ProviderCard from '$lib/components/ProviderCard.svelte';
	import ProviderRow from '$lib/components/ProviderRow.svelte';
	import StatusBadge from '$lib/components/StatusBadge.svelte';
	import RefreshIndicator from '$lib/components/RefreshIndicator.svelte';
	import { createOverview } from '$lib/api/overview.svelte';
	import { createOverviewPrefs, sortProviders, type SortMode, type ViewMode } from '$lib/prefs.svelte';

	const overview = createOverview();
	const prefs = createOverviewPrefs();

	const sorted = $derived(sortProviders(overview.data?.providers ?? [], prefs.sort));

	const sortOptions: { value: SortMode; label: string }[] = [
		{ value: 'name', label: 'Name' },
		{ value: 'added', label: 'Added' },
		{ value: 'status', label: 'Status' }
	];

	const viewOptions: { value: ViewMode; label: string; icon: string }[] = [
		{ value: 'grid', label: 'Grid', icon: '▦' },
		{ value: 'list', label: 'List', icon: '≡' }
	];
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

{#if overview.data && sorted.length > 0}
	<div class="mt-6 flex flex-wrap items-center justify-between gap-3">
		<label class="flex items-center gap-2 text-xs text-fg-muted">
			<span>Sort by</span>
			<select
				bind:value={prefs.sort}
				class="rounded-md border border-border bg-surface px-2 py-1 text-xs text-fg outline-none focus:border-border-strong"
			>
				{#each sortOptions as opt (opt.value)}
					<option value={opt.value}>{opt.label}</option>
				{/each}
			</select>
		</label>
		<div
			class="inline-flex items-center gap-0.5 rounded-md border border-border bg-surface p-0.5"
			role="group"
			aria-label="View"
		>
			{#each viewOptions as opt (opt.value)}
				<button
					type="button"
					onclick={() => (prefs.view = opt.value)}
					aria-label="{opt.label} view"
					aria-pressed={prefs.view === opt.value}
					title={opt.label}
					class={[
						'rounded px-2 py-1 text-xs transition',
						prefs.view === opt.value
							? 'bg-surface-hover text-fg'
							: 'text-fg-muted hover:text-fg'
					]}
				>
					<span aria-hidden="true">{opt.icon}</span>
					<span class="sr-only">{opt.label}</span>
				</button>
			{/each}
		</div>
	</div>
{/if}

{#if overview.data}
	{#if sorted.length > 0}
		{#if prefs.view === 'grid'}
			<div class="mt-4 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
				{#each sorted as p (p.id)}
					<ProviderCard provider={p} />
				{/each}
			</div>
		{:else}
			<div class="mt-4 flex flex-col gap-2">
				{#each sorted as p (p.id)}
					<ProviderRow provider={p} />
				{/each}
			</div>
		{/if}
	{:else}
		<div
			class="mt-6 rounded-lg border border-border bg-surface p-8 text-center text-fg-muted"
		>
			<p class="mb-3">No providers configured.</p>
			<a class="text-fg underline" href="/settings">Add one in settings →</a>
		</div>
	{/if}
{:else if !overview.error}
	<div class="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
		{#each Array(3) as _, i (i)}
			<div class="h-32 animate-pulse rounded-xl border border-border bg-surface"></div>
		{/each}
	</div>
{/if}
