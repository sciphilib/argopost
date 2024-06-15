package internal

import (
	"testing"
)

func TestParse(t *testing.T) {
	parser := GeneralCommandParser{}
	tests := []struct {
		input    string
		expected *Command
	}{
		{"HELO example.com", &Command{Type: HeloCommand, Payload: "example.com"}},
		{"MAIL FROM <test@example.com>", &Command{Type: MailFromCommand, Payload: "<test@example.com>"}},
		{"RCPT TO <test@example.com>", &Command{Type: RcptToCommand, Payload: "<test@example.com>"}},
		{"DATA", &Command{Type: DataCommand, Payload: ""}},
		{"QUIT", &Command{Type: QuitCommand, Payload: ""}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			cmd := parser.Parse(tt.input)
			if !compareCommands(cmd, tt.expected) {
				t.Errorf("Parse(%q) = %q, want %q", tt.input, cmd, tt.expected)
			}
		})
	}
}

func compareCommands(a, b *Command) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Type == b.Type && a.Payload == b.Payload
}
