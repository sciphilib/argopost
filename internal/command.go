package internal

import (
	"strings"
)

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

	cmdBegin := strings.ToUpper(parts[0])

	switch cmdBegin {
	case "MAIL", "RCPT":
		cmdEnd := strings.ToUpper(parts[1])
		if len(parts) > 1 && (cmdEnd == "FROM" || cmdEnd == "TO") {
			cmdString := cmdBegin + " " + cmdEnd
			cmdType = CommandType(cmdString)
			payload = strings.Join(parts[2:], " ")
		}
	default:
		cmdType = CommandType(cmdBegin)
		payload = strings.Join(parts[1:], " ")
	}

	return &Command{
		Type:    cmdType,
		Payload: payload,
	}
}
