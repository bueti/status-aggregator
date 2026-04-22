<script lang="ts">
	import { page } from '$app/state';
	import StatusBadge from '$lib/components/StatusBadge.svelte';
	import IncidentList from '$lib/components/IncidentList.svelte';
	import RefreshIndicator from '$lib/components/RefreshIndicator.svelte';
	import { api, APIError } from '$lib/api/client';
	import type { ProviderDetail } from '$lib/api/types';

	const id = $derived(page.params.id ?? '');

	let data = $state<ProviderDetail | null>(null);
	let error = $state<Error | null>(null);
	let loading = $state(false);
	let lastUpdated = $state<Date | null>(null);

	async function load() {
		if (!id) return;
		loading = true;
		try {
			data = await api.provider(id);
			error = null;
			lastUpdated = new Date();
		} catch (e) {
			error = e as Error;
			if (e instanceof APIError && e.status === 404) data = null;
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		void id;
		data = null;
		load();
		const t = setInterval(load, 30_000);
		return () => clearInterval(t);
	});
</script>

<div class="mb-4 flex items-center gap-3 text-sm">
	<a href="/" class="text-fg-muted hover:text-fg">← Overview</a>
</div>

{#if error}
	<div
		class="rounded-lg border border-critical/30 bg-critical/10 p-4 text-sm text-critical"
	>
		{error.message}
	</div>
{:else if data}
	<div class="flex flex-wrap items-start justify-between gap-4">
		<div>
			<div class="flex items-center gap-3">
				<h1 class="text-2xl font-semibold">{data.name}</h1>
				<StatusBadge indicator={data.indicator} />
			</div>
			<p class="mt-1 text-fg-muted">{data.description || '—'}</p>
			{#if data.url}
				<a
					href={data.url}
					target="_blank"
					rel="noreferrer"
					class="mt-1 inline-block text-xs text-fg-subtle hover:text-fg-muted"
				>
					{data.url} ↗
				</a>
			{/if}
			{#if data.err}
				<p class="mt-2 text-xs text-critical">Last fetch error: {data.err}</p>
			{/if}
		</div>
		<RefreshIndicator {loading} {lastUpdated} onrefresh={load} />
	</div>

	<section class="mt-8">
		<h2 class="mb-3 text-sm font-semibold tracking-wide text-fg-muted uppercase">
			Active incidents
		</h2>
		<IncidentList incidents={data.incidents ?? []} />
	</section>

	<section class="mt-8">
		<h2 class="mb-3 text-sm font-semibold tracking-wide text-fg-muted uppercase">
			Components
		</h2>
		{#if (data.components ?? []).length === 0}
			<p class="text-sm text-fg-muted">No components reported.</p>
		{:else}
			<ul class="grid gap-2 sm:grid-cols-2">
				{#each data.components ?? [] as c, i (i)}
					<li
						class="flex items-center justify-between rounded-md border border-border bg-surface px-3 py-2"
					>
						<span class="truncate">{c.name}</span>
						<StatusBadge indicator={c.status} size="sm" />
					</li>
				{/each}
			</ul>
		{/if}
	</section>
{:else if loading}
	<div class="space-y-3">
		<div class="h-8 w-64 animate-pulse rounded bg-surface"></div>
		<div class="h-4 w-96 animate-pulse rounded bg-surface"></div>
	</div>
{/if}
