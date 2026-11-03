package failure

import "testing"

func TestProbeRealMessages(t *testing.T) {
	for msg, want := range map[string]Kind{
		"Error: Free usage limit reached. Connect a provider or subscribe to OpenCode Go to keep going, or switch models.": Quota,
		"ERROR: Selected model is at capacity. Please try a different model.":                                              Transient,
	} {
		if got := Classify(1, msg); got != want {
			t.Errorf("%q -> %q, want %q", msg, got, want)
		}
	}
}

func TestProbeTrust(t *testing.T) {
	if got := Classify(1, "Gemini CLI is not running in a trusted directory. To proceed, use --skip-trust"); got != Auth {
		t.Fatalf("got %q", got)
	}
}
