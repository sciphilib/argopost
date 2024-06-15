package smtp

import (
	"bufio"
	"fmt"
	"hash/maphash"
	"net"
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

	switch command.Type {
	case internal.HeloCommand:
		args := m.ParseHeloArgs(command)
		s.handleHelo(args)
	default:
		s.invalidCommand(command)
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

func (m *SessionManager) ParseHeloArgs(c *internal.Command) string {
	return m.cmdParser.ParseHeloArgs(c)
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
	var builder strings.Builder
	if len(args) == 0 {
		s.Write("501", "Domain/address argument is required for HELO")
	}
	s.user = args

	builder.WriteString(fmt.Sprintf("Hello %s", s.user))
	s.Write("250", builder.String())
}

func (s *Session) invalidCommand(c *internal.Command) {
	s.Write("503", fmt.Sprintf("Command %s is invalid or out of sequence", c.Type))
}
