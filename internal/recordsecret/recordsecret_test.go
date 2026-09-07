package recordsecret

import (
	"context"
	"encoding/json"
	"github.com/hapyco/dygo/internal/secrets"
	"strings"
	"testing"
)

func testStore(t *testing.T) secrets.Store {
	t.Helper()
	s := secrets.NewStore(t.TempDir())
	if _, err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := Init(s, secrets.EnvironmentDevelopment); err != nil {
		t.Fatal(err)
	}
	return s
}
func TestEncryptionAndKeyLifecycle(t *testing.T) {
	s := testStore(t)
	env := secrets.EnvironmentDevelopment
	r, err := Load(s, env)
	if err != nil {
		t.Fatal(err)
	}
	if err = Init(s, env); err != nil {
		t.Fatal(err)
	}
	same, _ := Load(s, env)
	if same.Active != r.Active {
		t.Fatal("init replaced key")
	}
	first, err := r.Encrypt("crm/account/token", "sensitive-value")
	if err != nil {
		t.Fatal(err)
	}
	second, _ := r.Encrypt("crm/account/token", "sensitive-value")
	if first == second || strings.Contains(first, "sensitive-value") {
		t.Fatal("ciphertext is not randomized and opaque")
	}
	value, err := r.Decrypt("crm/account/token", first)
	if err != nil || value != "sensitive-value" {
		t.Fatal("round trip failed")
	}
	if _, err = r.Decrypt("crm/account/other", first); err == nil {
		t.Fatal("accepted another field's ciphertext")
	}
	for _, raw := range []string{"plaintext", `{"v":2,"key":"x","data":"x"}`, strings.Replace(first, `"data":"`, `"data":"!`, 1)} {
		if _, err = r.Decrypt("crm/account/token", raw); err == nil {
			t.Fatal("accepted malformed ciphertext")
		}
	}
	rotated, err := BeginRotation(s, env)
	if err != nil {
		t.Fatal(err)
	}
	resumed, err := BeginRotation(s, env)
	if err != nil || resumed.Active != rotated.Active || !resumed.Rotating {
		t.Fatal("rotation did not resume")
	}
	if value, err = rotated.Decrypt("crm/account/token", first); err != nil || value != "sensitive-value" {
		t.Fatal("old backup key lost")
	}
	if err = FinishRotation(s, env, rotated); err != nil {
		t.Fatal(err)
	}
	finished, _ := Load(s, env)
	if finished.Rotating || len(finished.Keys) != 2 {
		t.Fatal("bad completed key ring")
	}
	doc, err := s.Load(env)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := doc.Values[configKey].(map[string]any); !ok {
		t.Fatal("key ring must be structured encrypted YAML")
	}
	encoded, _ := json.Marshal(doc.Values[configKey])
	if !strings.Contains(string(encoded), "active") {
		t.Fatal("missing key data")
	}
}
func TestMissingAndCorruptKeysFailClosed(t *testing.T) {
	if _, err := FromContext(context.Background()); err == nil {
		t.Fatal("missing provider accepted")
	}
	s := testStore(t)
	doc, _ := s.Load(secrets.EnvironmentDevelopment)
	doc.Values[configKey] = map[string]any{"active": "bad", "keys": map[string]any{"bad": "private-invalid-value"}}
	if err := s.Save(secrets.EnvironmentDevelopment, doc); err != nil {
		t.Fatal(err)
	}
	if err := Init(s, secrets.EnvironmentDevelopment); err == nil || strings.Contains(err.Error(), "private-invalid-value") {
		t.Fatal("corrupt keys overwritten or leaked")
	}
}
