package outfmt

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"
)

func TestFromFlags(t *testing.T) {
	tests := []struct {
		name    string
		json    bool
		plain   bool
		want    Mode
		wantErr bool
	}{
		{
			name: "default mode",
			want: Mode{},
		},
		{
			name: "json mode",
			json: true,
			want: Mode{JSON: true},
		},
		{
			name:  "plain mode",
			plain: true,
			want:  Mode{Plain: true},
		},
		{
			name:    "conflict",
			json:    true,
			plain:   true,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FromFlags(tt.json, tt.plain)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Errorf("FromFlags(%v, %v) = %v, want %v", tt.json, tt.plain, got, tt.want)
			}
		})
	}
}

func TestFromEnv(t *testing.T) {
	t.Setenv("TEST_CLI_JSON", "true")
	t.Setenv("TEST_CLI_PLAIN", "false")

	mode := FromEnv("TEST_CLI")

	if !mode.JSON {
		t.Error("expected JSON to be true")
	}

	if mode.Plain {
		t.Error("expected Plain to be false")
	}
}

func TestFromEnv_YesValues(t *testing.T) {
	yesValues := []string{"1", "true", "yes", "y", "on", "TRUE", "Yes", "ON"}

	for _, v := range yesValues {
		t.Run(v, func(t *testing.T) {
			t.Setenv("TEST_CLI_ENVBOOL_JSON", v)

			mode := FromEnv("TEST_CLI_ENVBOOL")
			if !mode.JSON {
				t.Errorf("expected %q to parse as true", v)
			}
		})
	}
}

func TestFromEnv_NoValues(t *testing.T) {
	noValues := []string{"0", "false", "no", "n", "off", ""}

	for _, v := range noValues {
		t.Run("value_"+v, func(t *testing.T) {
			if v != "" {
				t.Setenv("TEST_CLI_ENVBOOL2_JSON", v)
			} else {
				os.Unsetenv("TEST_CLI_ENVBOOL2_JSON")
			}

			mode := FromEnv("TEST_CLI_ENVBOOL2")
			if mode.JSON {
				t.Errorf("expected %q to parse as false", v)
			}
		})
	}
}

func TestWithMode_RoundTrip(t *testing.T) {
	mode := Mode{JSON: true, Plain: false}
	ctx := WithMode(context.Background(), mode)

	got := FromContext(ctx)

	if got != mode {
		t.Errorf("FromContext = %v, want %v", got, mode)
	}
}

func TestFromContext_NoMode(t *testing.T) {
	ctx := context.Background()
	got := FromContext(ctx)

	if got.JSON || got.Plain {
		t.Errorf("empty context should return zero Mode, got %v", got)
	}
}

func TestIsJSON(t *testing.T) {
	ctx := WithMode(context.Background(), Mode{JSON: true})

	if !IsJSON(ctx) {
		t.Error("IsJSON should return true")
	}

	if IsPlain(ctx) {
		t.Error("IsPlain should return false")
	}
}

func TestIsPlain(t *testing.T) {
	ctx := WithMode(context.Background(), Mode{Plain: true})

	if !IsPlain(ctx) {
		t.Error("IsPlain should return true")
	}

	if IsJSON(ctx) {
		t.Error("IsJSON should return false")
	}
}

func TestWriteJSON(t *testing.T) {
	var buf bytes.Buffer

	data := map[string]any{
		"name":  "test",
		"count": 42,
	}

	err := WriteJSON(&buf, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if decoded["name"] != "test" {
		t.Errorf("expected name 'test', got %v", decoded["name"])
	}

	if decoded["count"] != float64(42) {
		t.Errorf("expected count 42, got %v", decoded["count"])
	}
}

func TestWriteJSON_NoHTMLEscape(t *testing.T) {
	var buf bytes.Buffer

	data := map[string]string{
		"url": "https://example.com?a=1&b=2",
	}

	err := WriteJSON(&buf, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if bytes.Contains(buf.Bytes(), []byte(`\u0026`)) {
		t.Errorf("expected no HTML escaping, got: %s", buf.String())
	}
}

func TestWriteJSON_Indented(t *testing.T) {
	var buf bytes.Buffer

	data := map[string]string{"key": "value"}

	err := WriteJSON(&buf, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.Contains(buf.Bytes(), []byte("  ")) {
		t.Errorf("expected indented output, got: %s", buf.String())
	}
}

func TestKeyValuePayload(t *testing.T) {
	p := KeyValuePayload("test-key", 42)

	if p["key"] != "test-key" {
		t.Errorf("expected key 'test-key', got %v", p["key"])
	}

	if p["value"] != 42 {
		t.Errorf("expected value 42, got %v", p["value"])
	}
}

func TestKeysPayload(t *testing.T) {
	p := KeysPayload([]string{"a", "b", "c"})

	keys, ok := p["keys"].([]string)
	if !ok {
		t.Fatal("expected keys to be []string")
	}

	if len(keys) != 3 {
		t.Errorf("expected 3 keys, got %d", len(keys))
	}
}

func TestPathPayload(t *testing.T) {
	p := PathPayload("/some/path")

	if p["path"] != "/some/path" {
		t.Errorf("expected path '/some/path', got %v", p["path"])
	}
}

func TestParseError(t *testing.T) {
	err := &ParseError{msg: "test error"}

	if err.Error() != "test error" {
		t.Errorf("expected 'test error', got %q", err.Error())
	}
}
