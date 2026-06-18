package oauth

import (
	"testing"
)

func TestDeriveAuthBaseURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "production API",
			input: "https://api.rootly.com",
			want:  "https://rootly.com",
		},
		{
			name:  "staging API",
			input: "https://api.staging.rootly.com",
			want:  "https://staging.rootly.com",
		},
		{
			name:  "localhost",
			input: "http://localhost:22166",
			want:  "http://localhost:22166",
		},
		{
			name:  "127.0.0.1",
			input: "http://127.0.0.1:3000",
			want:  "http://127.0.0.1:3000",
		},
		{
			name:  "no scheme",
			input: "api.rootly.com",
			want:  "https://rootly.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeriveAuthBaseURL(tt.input)
			if got != tt.want {
				t.Errorf("DeriveAuthBaseURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
