package rand

import (
	"encoding/hex"
	"testing"
)

func TestGenerateRandID(t *testing.T) {
	tests := []struct {
		name    string
		want    string
		wantErr bool
	}{
		{
			name:    "Successful ID generation and format check",
			wantErr: false,
		},
	}

	const expectedLength = 32

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateRandID()
			if (err != nil) != tt.wantErr {
				t.Fatalf("GenerateRandID() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			if len(got) != expectedLength {
				t.Errorf("GenerateRandID() length mismatch: got %v, want %v", len(got), expectedLength)
			}

			_, err = hex.DecodeString(got)
			if err != nil {
				t.Errorf("GenerateRandID() returned a non-hex string: %v", err)
			}
		})
	}
}
