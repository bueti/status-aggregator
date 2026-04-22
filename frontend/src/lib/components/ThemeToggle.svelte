<script lang="ts">
	import type { ThemePref } from '$lib/theme.svelte';

	type Props = {
		pref: ThemePref;
		onchange: (next: ThemePref) => void;
	};

	let { pref, onchange }: Props = $props();

	const options: { value: ThemePref; label: string; icon: string }[] = [
		{ value: 'light', label: 'Light', icon: '☀' },
		{ value: 'dark', label: 'Dark', icon: '☾' },
		{ value: 'system', label: 'System', icon: '◐' }
	];
</script>

<div
	class="inline-flex items-center gap-0.5 rounded-md border border-border bg-surface p-0.5"
	role="group"
	aria-label="Theme"
>
	{#each options as opt (opt.value)}
		<button
			type="button"
			onclick={() => onchange(opt.value)}
			aria-label="{opt.label} theme"
			aria-pressed={pref === opt.value}
			title={opt.label}
			class={[
				'rounded px-2 py-1 text-xs transition',
				pref === opt.value
					? 'bg-surface-hover text-fg'
					: 'text-fg-muted hover:text-fg'
			]}
		>
			<span aria-hidden="true">{opt.icon}</span>
		</button>
	{/each}
</div>
