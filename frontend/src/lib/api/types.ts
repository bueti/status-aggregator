import type { components } from './schema';

type S = components['schemas'];

export type Indicator =
	| 'operational'
	| 'maintenance'
	| 'minor'
	| 'major'
	| 'critical'
	| 'unknown';

export type Kind = 'statuspage_io' | string;

export type Component = S['Component'];
export type Incident = S['Incident'];
export type ProviderSummary = S['ProviderSummary'];
export type ProviderDetail = S['ProviderDetail'];
export type ParamField = S['ParamField'];
export type FeedKindInfo = S['FeedKindInfo'];
export type ProviderBody = S['ProviderBody'];

export type Overview = S['OverviewOutputBody'];
export type ProvidersList = S['ListProvidersOutputBody'];
export type FeedKindsList = S['FeedKindsOutputBody'];
export type ValidateResult = S['ValidateProviderOutputBody'];

export function asIndicator(s: string | undefined | null): Indicator {
	switch (s) {
		case 'operational':
		case 'maintenance':
		case 'minor':
		case 'major':
		case 'critical':
			return s;
		default:
			return 'unknown';
	}
}

export const INDICATORS: Indicator[] = [
	'operational',
	'maintenance',
	'minor',
	'major',
	'critical',
	'unknown'
];

export const INDICATOR_LABEL: Record<Indicator, string> = {
	operational: 'Operational',
	maintenance: 'Maintenance',
	minor: 'Minor',
	major: 'Major',
	critical: 'Critical',
	unknown: 'Unknown'
};

export const INDICATOR_CLASS: Record<Indicator, string> = {
	operational: 'bg-ok/20 text-ok ring-ok/30',
	maintenance: 'bg-maintenance/20 text-maintenance ring-maintenance/30',
	minor: 'bg-minor/20 text-minor ring-minor/30',
	major: 'bg-major/20 text-major ring-major/30',
	critical: 'bg-critical/20 text-critical ring-critical/30',
	unknown: 'bg-unknown/20 text-unknown ring-unknown/30'
};
