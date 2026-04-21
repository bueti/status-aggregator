<script lang="ts">
	import '../app.css';
	import { page } from '$app/state';

	let { children } = $props();

	const nav = [
		{ href: '/', label: 'Overview' },
		{ href: '/settings', label: 'Settings' }
	];

	const current = $derived(page.url.pathname);

	function isActive(href: string) {
		if (href === '/') return current === '/';
		return current === href || current.startsWith(href + '/');
	}
</script>

<svelte:head>
	<title>Status Aggregator</title>
</svelte:head>

<div class="min-h-screen">
	<header class="border-b border-white/10 bg-black/20 backdrop-blur">
		<div class="mx-auto flex max-w-5xl items-center gap-6 px-6 py-4">
			<a href="/" class="font-semibold tracking-tight">Status Aggregator</a>
			<nav class="flex gap-4 text-sm">
				{#each nav as item (item.href)}
					<a
						href={item.href}
						class={[
							'rounded px-2 py-1 transition hover:bg-white/5',
							isActive(item.href) ? 'text-white' : 'text-white/60'
						]}
					>
						{item.label}
					</a>
				{/each}
			</nav>
		</div>
	</header>
	<main class="mx-auto max-w-5xl px-6 py-8">
		{@render children()}
	</main>
</div>
