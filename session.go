package smtp

import (
	"bufio"
	"fmt"
	"hash/maphash"
	"io"
	"net"
	"net/mail"
	"strings"
	"time"

	"github.com/sciphilib/argopost/internal"
)

type (
	Session struct {
		hash   uint64
		conn   net.Conn
		state  SessionState
		reader *bufio.Reader
		user   string
		from   string
		to     string
		data   string
	}

	SessionState int

	SessionManager struct {
		sessions  map[uint64]*Session
		cmdParser internal.CommandParser
	}
)

var seed maphash.Seed = maphash.MakeSeed()

func NewManager() *SessionManager {
	return &SessionManager{
		cmdParser: &internal.GeneralCommandParser{},
		sessions:  make(map[uint64]*Session),
	}
}

func (m *SessionManager) PrintMap() {
	for key, session := range m.sessions {
		fmt.Printf("Key: %d, Session: %+v\n", key, session)
	}
}

func (m *SessionManager) HandleSession(s *Session) error {
	for {
		message, err := s.reader.ReadString('\n')
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		command := m.cmdParser.Parse(message)
		if command == nil {
			continue
		}

		switch command.Type {
		case internal.HeloCommand:
			s.handleHelo(command.Payload)
		case internal.MailFromCommand:
			s.handleMailFrom(command.Payload)
		case internal.RcptToCommand:
			s.handleRcptTo(command.Payload)
		case internal.DataCommand:
			s.handleData(command.Payload)
		case internal.QuitCommand:
			s.Write("221", "Goodnight and good luck")
			return nil
		default:
			s.invalidCommand(message)
		}
	}

	return nil
}

func (m *SessionManager) CreateSession(conn net.Conn) *Session {
	var h maphash.Hash
	h.SetSeed(seed)
	h.WriteString(conn.LocalAddr().String())
	h.WriteString(conn.RemoteAddr().String())
	hash := h.Sum64()

	s := &Session{
		hash:   hash,
		conn:   conn,
		reader: bufio.NewReader(conn),
	}
	m.sessions[s.hash] = s
	return s
}

func (m *SessionManager) CloseSession(s *Session) {
	delete(m.sessions, s.hash)
	s.conn.Close()
}

func (s *Session) Write(code string, text ...string) error {
	s.conn.SetDeadline(s.nextDeadline())

	var builder strings.Builder
	for i, line := range text {
		if i < len(text)-1 {
			builder.WriteString(fmt.Sprintf("%s-%s\r\n", code, line))
		} else {
			builder.WriteString(fmt.Sprintf("%s %s\r\n", code, line))
		}
	}

	if len(text) == 0 {
		builder.WriteString(fmt.Sprintf("%s \r\n", code))
	}

	_, err := s.conn.Write([]byte(builder.String()))
	if err != nil {
		return fmt.Errorf("failed to write to connection: %w", err)
	}
	return nil
}

func (s *Session) nextDeadline() time.Time {
	// todo: remove hardcode and start using config
	seconds := 100
	return time.Now().Add(time.Duration(seconds) * time.Second)
}

func (s *Session) handleHelo(args string) error {
	if len(args) == 0 {
		return s.Write("501", "Domain/address argument is required for HELO")
	}
	parts := strings.Trim(strings.Fields(args)[0], "<>\":")
	var builder strings.Builder
	s.user = parts

	builder.WriteString(fmt.Sprintf("Hello %s", s.user))
	return s.Write("250", builder.String())
}

func (s *Session) handleMailFrom(args string) error {
	// todo: check authorization
	if len(s.user) == 0 {
		return s.Write("502", "Enter domain/address first before MAIL FROM command")
	}

	parts := strings.Fields(args)
	if len(parts) == 0 {
		return s.Write("501", "Email argument is required for MAIL FROM")
	}

	email, err := mail.ParseAddress(strings.Trim(parts[0], "<>\":"))
	if err != nil {
		return s.Write("501", "Invalid email address")
	}

	s.from = email.Address

	return s.Write("250", fmt.Sprintf("Accepting mail from %s", s.from))
}

func (s *Session) handleRcptTo(args string) error {
	parts := strings.Fields(args)
	if len(args) == 0 {
		return s.Write("501", "Email argument is required for RCPT TO")
	}

	if len(s.from) == 0 {
		return s.Write("502", "Missign MAIL FROM command")
	}

	email, err := mail.ParseAddress(strings.Trim(parts[0], "<>\":"))
	if err != nil {
		return s.Write("501", "Invalid email address")
	}

	s.to = email.Address

	return s.Write("250", fmt.Sprintf("Will deliever mail to %s", s.to))
}

func (s *Session) handleData(args string) error {
	if len(args) != 0 {
		return s.Write("501", "There are should be no arguments for DATA command")
	}

	if len(s.to) == 0 || len(s.from) == 0 {
		return s.Write("502", "Missing RCPT TO command")
	}

	s.Write("354", "Enter data with a terminating .")

	s.processData()

	return nil
}

func (s *Session) processData() error {
	var (
		builder strings.Builder
		message string
		err     error
	)

	for {
		message, err = s.reader.ReadString('\n')
		if strings.HasSuffix(message, ".\n") {
			builder.WriteString(strings.TrimSuffix(message, "."))
			break
		}
		builder.WriteString(message)
	}

	s.data = builder.String()

	// todo: realize backend to send mail with data to another server
	if err == nil {
		s.Write("250", "Ok: queued")
	} else {
		s.Write("554", "Error: transaction failed")
	}

	s.reset()

	return err
}

func (s *Session) reset() {
	s.user = ""
	s.from = ""
	s.to = ""
	s.data = ""
}

func (s *Session) invalidCommand(m string) {
	s.Write("503", fmt.Sprintf("Command %s is invalid", strings.Join(strings.Fields(m), " ")))
}

func (s *Session) unexpectedCommand(c *internal.Command) {
	s.Write("503", fmt.Sprintf("Command %s is unexpected", c.Type))
}
