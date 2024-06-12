package smtp

import (
	"testing"
)

func TestServer(t *testing.T) {
	server := New(&Config{
		Host: "localhost",
		Port: "6969",
	})
	server.Run()
}
