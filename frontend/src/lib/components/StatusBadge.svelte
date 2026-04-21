<script lang="ts">
	import { asIndicator, INDICATOR_CLASS, INDICATOR_LABEL } from '$lib/api/types';

	type Props = {
		indicator: string;
		label?: string;
		size?: 'sm' | 'md';
	};

	let { indicator, label, size = 'md' }: Props = $props();

	const normalized = $derived(asIndicator(indicator));
	const cls = $derived(INDICATOR_CLASS[normalized]);
	const text = $derived(label ?? INDICATOR_LABEL[normalized]);
</script>

<span
	class="inline-flex items-center gap-1.5 rounded-full ring-1 ring-inset font-medium {cls}"
	class:px-2={size === 'sm'}
	class:py-0.5={size === 'sm'}
	class:text-xs={size === 'sm'}
	class:px-2.5={size === 'md'}
	class:py-1={size === 'md'}
	class:text-sm={size === 'md'}
>
	<span class="inline-block h-1.5 w-1.5 rounded-full bg-current"></span>
	{text}
</span>
