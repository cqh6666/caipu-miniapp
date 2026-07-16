package aialert

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestSMTPSenderClosesBlockedConnectionOnContextCancellation(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	accepted := make(chan net.Conn, 1)
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			accepted <- conn
		}
	}()

	port := listener.Addr().(*net.TCPAddr).Port
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		result <- NewSMTPSender().Send(ctx, SendRequest{
			Config: Config{
				SMTPHost:     "127.0.0.1",
				SMTPPort:     port,
				SMTPUsername: "bot@example.com",
				SMTPPassword: "secret",
				FromEmail:    "bot@example.com",
				ToEmails:     "ops@example.com",
			},
			Subject: "test",
			Body:    "test",
		})
	}()

	var serverConn net.Conn
	select {
	case serverConn = <-accepted:
		defer serverConn.Close()
	case <-time.After(time.Second):
		t.Fatal("SMTP sender did not connect")
	}
	cancel()
	select {
	case err := <-result:
		if err == nil {
			t.Fatal("Send() error = nil after cancellation")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Send() did not return after context cancellation")
	}
}
