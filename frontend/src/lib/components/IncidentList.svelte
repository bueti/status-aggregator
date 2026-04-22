<script lang="ts">
	import StatusBadge from './StatusBadge.svelte';
	import type { Incident } from '$lib/api/types';

	let { incidents }: { incidents: Incident[] | null | undefined } = $props();

	const list = $derived(incidents ?? []);

	function fmt(ts: string) {
		const d = new Date(ts);
		if (isNaN(+d)) return ts;
		return d.toLocaleString();
	}
</script>

{#if list.length === 0}
	<p class="text-sm text-fg-muted">No active incidents.</p>
{:else}
	<ul class="flex flex-col gap-3">
		{#each list as inc (inc.id)}
			<li
				class="flex flex-col gap-2 rounded-lg border border-border bg-surface p-3 sm:flex-row sm:items-start sm:justify-between"
			>
				<div class="min-w-0">
					<div class="flex items-center gap-2">
						<StatusBadge indicator={inc.impact} label={inc.status} size="sm" />
						<a
							href={inc.url}
							target="_blank"
							rel="noreferrer"
							class="truncate font-medium hover:underline"
						>
							{inc.name}
						</a>
					</div>
					<div class="mt-1 text-xs text-fg-subtle">Updated {fmt(inc.updated_at)}</div>
				</div>
			</li>
		{/each}
	</ul>
{/if}
