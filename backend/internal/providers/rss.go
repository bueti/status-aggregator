package providers

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Generic RSS 2.0 / Atom status feed adapter. Works for any status page that
// publishes incidents as a feed — Slack's https://slack-status.com/feed/rss,
// Google Workspace's https://www.google.com/appsstatus/dashboard/en/feed.atom,
// etc.
//
// Determining the *current* status from a feed is inherently a heuristic: feeds
// carry a history, not a liveness flag. The adapter:
//   1. fetches the feed,
//   2. considers entries whose pubDate/updated falls within `active_hours`,
//   3. classifies each by title prefix (Outage/Incident/Notice/Resolved/...),
//   4. reports the worst non-resolved impact as the indicator.
// If no entries fall inside the active window, or all recent entries are
// prefixed "RESOLVED", the source is reported as operational.

func init() {
	Register(&rssFactory{})
}

type rssFactory struct{}

func (f *rssFactory) Kind() Kind    { return KindRSS }
func (f *rssFactory) Label() string { return "RSS / Atom feed" }

func (f *rssFactory) Fields() []ParamField {
	return []ParamField{
		{
			Name:        "feed_url",
			Label:       "Feed URL",
			Type:        "url",
			Placeholder: "https://slack-status.com/feed/rss",
			Required:    true,
			Help:        "RSS 2.0 or Atom feed URL.",
		},
		{
			Name:        "active_hours",
			Label:       "Active window (hours)",
			Type:        "number",
			Placeholder: "24",
			Help:        "Entries within this many hours of now count as active. Older entries are treated as resolved. Default 24.",
		},
		{
			Name:  "link",
			Label: "Home URL (optional)",
			Type:  "url",
			Help:  "Optional public URL for this status page. Falls back to the feed's channel/alternate link.",
		},
	}
}

type rssParams struct {
	FeedURL     string          `json:"feed_url"`
	ActiveHours jsonIntOrString `json:"active_hours"`
	Link        string          `json:"link"`
}

// jsonIntOrString accepts either a JSON number or a JSON string containing an
// integer. The admin form posts values as strings, but we want the stored
// representation to read as a number too.
type jsonIntOrString int

func (x *jsonIntOrString) UnmarshalJSON(b []byte) error {
	if len(b) == 0 || string(b) == "null" {
		*x = 0
		return nil
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		s = strings.TrimSpace(s)
		if s == "" {
			*x = 0
			return nil
		}
		n, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("active_hours: %w", err)
		}
		*x = jsonIntOrString(n)
		return nil
	}
	var n int
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*x = jsonIntOrString(n)
	return nil
}

func (f *rssFactory) parse(cfg Config) (rssParams, error) {
	var p rssParams
	if len(cfg.Params) == 0 {
		return p, fmt.Errorf("params are required")
	}
	if err := json.Unmarshal(cfg.Params, &p); err != nil {
		return p, fmt.Errorf("invalid params: %w", err)
	}
	u, err := parseHTTPURL("feed_url", p.FeedURL)
	if err != nil {
		return p, err
	}
	p.FeedURL = u.String()
	return p, nil
}

// schemePrefixRE matches an RFC 3986 scheme followed by "://" at the start of
// a string. Used to distinguish "user already typed a scheme" from the bare
// hostname / path case.
var schemePrefixRE = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.\-]*://`)

// parseHTTPURL trims whitespace, auto-prepends https:// to scheme-less input
// (the common paste-a-bare-hostname case), and returns the parsed URL. Errors
// are prefixed with field so callers don't need to wrap.
func parseHTTPURL(field, s string) (*url.URL, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("%s is required", field)
	}
	// "//host/path" (scheme-relative) and "host/path" (bare) both become
	// https://host/path. Any existing scheme is left to url.Parse so a non-http
	// scheme is reported as "must be http or https" rather than silently
	// wrapped in a second https://.
	if !schemePrefixRE.MatchString(s) {
		s = "https://" + strings.TrimPrefix(s, "//")
	}
	u, err := url.Parse(s)
	if err != nil {
		return nil, fmt.Errorf("%s: invalid URL: %w", field, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("%s: must be http or https (got %q)", field, u.Scheme)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("%s: must include a host", field)
	}
	return u, nil
}

func (f *rssFactory) Build(cfg Config) (Provider, error) {
	p, err := f.parse(cfg)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("marshal params: %w", err)
	}
	cfg.Params = raw
	return &rssProvider{cfg: cfg, params: p, client: sharedHTTP}, nil
}

type rssProvider struct {
	cfg    Config
	params rssParams
	client *http.Client
}

func (p *rssProvider) Config() Config { return p.cfg }

func (p *rssProvider) Fetch(ctx context.Context) (Status, error) {
	feed, err := fetchFeed(ctx, p.client, p.params.FeedURL)
	if err != nil {
		return Status{}, err
	}
	window := time.Duration(int(p.params.ActiveHours)) * time.Hour
	if window <= 0 {
		window = 24 * time.Hour
	}
	return feedToStatus(feed, window), nil
}

// --- wire format (RSS 2.0 + Atom) ---

type parsedFeed struct {
	Title   string
	Link    string
	Entries []feedEntry
}

type feedEntry struct {
	ID      string
	Title   string
	Link    string
	Body    string
	Updated time.Time
}

type rssEnvelope struct {
	XMLName xml.Name `xml:"rss"`
	Channel struct {
		Title string `xml:"title"`
		Link  string `xml:"link"`
		Items []struct {
			Title       string `xml:"title"`
			Link        string `xml:"link"`
			GUID        string `xml:"guid"`
			PubDate     string `xml:"pubDate"`
			Description string `xml:"description"`
		} `xml:"item"`
	} `xml:"channel"`
}

type atomEnvelope struct {
	XMLName xml.Name `xml:"feed"`
	Title   string   `xml:"title"`
	Links   []struct {
		Href string `xml:"href,attr"`
		Rel  string `xml:"rel,attr"`
	} `xml:"link"`
	Entries []struct {
		Title   string `xml:"title"`
		ID      string `xml:"id"`
		Updated string `xml:"updated"`
		Links   []struct {
			Href string `xml:"href,attr"`
			Rel  string `xml:"rel,attr"`
		} `xml:"link"`
		Summary string `xml:"summary"`
		Content string `xml:"content"`
	} `xml:"entry"`
}

func fetchFeed(ctx context.Context, client *http.Client, feedURL string) (*parsedFeed, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml;q=0.9, */*;q=0.5")
	req.Header.Set("User-Agent", "status-aggregator/0.1")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("feed %s: HTTP %d", feedURL, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	return parseFeed(body)
}

func parseFeed(body []byte) (*parsedFeed, error) {
	// RSS 2.0 first
	var rss rssEnvelope
	if err := xml.Unmarshal(body, &rss); err == nil && rss.XMLName.Local == "rss" {
		out := &parsedFeed{
			Title: strings.TrimSpace(rss.Channel.Title),
			Link:  strings.TrimSpace(rss.Channel.Link),
		}
		for _, it := range rss.Channel.Items {
			out.Entries = append(out.Entries, feedEntry{
				ID:      firstNonEmpty(it.GUID, it.Link, it.Title),
				Title:   normalizeText(it.Title),
				Link:    strings.TrimSpace(it.Link),
				Body:    it.Description,
				Updated: parseFeedTime(it.PubDate),
			})
		}
		return out, nil
	}
	// Atom fallback
	var atom atomEnvelope
	if err := xml.Unmarshal(body, &atom); err == nil && atom.XMLName.Local == "feed" {
		out := &parsedFeed{Title: strings.TrimSpace(atom.Title)}
		for _, l := range atom.Links {
			if l.Rel == "" || l.Rel == "alternate" {
				out.Link = l.Href
				break
			}
		}
		for _, e := range atom.Entries {
			link := ""
			for _, l := range e.Links {
				if l.Rel == "" || l.Rel == "alternate" {
					link = l.Href
					break
				}
			}
			out.Entries = append(out.Entries, feedEntry{
				ID:      firstNonEmpty(e.ID, link, e.Title),
				Title:   normalizeText(e.Title),
				Link:    link,
				Body:    firstNonEmpty(e.Summary, e.Content),
				Updated: parseFeedTime(e.Updated),
			})
		}
		return out, nil
	}
	return nil, fmt.Errorf("unrecognized feed format (expected RSS 2.0 <rss> or Atom <feed>)")
}

// --- classification ---

func feedToStatus(feed *parsedFeed, window time.Duration) Status {
	out := Status{
		Components: []Component{},
		Incidents:  make([]Incident, 0),
		FetchedAt:  time.Now().UTC(),
	}
	cutoff := time.Now().Add(-window)

	indicators := []Indicator{}
	for _, e := range feed.Entries {
		// If we couldn't parse the time, include the entry but treat it as
		// inside the window so operators still see it.
		if !e.Updated.IsZero() && e.Updated.Before(cutoff) {
			continue
		}
		cls := classifyFeedEntry(e.Title, e.Body)
		if cls.resolved {
			continue
		}
		indicators = append(indicators, cls.indicator)
		out.Incidents = append(out.Incidents, Incident{
			ID:        e.ID,
			Name:      e.Title,
			Status:    cls.label,
			Impact:    cls.indicator,
			URL:       e.Link,
			UpdatedAt: e.Updated,
		})
	}

	out.Indicator = WorstIndicator(indicators)
	switch out.Indicator {
	case IndicatorOperational:
		out.Description = "No recent incidents"
	case IndicatorMaintenance:
		out.Description = "Scheduled maintenance"
	default:
		if len(out.Incidents) == 1 {
			out.Description = "1 recent incident"
		} else {
			out.Description = fmt.Sprintf("%d recent incidents", len(out.Incidents))
		}
	}
	return out
}

type titleClass struct {
	indicator Indicator
	label     string
	resolved  bool
}

var (
	resolvedPrefixRE  = regexp.MustCompile(`(?i)^(resolved|fixed|closed|completed)\b`)
	outagePrefixRE    = regexp.MustCompile(`(?i)^(outage|major\s+outage|critical)\b`)
	incidentPrefixRE  = regexp.MustCompile(`(?i)^(incident|issue|disruption|degraded)\b`)
	noticePrefixRE    = regexp.MustCompile(`(?i)^(notice|advisory|info|information)\b`)
	maintenancePrefRE = regexp.MustCompile(`(?i)^(scheduled|maintenance|planned)\b`)
)

// resolutionBodyRE matches past-tense markers in the description that
// indicate the incident has been resolved, even when the feed's title was
// posted at the start and never updated (Slack's RSS works this way — the
// item title stays "Incident: foo", but the body narrates the resolution).
// Deliberately narrow to avoid false positives from active updates like
// "we are working to resolve this".
var resolutionBodyRE = regexp.MustCompile(
	`(?i)\b(` +
		`(has|have) been (resolved|fixed|mitigated|restored)|` +
		`is (now|fully) (resolved|operational|restored)|` +
		`issue (has been )?resolved|` +
		`incident (has been |is now )?resolved|` +
		`(resolution|resolved) at\b|` +
		`^resolved\b|` + // Statuspage.io body often starts with "Resolved - ..."
		`fully (caught up|restored|operational)` +
		`)`)

func classifyFeedEntry(title, body string) titleClass {
	t := strings.TrimSpace(title)

	// Title prefix is the primary signal.
	switch {
	case resolvedPrefixRE.MatchString(t):
		return titleClass{indicator: IndicatorOperational, label: "resolved", resolved: true}
	case maintenancePrefRE.MatchString(t):
		return titleClass{indicator: IndicatorMaintenance, label: "maintenance"}
	}

	// Body can upgrade an "active-looking" title to resolved when the narrative
	// clearly describes a completed incident.
	if body != "" && resolutionBodyRE.MatchString(body) {
		return titleClass{indicator: IndicatorOperational, label: "resolved", resolved: true}
	}

	switch {
	case outagePrefixRE.MatchString(t):
		return titleClass{indicator: IndicatorCritical, label: "outage"}
	case incidentPrefixRE.MatchString(t):
		return titleClass{indicator: IndicatorMajor, label: "incident"}
	case noticePrefixRE.MatchString(t):
		return titleClass{indicator: IndicatorMinor, label: "notice"}
	}
	// Unknown prefix — assume minor so the operator notices, but don't escalate.
	return titleClass{indicator: IndicatorMinor, label: "update"}
}

// --- helpers ---

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if t := strings.TrimSpace(s); t != "" {
			return t
		}
	}
	return ""
}

func normalizeText(s string) string {
	s = strings.TrimSpace(s)
	// Collapse runs of whitespace (feeds often pretty-print titles).
	return regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
}

func parseFeedTime(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	layouts := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z",
		"Mon, 2 Jan 2006 15:04:05 -0700",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}
