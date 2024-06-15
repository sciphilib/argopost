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
	ParseHeloArgs(command *Command) string
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

	switch parts[0] {
	case "MAIL", "RCPT":
		if len(parts) > 1 && (parts[1] == "FROM:" || parts[1] == "TO:") {
			cmdString := strings.Trim(parts[0]+" "+parts[1], ":")
			cmdType = CommandType(strings.ToUpper(cmdString))
			payload = strings.Join(parts[2:], " ")
		}
	default:
		cmdType = CommandType(strings.ToUpper(parts[0]))
		payload = strings.Join(parts[1:], " ")
	}

	return &Command{
		Type:    cmdType,
		Payload: payload,
	}
}

func (p *GeneralCommandParser) ParseHeloArgs(command *Command) string {
	return strings.Fields(command.Payload)[0]
}
