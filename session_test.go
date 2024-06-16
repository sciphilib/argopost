package smtp

import (
	"bufio"
	"bytes"
	"net"
	"strings"
	"testing"
	"time"
)

func TestSession_HandleHelo(t *testing.T) {
	conn := mockNetConn{}
	session := &Session{
		conn:   &conn,
		reader: bufio.NewReader(strings.NewReader("HELO email@mail.com some text\n")),
	}

	manager := NewManager()
	err := manager.HandleSession(session)
	if err != nil {
		t.Errorf("HandleSession returned an error: %v", err)
	}

	expectedResponse := "250 Hello email@mail.com\r\n"
	if conn.data.String() != expectedResponse {
		t.Errorf("Expected response %q, got %q", expectedResponse, conn.data.String())
	}
}

func TestSession_HandleEmptyHelo(t *testing.T) {
	conn := mockNetConn{}
	session := &Session{
		conn:   &conn,
		reader: bufio.NewReader(strings.NewReader("HELO\n")),
	}

	manager := NewManager()
	err := manager.HandleSession(session)
	if err != nil {
		t.Errorf("HandleSession returned an error: %v", err)
	}

	expectedResponse := "501 Domain/address argument is required for HELO\r\n"
	if conn.data.String() != expectedResponse {
		t.Errorf("Expected response %q, got %q", expectedResponse, conn.data.String())
	}
}

func TestSession_HandleErrorHelo(t *testing.T) {
	conn := mockNetConn{}
	session := &Session{
		conn:   &conn,
		reader: bufio.NewReader(strings.NewReader("HELLO test\n")),
	}

	manager := NewManager()
	err := manager.HandleSession(session)
	if err != nil {
		t.Errorf("HandleSession returned an error: %v", err)
	}

	expectedResponse := "503 Command HELLO test is invalid\r\n"
	if conn.data.String() != expectedResponse {
		t.Errorf("Expected response %q, got %q", expectedResponse, conn.data.String())
	}
}

func TestSession_Write(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		text    []string
		want    string
		wantErr bool
	}{
		{
			name:    "Single line message",
			code:    "250",
			text:    []string{"OK"},
			want:    "250 OK\r\n",
			wantErr: false,
		},
		{
			name:    "Multi-line message",
			code:    "250",
			text:    []string{"First line", "Second line", "Last line"},
			want:    "250-First line\r\n250-Second line\r\n250 Last line\r\n",
			wantErr: false,
		},
		{
			name:    "Empty message",
			code:    "250",
			text:    []string{},
			want:    "250 \r\n",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := &mockNetConn{data: bytes.Buffer{}}
			s := &Session{conn: mc}

			if err := s.Write(tt.code, tt.text...); (err != nil) != tt.wantErr {
				t.Errorf("Session.Write() error = %v, wantErr %v", err, tt.wantErr)
			}

			if got := mc.data.String(); got != tt.want {
				t.Errorf("Session.Write() got = %s, want %s", got, tt.want)
			}
		})
	}
}

type mockNetConn struct {
	net.Conn
	data bytes.Buffer
}

func (m *mockNetConn) SetDeadline(t time.Time) error {
	return nil
}

func (m *mockNetConn) Write(b []byte) (int, error) {
	return m.data.Write(b)
}

func (m *mockNetConn) Close() error {
	return nil
}
