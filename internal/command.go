package internal

import "strings"

type CommandType string

const (
	HeloCommand     CommandType = "HELO"
	MailFromCommand             = "MAIL FROM"
	RcptToCommand               = "RCPT TO"
	DataCommand                 = "DATA"
	QuitCommand                 = "QUIT"
)

type Command struct {
	Type    CommandType
	Payload string
}

type CommandParser interface {
	Parse(command string) *Command
}

type GeneralCommandParser struct{}

func (p *GeneralCommandParser) Parse(command string) *Command {
	var (
		cmdType CommandType
		payload string
	)

	parts := strings.Fields(command)
	if len(parts) == 0 {
		return nil
	}

	if len(parts) > 2 {
		cmdType = CommandType(strings.ToUpper(parts[0]) + " " + strings.Trim(strings.ToUpper(parts[1]), ":"))
		payload = strings.Join(parts[2:], " ")
	} else {
		cmdType = CommandType(strings.ToUpper(parts[0]))
		payload = strings.Join(parts[1:], " ")
	}

	return &Command{Type: cmdType, Payload: payload}
}
