package wrapper

import (
	"errors"
	"testing"
)

func TestWrap(t *testing.T) {
	baseErr := errors.New("base error for testing")

	type args struct {
		msg string
		err error
	}
	tests := []struct {
		name        string
		args        args
		wantErr     bool
		expectedMsg string
	}{
		{
			name: "Case 1: Nil error should return nil",
			args: args{
				msg: "op: failed to run",
				err: nil,
			},
			wantErr:     false,
			expectedMsg: "",
		},
		{
			name: "Case 2: Non-nil error should return a wrapped error with a message",
			args: args{
				msg: "database access failed",
				err: baseErr,
			},
			wantErr:     true,
			expectedMsg: "database access failed: base error for testing",
		},
		{
			name: "Case 3: Empty message should still wrap the error",
			args: args{
				msg: "",
				err: baseErr,
			},
			wantErr:     true,
			expectedMsg: ": base error for testing",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := Wrap(tt.args.msg, tt.args.err)

			if (gotErr != nil) != tt.wantErr {
				t.Errorf("Wrap() error = %v, wantErr %v", gotErr, tt.wantErr)
				return
			}

			if tt.wantErr {
				if gotErr.Error() != tt.expectedMsg {
					t.Errorf("Wrap() got error message = %q, want %q", gotErr.Error(), tt.expectedMsg)
				}

				if !errors.Is(gotErr, baseErr) {
					t.Errorf("Wrap() returned error does not contain the original error via errors.Is. Got: %v, Want to find: %v", gotErr, baseErr)
				}
			}
		})
	}
}
