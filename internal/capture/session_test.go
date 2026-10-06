package capture

import "testing"

func TestTypingSessionPreservesUnicodeAndBackspace(t *testing.T) {
	var got *TypingSessionData
	b := NewTypingSessionBuffer(func(s *TypingSessionData) { got = s })
	for _, key := range []string{"H", "ä", "backspace", "é", "space", "!"} {
		b.AddKey(key, "Notes", "Trial")
	}
	b.Flush()
	if got == nil || got.Text != "Hé !" {
		t.Fatalf("session: %#v", got)
	}
}
