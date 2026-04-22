<script lang="ts">
	import '../app.css';
	import { page } from '$app/state';
	import ThemeToggle from '$lib/components/ThemeToggle.svelte';
	import { createTheme } from '$lib/theme.svelte';

	let { children } = $props();

	const nav = [
		{ href: '/', label: 'Overview' },
		{ href: '/settings', label: 'Settings' }
	];

	const current = $derived(page.url.pathname);
	const theme = createTheme();

	function isActive(href: string) {
		if (href === '/') return current === '/';
		return current === href || current.startsWith(href + '/');
	}
</script>

<svelte:head>
	<title>Status Aggregator</title>
</svelte:head>

<div class="flex min-h-screen flex-col">
	<header class="border-b border-border bg-surface backdrop-blur">
		<div class="mx-auto flex max-w-5xl items-center gap-6 px-6 py-4">
			<a href="/" class="font-semibold tracking-tight">Status Aggregator</a>
			<nav class="flex flex-1 gap-4 text-sm">
				{#each nav as item (item.href)}
					<a
						href={item.href}
						class={[
							'rounded px-2 py-1 transition hover:bg-surface-hover',
							isActive(item.href) ? 'text-fg' : 'text-fg-muted'
						]}
					>
						{item.label}
					</a>
				{/each}
			</nav>
			<ThemeToggle pref={theme.pref} onchange={theme.set} />
		</div>
	</header>
	<main class="mx-auto w-full max-w-5xl flex-1 px-6 py-8">
		{@render children()}
	</main>
	<footer class="mt-12 border-t border-border">
		<div class="mx-auto flex max-w-5xl items-center justify-between gap-4 px-6 py-4 text-xs text-fg-subtle">
			<span>Status Aggregator</span>
			<a
				href="https://github.com/bueti/status-aggregator"
				target="_blank"
				rel="noreferrer"
				class="hover:text-fg"
			>
				github.com/bueti/status-aggregator ↗
			</a>
		</div>
	</footer>
</div>
