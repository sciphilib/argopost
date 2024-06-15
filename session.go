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
	}

	SessionState int

	SessionManager struct {
		sessions  map[uint64]*Session
		cmdParser internal.CommandParser
	}
)

const (
	WaitingForCmd SessionState = iota
	ReadingData
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
		// todo handle
		s.Write("250", "OK")
	default:
		// todo handle
		s.Write("550", "Error occured")
	}

	return nil
}

func (m *SessionManager) CreateSession(conn net.Conn) *Session {
	s := &Session{
		hash:   maphash.String(seed, conn.LocalAddr().String()),
		conn:   conn,
		state:  WaitingForCmd,
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
	seconds := 10
	return time.Now().Add(time.Duration(seconds) * time.Second)
}
