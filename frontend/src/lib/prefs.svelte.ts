export type SortMode = 'name' | 'added' | 'status';
export type ViewMode = 'grid' | 'list';

const SORT_KEY = 'status-agg.sort';
const VIEW_KEY = 'status-agg.view';

function readSort(): SortMode {
	if (typeof localStorage === 'undefined') return 'name';
	const v = localStorage.getItem(SORT_KEY);
	return v === 'name' || v === 'added' || v === 'status' ? v : 'name';
}

function readView(): ViewMode {
	if (typeof localStorage === 'undefined') return 'grid';
	const v = localStorage.getItem(VIEW_KEY);
	return v === 'grid' || v === 'list' ? v : 'grid';
}

export function createOverviewPrefs() {
	let sort = $state<SortMode>(readSort());
	let view = $state<ViewMode>(readView());

	$effect(() => {
		if (typeof localStorage !== 'undefined') localStorage.setItem(SORT_KEY, sort);
	});
	$effect(() => {
		if (typeof localStorage !== 'undefined') localStorage.setItem(VIEW_KEY, view);
	});

	return {
		get sort() {
			return sort;
		},
		set sort(v: SortMode) {
			sort = v;
		},
		get view() {
			return view;
		},
		set view(v: ViewMode) {
			view = v;
		}
	};
}

// Worst-first indicator ranking for sorting. Matches the backend Indicator.Rank
// order (operational < maintenance < minor < major < critical) and treats
// unknown as its own bucket below operational — an unknown status usually
// means a fetch error, which operators care about but not as much as a known
// critical outage.
const INDICATOR_SORT: Record<string, number> = {
	critical: 5,
	major: 4,
	minor: 3,
	maintenance: 2,
	operational: 1,
	unknown: 0
};

export function sortProviders<T extends { name: string; indicator: string; created_at: string }>(
	providers: T[],
	mode: SortMode
): T[] {
	const copy = providers.slice();
	switch (mode) {
		case 'name':
			copy.sort((a, b) => a.name.localeCompare(b.name));
			break;
		case 'added':
			copy.sort((a, b) => {
				const at = Date.parse(a.created_at) || 0;
				const bt = Date.parse(b.created_at) || 0;
				return bt - at;
			});
			break;
		case 'status':
			copy.sort((a, b) => {
				const ar = INDICATOR_SORT[a.indicator] ?? 0;
				const br = INDICATOR_SORT[b.indicator] ?? 0;
				if (ar !== br) return br - ar;
				return a.name.localeCompare(b.name);
			});
			break;
	}
	return copy;
}
