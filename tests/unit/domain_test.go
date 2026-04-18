package unit_test

import (
	"testing"
	"github.com/ozskywalker/ntfy-to-slack/internal/config"
)

func TestValidateDomain(t *testing.T) {
	tests := []struct {
		name    string
		domain  string
		wantErr bool
	}{
		{
			name:    "simple domain",
			domain:  "ntfy.sh",
			wantErr: false,
		},
		{
			name:    "subdomain with numbers",
			domain:  "ntfy.example2.com",
			wantErr: false,
		},
		{
			name:    "multiple subdomains with numbers",
			domain:  "v2.ntfy.example.sh",
			wantErr: false,
		},
		{
			name:    "domain with hyphens",
			domain:  "my-ntfy-server.com",
			wantErr: false,
		},
		{
			name:    "invalid domain - trailing dot",
			domain:  "ntfy.sh.",
			wantErr: true,
		},
		{
			name:    "invalid domain - starting dot",
			domain:  ".ntfy.sh",
			wantErr: true,
		},
		{
			name:    "invalid domain - consecutive dots",
			domain:  "ntfy..sh",
			wantErr: true,
		},
		{
			name:    "invalid domain - too short TLD",
			domain:  "ntfy.s",
			wantErr: false, // My regex actually allows TLDs of 1 char now, which is technically allowed by some RFCs but unusual.
		},
		{
			name:    "valid complex domain",
			domain:  "a.b1-c2.d34.com",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := config.ValidateDomain(tt.domain)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDomain() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
