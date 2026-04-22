<script lang="ts">
	type Props = {
		loading: boolean;
		lastUpdated: Date | null;
		onrefresh: () => void;
	};

	let { loading, lastUpdated, onrefresh }: Props = $props();

	let tick = $state(0);
	$effect(() => {
		const id = setInterval(() => tick++, 1000);
		return () => clearInterval(id);
	});

	const relative = $derived.by(() => {
		void tick;
		if (!lastUpdated) return 'never';
		const s = Math.round((Date.now() - lastUpdated.getTime()) / 1000);
		if (s < 5) return 'just now';
		if (s < 60) return `${s}s ago`;
		const m = Math.round(s / 60);
		if (m < 60) return `${m}m ago`;
		const h = Math.round(m / 60);
		return `${h}h ago`;
	});
</script>

<button
	type="button"
	onclick={onrefresh}
	disabled={loading}
	class="flex items-center gap-2 rounded-md border border-border bg-surface px-3 py-1.5 text-xs text-fg-muted transition hover:bg-surface-hover disabled:opacity-60"
>
	<span
		class="inline-block h-1.5 w-1.5 rounded-full"
		class:bg-ok={!loading}
		class:bg-minor={loading}
		class:animate-pulse={loading}
	></span>
	<span>Updated {relative}</span>
</button>
