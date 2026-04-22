import type {
	Overview,
	ProvidersList,
	FeedKindsList,
	ProviderDetail,
	ProviderBody,
	ValidateResult
} from './types';

const BASE = '/api';

const TOKEN_KEY = 'status-agg.admin-token';

export function getAdminToken(): string {
	if (typeof localStorage === 'undefined') return '';
	return localStorage.getItem(TOKEN_KEY) ?? '';
}

export function setAdminToken(t: string) {
	if (typeof localStorage === 'undefined') return;
	if (t) localStorage.setItem(TOKEN_KEY, t);
	else localStorage.removeItem(TOKEN_KEY);
}

export class APIError extends Error {
	status: number;
	detail?: string;
	constructor(status: number, message: string, detail?: string) {
		super(message);
		this.status = status;
		this.detail = detail;
	}
}

async function request<T>(
	path: string,
	init?: RequestInit & { admin?: boolean }
): Promise<T> {
	const headers = new Headers(init?.headers);
	if (!headers.has('Accept')) headers.set('Accept', 'application/json');
	if (init?.body && !headers.has('Content-Type')) {
		headers.set('Content-Type', 'application/json');
	}
	if (init?.admin) {
		const t = getAdminToken();
		if (!t) throw new APIError(401, 'Admin token is not set');
		headers.set('Authorization', `Bearer ${t}`);
	}
	const res = await fetch(BASE + path, { ...init, headers });
	if (!res.ok) {
		let title = res.statusText;
		let detail: string | undefined;
		try {
			const body = await res.json();
			if (typeof body?.title === 'string') title = body.title;
			if (typeof body?.detail === 'string') detail = body.detail;
		} catch {
			// ignore
		}
		// Prefer the detail (e.g. "feed_url is required") — the title is
		// usually the generic HTTP phrase. Fall back to the title if no detail.
		const message = detail ? `${title}: ${detail}` : title;
		throw new APIError(res.status, message, detail);
	}
	if (res.status === 204) return undefined as T;
	return (await res.json()) as T;
}

export const api = {
	overview: () => request<Overview>('/overview'),
	provider: (id: string) => request<ProviderDetail>(`/providers/${encodeURIComponent(id)}`),
	listProviders: () => request<ProvidersList>('/providers'),
	feedKinds: () => request<FeedKindsList>('/feed-kinds'),
	createProvider: (body: ProviderBody) =>
		request<ProviderDetail>('/providers', {
			method: 'POST',
			body: JSON.stringify(body),
			admin: true
		}),
	updateProvider: (id: string, body: ProviderBody) =>
		request<ProviderDetail>(`/providers/${encodeURIComponent(id)}`, {
			method: 'PUT',
			body: JSON.stringify(body),
			admin: true
		}),
	deleteProvider: (id: string) =>
		request<{ ok: boolean }>(`/providers/${encodeURIComponent(id)}`, {
			method: 'DELETE',
			admin: true
		}),
	validateProvider: (body: ProviderBody) =>
		request<ValidateResult>('/providers/validate', {
			method: 'POST',
			body: JSON.stringify(body),
			admin: true
		})
};
