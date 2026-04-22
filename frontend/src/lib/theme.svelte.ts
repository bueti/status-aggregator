export type ThemePref = 'light' | 'dark' | 'system';

const KEY = 'status-agg.theme';

function resolvePref(pref: ThemePref): 'light' | 'dark' {
	if (pref !== 'system') return pref;
	if (typeof window === 'undefined') return 'dark';
	return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

function readStored(): ThemePref {
	if (typeof localStorage === 'undefined') return 'system';
	const v = localStorage.getItem(KEY);
	return v === 'light' || v === 'dark' || v === 'system' ? v : 'system';
}

function apply(pref: ThemePref) {
	if (typeof document === 'undefined') return;
	document.documentElement.setAttribute('data-theme', resolvePref(pref));
}

export function createTheme() {
	let pref = $state<ThemePref>(readStored());

	$effect(() => {
		apply(pref);
		if (typeof localStorage !== 'undefined') {
			if (pref === 'system') localStorage.removeItem(KEY);
			else localStorage.setItem(KEY, pref);
		}
	});

	$effect(() => {
		if (typeof window === 'undefined') return;
		const mql = window.matchMedia('(prefers-color-scheme: dark)');
		const onChange = () => {
			if (pref === 'system') apply(pref);
		};
		mql.addEventListener('change', onChange);
		return () => mql.removeEventListener('change', onChange);
	});

	return {
		get pref() {
			return pref;
		},
		set(next: ThemePref) {
			pref = next;
		}
	};
}
