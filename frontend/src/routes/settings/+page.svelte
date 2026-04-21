<script lang="ts">
	import { api, APIError, getAdminToken, setAdminToken } from '$lib/api/client';
	import type {
		FeedKindInfo,
		ProviderDetail,
		ProviderBody,
		ValidateResult
	} from '$lib/api/types';
	import StatusBadge from '$lib/components/StatusBadge.svelte';

	let kinds = $state<FeedKindInfo[]>([]);
	let providers = $state<ProviderDetail[]>([]);
	let listError = $state<string>('');
	let tokenInput = $state(getAdminToken());
	let hasToken = $derived(tokenInput.trim().length > 0);
	let tokenSaved = $state(false);

	type FormState = {
		name: string;
		id: string;
		kind: string;
		params: Record<string, string>;
		busy: boolean;
		error: string;
		validated: ValidateResult | null;
	};

	let form = $state<FormState>({
		name: '',
		id: '',
		kind: '',
		params: {},
		busy: false,
		error: '',
		validated: null
	});

	const currentKind = $derived(kinds.find((k) => k.kind === form.kind));

	async function loadAll() {
		listError = '';
		try {
			const [k, p] = await Promise.all([api.feedKinds(), api.listProviders()]);
			kinds = k.kinds ?? [];
			providers = p.providers ?? [];
			if (!form.kind && kinds.length > 0) {
				form.kind = kinds[0].kind;
				resetParams();
			}
		} catch (e) {
			listError = (e as Error).message;
		}
	}

	function resetParams() {
		form.params = {};
		for (const f of currentKind?.fields ?? []) {
			form.params[f.name] = '';
		}
		form.validated = null;
		form.error = '';
	}

	function buildBody(): ProviderBody {
		const body: Record<string, string> = {};
		for (const f of currentKind?.fields ?? []) {
			if (form.params[f.name]) body[f.name] = form.params[f.name];
		}
		return {
			id: form.id || undefined,
			name: form.name,
			kind: form.kind as ProviderBody['kind'],
			params: body
		} as ProviderBody;
	}

	async function validate() {
		form.busy = true;
		form.error = '';
		form.validated = null;
		try {
			form.validated = await api.validateProvider(buildBody());
		} catch (e) {
			form.error = (e as Error).message;
		} finally {
			form.busy = false;
		}
	}

	async function create() {
		form.busy = true;
		form.error = '';
		try {
			await api.createProvider(buildBody());
			form.name = '';
			form.id = '';
			resetParams();
			await loadAll();
		} catch (e) {
			form.error = (e as Error).message;
		} finally {
			form.busy = false;
		}
	}

	async function del(id: string, name: string) {
		if (!confirm(`Delete ${name}?`)) return;
		try {
			await api.deleteProvider(id);
			await loadAll();
		} catch (e) {
			alert((e as Error).message);
		}
	}

	let tokenSavedTimer: ReturnType<typeof setTimeout> | null = null;
	function saveToken() {
		setAdminToken(tokenInput.trim());
		tokenSaved = true;
		if (tokenSavedTimer) clearTimeout(tokenSavedTimer);
		tokenSavedTimer = setTimeout(() => (tokenSaved = false), 2000);
	}

	$effect(() => {
		loadAll();
	});
</script>

<h1 class="text-xl font-semibold">Settings</h1>

<section
	class="mt-6 rounded-xl border border-white/10 bg-white/5 p-4 sm:flex sm:items-end sm:gap-3"
>
	<label class="block flex-1">
		<span class="text-xs text-white/60">Admin token</span>
		<input
			type="password"
			bind:value={tokenInput}
			placeholder="STATUS_ADMIN_TOKEN"
			class="mt-1 w-full rounded-md border border-white/10 bg-black/30 px-3 py-2 text-sm outline-none focus:border-white/30"
		/>
	</label>
	<button
		type="button"
		onclick={saveToken}
		class="mt-2 rounded-md bg-white/10 px-4 py-2 text-sm hover:bg-white/20 sm:mt-0"
	>
		Save token
	</button>
</section>

{#if tokenSaved}
	<p class="mt-2 text-xs text-ok">Token saved to this browser.</p>
{/if}

{#if !hasToken}
	<p class="mt-3 text-xs text-white/50">
		Mutations require the admin token that was passed to the backend via
		<code class="rounded bg-white/5 px-1">STATUS_ADMIN_TOKEN</code>.
	</p>
{/if}

<section class="mt-8">
	<h2 class="mb-3 text-sm font-semibold tracking-wide text-white/60 uppercase">
		Configured providers
	</h2>
	{#if listError}
		<div class="rounded-lg border border-critical/30 bg-critical/10 p-3 text-sm text-critical">
			{listError}
		</div>
	{/if}
	{#if providers.length === 0 && !listError}
		<p class="text-sm text-white/50">None yet.</p>
	{/if}
	<ul class="flex flex-col gap-2">
		{#each providers as p (p.id)}
			<li
				class="flex items-center justify-between gap-3 rounded-lg border border-white/10 bg-white/5 p-3"
			>
				<div class="min-w-0">
					<div class="flex items-center gap-2">
						<span class="font-medium">{p.name}</span>
						<StatusBadge indicator={p.indicator} size="sm" />
					</div>
					<div class="truncate text-xs text-white/50">
						{p.kind} · <code class="text-white/40">{p.id}</code>
						{#if p.url}· {p.url}{/if}
					</div>
				</div>
				<button
					type="button"
					onclick={() => del(p.id, p.name)}
					disabled={!hasToken}
					class="rounded-md border border-critical/30 px-3 py-1 text-xs text-critical hover:bg-critical/10 disabled:opacity-40"
				>
					Delete
				</button>
			</li>
		{/each}
	</ul>
</section>

<section class="mt-8">
	<h2 class="mb-3 text-sm font-semibold tracking-wide text-white/60 uppercase">
		Add a provider
	</h2>
	<form
		class="flex flex-col gap-3 rounded-xl border border-white/10 bg-white/5 p-4"
		onsubmit={(e) => {
			e.preventDefault();
			create();
		}}
	>
		<div class="grid gap-3 sm:grid-cols-2">
			<label class="block">
				<span class="text-xs text-white/60">Name</span>
				<input
					bind:value={form.name}
					required
					placeholder="GitHub"
					class="mt-1 w-full rounded-md border border-white/10 bg-black/30 px-3 py-2 text-sm outline-none focus:border-white/30"
				/>
			</label>
			<label class="block">
				<span class="text-xs text-white/60">ID (optional)</span>
				<input
					bind:value={form.id}
					placeholder="auto-derived from name"
					class="mt-1 w-full rounded-md border border-white/10 bg-black/30 px-3 py-2 text-sm outline-none focus:border-white/30"
				/>
			</label>
		</div>

		<label class="block">
			<span class="text-xs text-white/60">Feed kind</span>
			<select
				bind:value={form.kind}
				onchange={resetParams}
				class="mt-1 w-full rounded-md border border-white/10 bg-black/30 px-3 py-2 text-sm outline-none focus:border-white/30"
			>
				{#each kinds as k (k.kind)}
					<option value={k.kind}>{k.label}</option>
				{/each}
			</select>
		</label>

		{#if currentKind}
			{#each currentKind.fields ?? [] as f (f.name)}
				<label class="block">
					<span class="text-xs text-white/60">{f.label}</span>
					<input
						type={f.type}
						bind:value={form.params[f.name]}
						required={f.required}
						placeholder={f.placeholder ?? ''}
						class="mt-1 w-full rounded-md border border-white/10 bg-black/30 px-3 py-2 text-sm outline-none focus:border-white/30"
					/>
					{#if f.help}
						<span class="mt-1 block text-xs text-white/40">{f.help}</span>
					{/if}
				</label>
			{/each}
		{/if}

		{#if form.error}
			<div
				class="rounded-md border border-critical/30 bg-critical/10 p-2 text-xs text-critical"
			>
				{form.error}
			</div>
		{/if}
		{#if form.validated}
			<div class="rounded-md border border-ok/30 bg-ok/10 p-2 text-xs text-ok">
				Connected · {form.validated.description || form.validated.indicator}
			</div>
		{/if}

		{#if !hasToken}
			<div
				class="rounded-md border border-minor/30 bg-minor/10 p-2 text-xs text-minor"
			>
				Enter your admin token above to enable Test &amp; Save.
			</div>
		{/if}

		<div class="flex gap-2">
			<button
				type="button"
				onclick={validate}
				disabled={!hasToken || form.busy || !form.name}
				class="rounded-md border border-white/20 px-4 py-2 text-sm hover:bg-white/10 disabled:opacity-40"
			>
				Test connection
			</button>
			<button
				type="submit"
				disabled={!hasToken || form.busy || !form.name}
				class="rounded-md bg-white px-4 py-2 text-sm font-medium text-black hover:bg-white/90 disabled:opacity-40"
			>
				{form.busy ? 'Saving…' : 'Save provider'}
			</button>
		</div>
	</form>
</section>
