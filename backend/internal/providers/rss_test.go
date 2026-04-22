package providers

import "testing"

func TestNormalizeHTTPURL(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{in: "https://example.com/feed.rss", want: "https://example.com/feed.rss"},
		{in: "http://example.com/feed.rss", want: "http://example.com/feed.rss"},
		{in: " https://example.com/feed.rss ", want: "https://example.com/feed.rss"},
		{in: "example.com/feed.rss", want: "https://example.com/feed.rss"},
		{in: "status.auth0.com/rss?domain=x.y.z", want: "https://status.auth0.com/rss?domain=x.y.z"},
		{in: "//example.com/feed", want: "https://example.com/feed"},
		{in: "", wantErr: true},
		{in: "   ", wantErr: true},
		{in: "ftp://example.com/feed", wantErr: true},
		{in: "://nope", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, err := normalizeHTTPURL(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Errorf("want error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestClassifyFeedEntry(t *testing.T) {
	cases := []struct {
		name     string
		title    string
		body     string
		want     Indicator
		resolved bool
	}{
		{
			name:  "bare incident title, no body",
			title: "Incident: Custom status not expiring",
			want:  IndicatorMajor,
		},
		{
			// The reported Slack case: title stays "Incident: ...", but the
			// body narrates the resolution.
			name:     "incident title with resolution in body (slack)",
			title:    "Incident: Custom status not expiring",
			body:     `<p>We're aware of an issue affecting Slack. ...</p><p>Resolved</p><p>This issue is now resolved for all users.</p>`,
			want:     IndicatorOperational,
			resolved: true,
		},
		{
			name:  "outage prefix, empty body",
			title: "Outage: Slack Connectivity issue",
			want:  IndicatorCritical,
		},
		{
			name:  "notice prefix classifies as minor",
			title: "Notice: Brief advisory about rate limits",
			want:  IndicatorMinor,
		},
		{
			name:  "scheduled prefix classifies as maintenance",
			title: "Scheduled maintenance: database upgrade",
			want:  IndicatorMaintenance,
		},
		{
			name:     "atom title already says RESOLVED (google workspace)",
			title:    "RESOLVED: Customers may experience delays in Gmail",
			want:     IndicatorOperational,
			resolved: true,
		},
		{
			name:  "active incident mentions 'working to resolve' — not resolved",
			title: "Incident: Trouble sending messages",
			body:  "We are aware of an issue and are working to resolve this.",
			want:  IndicatorMajor,
		},
		{
			name:     "statuspage-style body starts with 'Resolved -'",
			title:    "Some service feature",
			body:     "Resolved - The issue has been identified and fixed.",
			want:     IndicatorOperational,
			resolved: true,
		},
		{
			name:     "body says 'fully caught up' (slack postmortem idiom)",
			title:    "Incident: Search backlog",
			body:     "The service was fully caught up and returning all results by 4:19 PM PDT.",
			want:     IndicatorOperational,
			resolved: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyFeedEntry(tc.title, tc.body)
			if got.indicator != tc.want {
				t.Errorf("indicator = %s, want %s", got.indicator, tc.want)
			}
			if got.resolved != tc.resolved {
				t.Errorf("resolved = %v, want %v", got.resolved, tc.resolved)
			}
		})
	}
}
