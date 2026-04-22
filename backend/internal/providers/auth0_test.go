package providers

import (
	"strings"
	"testing"
)

func TestExtractAuth0NextData(t *testing.T) {
	t.Run("extracts payload between tags", func(t *testing.T) {
		body := []byte(`<html><head>...</head><body>` +
			`<script id="__NEXT_DATA__" type="application/json">{"a":1}</script>` +
			`</body></html>`)
		got, err := extractAuth0NextData(body)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(got) != `{"a":1}` {
			t.Errorf("got %q, want %q", got, `{"a":1}`)
		}
	})

	t.Run("errors when open tag is missing", func(t *testing.T) {
		_, err := extractAuth0NextData([]byte(`<html>no script here</html>`))
		if err == nil {
			t.Fatal("want error")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("errors when close tag is missing", func(t *testing.T) {
		_, err := extractAuth0NextData([]byte(
			`<script id="__NEXT_DATA__" type="application/json">{"a":1}`))
		if err == nil {
			t.Fatal("want error")
		}
		if !strings.Contains(err.Error(), "close tag") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("handles large adversarial input without catastrophic backtracking", func(t *testing.T) {
		// An earlier implementation used `(?s)<script...>(.*?)</script>`, which
		// is O(n^2) on input that opens many <script> tags without a matching
		// id="__NEXT_DATA__". This test would hang under that implementation;
		// bytes.Cut is O(n).
		var b strings.Builder
		b.WriteString("<html>")
		for range 200_000 {
			b.WriteString(`<script>noop</script>`)
		}
		b.WriteString(`<script id="__NEXT_DATA__" type="application/json">{"ok":true}</script></html>`)
		got, err := extractAuth0NextData([]byte(b.String()))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(got) != `{"ok":true}` {
			t.Errorf("got %q", got)
		}
	})
}
