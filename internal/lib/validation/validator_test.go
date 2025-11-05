package validation

import "testing"

func TestHasDangerousCharacters(t *testing.T) {
	type args struct {
		input string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Safe: Plain alphanumeric text",
			args: args{input: "username123"},
			want: false,
		},
		{
			name: "Safe: Text with safe special characters",
			args: args{input: "This is a good string with spaces, commas, and dots."},
			want: false,
		},
		{
			name: "Safe: Empty string",
			args: args{input: ""},
			want: false,
		},

		{
			name: "Dangerous: HTML Tag Open",
			args: args{input: "<div>Test<"},
			want: true,
		},
		{
			name: "Dangerous: HTML Tag Close",
			args: args{input: "Test>"},
			want: true,
		},
		{
			name: "Dangerous: 'script' keyword (lowercase)",
			args: args{input: "hello script world"},
			want: true,
		},
		{
			name: "Dangerous: 'SCRIPT' keyword (uppercase)",
			args: args{input: "hello SCRIPT world"},
			want: true,
		},
		{
			name: "Dangerous: 'JavaScript:' URI scheme (mixed case)",
			args: args{input: "a href='JAVASCRIPT:alert(1)'"},
			want: true,
		},
		{
			name: "Dangerous: 'onload' event handler",
			args: args{input: "<body onload=alert(1)>"},
			want: true,
		},
		{
			name: "Dangerous: 'onerror' event handler",
			args: args{input: "img onerror='do_something()'"},
			want: true,
		},

		// --- SQL Injection Vectors (Should be true) ---
		{
			name: "Dangerous: Single quote",
			args: args{input: "O'Malley"},
			want: true,
		},
		{
			name: "Dangerous: Double quote",
			args: args{input: "id=\"1\""},
			want: true,
		},
		{
			name: "Dangerous: SQL Comment/Dash",
			args: args{input: "username'--"},
			want: true,
		},
		{
			name: "Dangerous: C-Style Comment Open",
			args: args{input: "id=1 /* comment"},
			want: true,
		},
		{
			name: "Dangerous: C-Style Comment Close",
			args: args{input: "comment */"},
			want: true,
		},
		{
			name: "Dangerous: Semicolon",
			args: args{input: "user=admin;DROP TABLE"},
			want: true,
		},
		{
			name: "Dangerous: Ampersand (command separator)",
			args: args{input: "cmd=ls&cat /etc/passwd"},
			want: true,
		},
		{
			name: "Dangerous: Pipe (command separator)",
			args: args{input: "cmd=ls|cat /etc/passwd"},
			want: true,
		},
		{
			name: "Dangerous: Newline character",
			args: args{input: "multi\nline"},
			want: true,
		},
		{
			name: "Dangerous: Carriage Return character",
			args: args{input: "multi\rline"},
			want: true,
		},

		// --- Path Traversal / LFI / Remote Content (Should be true) ---
		{
			name: "Dangerous: Path Traversal '.. /'",
			args: args{input: "filename=../etc/passwd"},
			want: true,
		},
		{
			name: "Dangerous: 'file://' scheme (LFI)",
			args: args{input: "source=file://server/share"},
			want: true,
		},
		{
			name: "Dangerous: 'http://' scheme (SSRF)",
			args: args{input: "url=http://malicious.com"},
			want: true,
		},
		{
			name: "Dangerous: 'https://' scheme (SSRF)",
			args: args{input: "url=https://malicious.com"},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasDangerousCharacters(tt.args.input); got != tt.want {
				t.Errorf("HasDangerousCharacters() = %v, want %v", got, tt.want)
			}
		})
	}
}
