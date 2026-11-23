package helper

import "testing"

func TestIsEditableTextBytes(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{"empty", []byte{}, true},
		{"ASCII", []byte("sample\ntext"), true},
		{"UTF8 Chinese", []byte("中文内容\n"), true},
		{"invalid UTF8", []byte{'a', 0xff, 'b'}, false},
		{"truncated UTF8", []byte{0xe4, 0xb8}, false},
		{"binary NUL", []byte{'a', 0, 'b'}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsEditableTextBytes(tt.data); got != tt.want {
				t.Fatalf("IsEditableTextBytes(%q) = %v, want %v", tt.data, got, tt.want)
			}
		})
	}
}
