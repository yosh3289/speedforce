package i18n

import "testing"

func TestT_ReturnsKey(t *testing.T) {
	tr, err := New("en")
	if err != nil {
		t.Fatal(err)
	}
	if got := tr.T("app_name"); got != "SpeedForce" {
		t.Errorf("got %q", got)
	}
}

func TestT_ChineseLoad(t *testing.T) {
	tr, err := New("zh")
	if err != nil {
		t.Fatal(err)
	}
	if got := tr.T("tray.menu.quit"); got != "退出" {
		t.Errorf("got %q", got)
	}
}

func TestT_FallbackToEn(t *testing.T) {
	tr, err := New("zz")
	if err != nil {
		t.Fatal(err)
	}
	if got := tr.T("app_name"); got != "SpeedForce" {
		t.Errorf("expected fallback: got %q", got)
	}
}

func TestT_MissingKeyReturnsKey(t *testing.T) {
	tr, _ := New("en")
	if got := tr.T("nonexistent.key"); got != "nonexistent.key" {
		t.Errorf("missing key should return key itself, got %q", got)
	}
}

func TestT_Interpolation(t *testing.T) {
	tr, _ := New("en")
	got := tr.T("notify.service_down", map[string]string{"name": "Claude"})
	if got != "Claude is unreachable" {
		t.Errorf("got %q", got)
	}
}
