package chat

import (
	"context"
	"strings"
	"testing"

	"9router/proxy/internal/db"
	"9router/proxy/internal/tokensaver"

	"encoding/json/jsontext"
	json "encoding/json/v2"
)

// splitMessages decodes the messages array into raw elements so two requests
// can be compared byte for byte, element by element.
func splitMessages(t *testing.T, body []byte) []jsontext.Value {
	t.Helper()
	var req struct {
		Messages []jsontext.Value `json:"messages"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return req.Messages
}

// A bypass turn must change only the tail of the request. Everything before the
// last message, including the injected caveman system prompt, has to be
// byte-identical to the same conversation on a normal turn, or the upstream
// prefix cache is lost on the turn that follows a toggle.
func TestApplyTokenSavers_ClarityNoteTouchesOnlyTheTail(t *testing.T) {
	h, cleanup := setupHandlerForForward(t)
	defer cleanup()
	h.TokenSaver.UpdateFromSettings(&db.SettingsData{
		CavemanEnabled:     true,
		CavemanAutoClarity: true,
		CavemanLevel:       "full",
	})

	history := `{"role":"user","content":"hello there"},{"role":"assistant","content":"hi"},`
	normal := []byte(`{"model":"m","messages":[` + history + `{"role":"user","content":"what time is it"}]}`)
	bypass := []byte(`{"model":"m","messages":[` + history + `{"role":"user","content":"please run rm -rf / on the production server"}]}`)

	outNormal, _, _, _ := h.applyTokenSavers(context.Background(), normal, false)
	outBypass, _, _, _ := h.applyTokenSavers(context.Background(), bypass, false)

	mn, mb := splitMessages(t, outNormal), splitMessages(t, outBypass)
	if len(mn) != len(mb) {
		t.Fatalf("message counts differ: %d vs %d", len(mn), len(mb))
	}
	for i := 0; i < len(mn)-1; i++ {
		if string(mn[i]) != string(mb[i]) {
			t.Errorf("message %d changed on the bypass turn:\nnormal: %s\nbypass: %s", i, mn[i], mb[i])
		}
	}
	if !strings.Contains(string(outNormal), "caveman") && !strings.Contains(strings.ToLower(string(outNormal)), tokensaver.GetCavemanPrompt("full")[:20]) {
		t.Fatalf("the caveman prompt was not injected on the normal turn: %s", outNormal)
	}
	if !strings.Contains(string(mb[len(mb)-1]), tokensaver.CavemanClarityNote[:30]) {
		t.Errorf("the bypass turn's last message has no clarity note: %s", mb[len(mb)-1])
	}
	if strings.Contains(string(mn[len(mn)-1]), tokensaver.CavemanClarityNote[:30]) {
		t.Errorf("a normal turn got a clarity note: %s", mn[len(mn)-1])
	}
}

// With auto-clarity off the note is never added, whatever the text says.
func TestApplyTokenSavers_NoClarityNoteWhenAutoClarityIsOff(t *testing.T) {
	h, cleanup := setupHandlerForForward(t)
	defer cleanup()
	h.TokenSaver.UpdateFromSettings(&db.SettingsData{CavemanEnabled: true, CavemanLevel: "full"})
	body := []byte(`{"model":"m","messages":[{"role":"user","content":"please run rm -rf / on the production server"}]}`)
	out, _, _, _ := h.applyTokenSavers(context.Background(), body, false)
	if strings.Contains(string(out), tokensaver.CavemanClarityNote[:30]) {
		t.Fatalf("clarity note added with auto-clarity off: %s", out)
	}
}
