import { api, APIError } from './client';
import type { Overview } from './types';

export function createOverview(refreshMs = 30_000) {
	let data = $state<Overview | null>(null);
	let error = $state<APIError | Error | null>(null);
	let loading = $state(false);
	let lastUpdated = $state<Date | null>(null);

	async function refresh() {
		loading = true;
		try {
			data = await api.overview();
			error = null;
			lastUpdated = new Date();
		} catch (e) {
			error = e as Error;
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		refresh();
		const id = setInterval(refresh, refreshMs);
		return () => clearInterval(id);
	});

	return {
		get data() {
			return data;
		},
		get error() {
			return error;
		},
		get loading() {
			return loading;
		},
		get lastUpdated() {
			return lastUpdated;
		},
		refresh
	};
}
