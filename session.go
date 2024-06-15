package smtp

import (
	"bufio"
	"fmt"
	"hash/maphash"
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

func (m *SessionManager) HandleSession(s *Session) error {
	message, err := s.reader.ReadString('\n')
	if err != nil {
		return err
	}

	command := m.cmdParser.Parse(message)
	fmt.Println(command)

	switch command.Type {
	case internal.HeloCommand:
		s.handleHelo(command.Payload)
	case internal.MailFromCommand:
		s.handleMailFrom(command.Payload)
	default:
		s.invalidCommand(message)
	}

	return nil
}

func (m *SessionManager) CreateSession(conn net.Conn) *Session {
	s := &Session{
		hash:   maphash.String(seed, conn.LocalAddr().String()),
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
	seconds := 10
	return time.Now().Add(time.Duration(seconds) * time.Second)
}

func (s *Session) handleHelo(args string) {
	parts := strings.Fields(args)[0]
	var builder strings.Builder
	if len(parts) == 0 {
		s.Write("501", "Domain/address argument is required for HELO")
	}
	s.user = parts

	builder.WriteString(fmt.Sprintf("Hello %s", s.user))
	s.Write("250", builder.String())
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

	email, err := mail.ParseAddress(parts[0])
	if err != nil {
		return s.Write("501", "Invalid email address")
	}

	s.from = email.Address

	return s.Write("250", fmt.Sprintf("Accepting mail from %s", s.from))
}

func (s *Session) invalidCommand(m string) {
	s.Write("503", fmt.Sprintf("Command %s is invalid", strings.Join(strings.Fields(m), " ")))
}

func (s *Session) unexpectedCommand(c *internal.Command) {
	s.Write("503", fmt.Sprintf("Command %s is unexpected", c.Type))
}
