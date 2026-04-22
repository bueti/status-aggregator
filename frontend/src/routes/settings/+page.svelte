<script lang="ts">
	import { api, getAdminToken, setAdminToken } from '$lib/api/client';
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
		editingId: string | null;
		name: string;
		id: string;
		kind: string;
		params: Record<string, string>;
		busy: boolean;
		error: string;
		validated: ValidateResult | null;
	};

	function emptyForm(): FormState {
		return {
			editingId: null,
			name: '',
			id: '',
			kind: '',
			params: {},
			busy: false,
			error: '',
			validated: null
		};
	}

	let form = $state<FormState>(emptyForm());
	let formSection = $state<HTMLElement | null>(null);

	const currentKind = $derived(kinds.find((k) => k.kind === form.kind));
	const isEditing = $derived(form.editingId !== null);

	// Save requires the current form to match the last validated fingerprint.
	let lastValidatedFingerprint = $state('');
	const currentFingerprint = $derived(
		JSON.stringify({
			name: form.name.trim(),
			id: form.id.trim(),
			kind: form.kind,
			params: form.params
		})
	);
	const isValidated = $derived(
		form.validated !== null && lastValidatedFingerprint === currentFingerprint
	);

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
		lastValidatedFingerprint = '';
		try {
			const fp = currentFingerprint;
			form.validated = await api.validateProvider(buildBody());
			lastValidatedFingerprint = fp;
		} catch (e) {
			form.error = (e as Error).message;
		} finally {
			form.busy = false;
		}
	}

	async function submit() {
		form.busy = true;
		form.error = '';
		try {
			if (form.editingId) {
				await api.updateProvider(form.editingId, buildBody());
			} else {
				await api.createProvider(buildBody());
			}
			cancelEdit();
			await loadAll();
		} catch (e) {
			form.error = (e as Error).message;
		} finally {
			form.busy = false;
		}
	}

	function startEdit(p: ProviderDetail) {
		const params: Record<string, string> = {};
		const kind = kinds.find((k) => k.kind === p.kind);
		const raw = (p.params as unknown as Record<string, unknown>) ?? {};
		for (const f of kind?.fields ?? []) {
			const v = raw[f.name];
			params[f.name] = v == null ? '' : String(v);
		}
		form = {
			editingId: p.id,
			name: p.name,
			id: p.id,
			kind: p.kind,
			params,
			busy: false,
			error: '',
			validated: null
		};
		lastValidatedFingerprint = '';
		formSection?.scrollIntoView({ behavior: 'smooth', block: 'start' });
	}

	function cancelEdit() {
		const keepKind = form.kind || kinds[0]?.kind || '';
		form = emptyForm();
		form.kind = keepKind;
		resetParams();
		lastValidatedFingerprint = '';
	}

	async function del(id: string, name: string) {
		if (!confirm(`Delete ${name}?`)) return;
		try {
			await api.deleteProvider(id);
			if (form.editingId === id) cancelEdit();
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
	class="mt-6 rounded-xl border border-border bg-surface p-4 sm:flex sm:items-end sm:gap-3"
>
	<label class="block flex-1">
		<span class="text-xs text-fg-muted">Admin token</span>
		<input
			type="password"
			bind:value={tokenInput}
			placeholder="STATUS_ADMIN_TOKEN"
			class="mt-1 w-full rounded-md border border-border bg-surface-sunken px-3 py-2 text-sm outline-none focus:border-border-strong"
		/>
	</label>
	<button
		type="button"
		onclick={saveToken}
		class="mt-2 rounded-md bg-surface-hover px-4 py-2 text-sm hover:bg-surface-strong sm:mt-0"
	>
		Save token
	</button>
</section>

{#if tokenSaved}
	<p class="mt-2 text-xs text-ok">Token saved to this browser.</p>
{/if}

{#if !hasToken}
	<p class="mt-3 text-xs text-fg-subtle">
		Mutations require the admin token that was passed to the backend via
		<code class="rounded bg-surface px-1">STATUS_ADMIN_TOKEN</code>.
	</p>
{/if}

<section class="mt-8" bind:this={formSection}>
	<h2 class="mb-3 text-sm font-semibold tracking-wide text-fg-muted uppercase">
		{isEditing ? `Edit ${form.name || form.id}` : 'Add a provider'}
	</h2>
	<form
		class="flex flex-col gap-3 rounded-xl border border-border bg-surface p-4"
		onsubmit={(e) => {
			e.preventDefault();
			submit();
		}}
	>
		<div class="grid gap-3 sm:grid-cols-2">
			<label class="block">
				<span class="text-xs text-fg-muted">Name</span>
				<input
					bind:value={form.name}
					required
					placeholder="GitHub"
					class="mt-1 w-full rounded-md border border-border bg-surface-sunken px-3 py-2 text-sm outline-none focus:border-border-strong"
				/>
			</label>
			<label class="block">
				<span class="text-xs text-fg-muted">
					ID {isEditing ? '(locked)' : '(optional)'}
				</span>
				<input
					bind:value={form.id}
					disabled={isEditing}
					placeholder="auto-derived from name"
					class="mt-1 w-full rounded-md border border-border bg-surface-sunken px-3 py-2 text-sm outline-none focus:border-border-strong disabled:opacity-60"
				/>
			</label>
		</div>

		<label class="block">
			<span class="text-xs text-fg-muted">Feed kind</span>
			<select
				bind:value={form.kind}
				onchange={resetParams}
				class="mt-1 w-full rounded-md border border-border bg-surface-sunken px-3 py-2 text-sm outline-none focus:border-border-strong"
			>
				{#each kinds as k (k.kind)}
					<option value={k.kind}>{k.label}</option>
				{/each}
			</select>
		</label>

		{#if currentKind}
			{#each currentKind.fields ?? [] as f (f.name)}
				<label class="block">
					<span class="text-xs text-fg-muted">{f.label}</span>
					<input
						type={f.type}
						bind:value={form.params[f.name]}
						required={f.required}
						placeholder={f.placeholder ?? ''}
						class="mt-1 w-full rounded-md border border-border bg-surface-sunken px-3 py-2 text-sm outline-none focus:border-border-strong"
					/>
					{#if f.help}
						<span class="mt-1 block text-xs text-fg-subtle">{f.help}</span>
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
		{#if isValidated && form.validated}
			<div class="rounded-md border border-ok/30 bg-ok/10 p-2 text-xs text-ok">
				Connected · {form.validated.description || form.validated.indicator}
			</div>
		{:else if form.validated}
			<div class="rounded-md border border-minor/30 bg-minor/10 p-2 text-xs text-minor">
				Form changed since last test. Re-run Test connection before saving.
			</div>
		{/if}

		{#if !hasToken}
			<div
				class="rounded-md border border-minor/30 bg-minor/10 p-2 text-xs text-minor"
			>
				Enter your admin token above to enable Test &amp; Save.
			</div>
		{:else if !isValidated && form.name && !form.error}
			<p class="text-xs text-fg-muted">
				Test connection first to enable Save.
			</p>
		{/if}

		<div class="flex flex-wrap gap-2">
			<button
				type="button"
				onclick={validate}
				disabled={!hasToken || form.busy || !form.name}
				class="rounded-md border border-border-strong px-4 py-2 text-sm hover:bg-surface-hover disabled:opacity-40"
			>
				Test connection
			</button>
			<button
				type="submit"
				disabled={!hasToken || form.busy || !form.name || !isValidated}
				class="rounded-md bg-accent px-4 py-2 text-sm font-medium text-on-accent hover:bg-accent-hover disabled:opacity-40"
			>
				{#if form.busy}
					Saving…
				{:else if isEditing}
					Save changes
				{:else}
					Save provider
				{/if}
			</button>
			{#if isEditing}
				<button
					type="button"
					onclick={cancelEdit}
					class="rounded-md border border-border px-4 py-2 text-sm text-fg-muted hover:bg-surface-hover"
				>
					Cancel
				</button>
			{/if}
		</div>
	</form>
</section>

<section class="mt-8">
	<h2 class="mb-3 text-sm font-semibold tracking-wide text-fg-muted uppercase">
		Configured providers
	</h2>
	{#if listError}
		<div class="rounded-lg border border-critical/30 bg-critical/10 p-3 text-sm text-critical">
			{listError}
		</div>
	{/if}
	{#if providers.length === 0 && !listError}
		<p class="text-sm text-fg-muted">None yet.</p>
	{/if}
	<ul class="flex flex-col gap-2">
		{#each providers as p (p.id)}
			<li
				class={[
					'flex items-center justify-between gap-3 rounded-lg border bg-surface p-3',
					form.editingId === p.id ? 'border-border-strong' : 'border-border'
				]}
			>
				<div class="min-w-0">
					<div class="flex items-center gap-2">
						<span class="font-medium">{p.name}</span>
						<StatusBadge indicator={p.indicator} size="sm" />
					</div>
					<div class="truncate text-xs text-fg-subtle">
						{p.kind} · <code class="text-fg-subtle">{p.id}</code>
						{#if p.url}· {p.url}{/if}
					</div>
				</div>
				<div class="flex shrink-0 gap-2">
					<button
						type="button"
						onclick={() => startEdit(p)}
						disabled={!hasToken}
						class="rounded-md border border-border px-3 py-1 text-xs hover:bg-surface-hover disabled:opacity-40"
					>
						Edit
					</button>
					<button
						type="button"
						onclick={() => del(p.id, p.name)}
						disabled={!hasToken}
						class="rounded-md border border-critical/30 px-3 py-1 text-xs text-critical hover:bg-critical/10 disabled:opacity-40"
					>
						Delete
					</button>
				</div>
			</li>
		{/each}
	</ul>
</section>
