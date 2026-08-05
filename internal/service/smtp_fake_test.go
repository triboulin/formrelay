package service

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// fakeSMTPServer est un serveur SMTP minimal utilisé pour tester l'envoi
// réel via net/smtp sans dépendre d'un service externe. Il accepte EHLO,
// AUTH PLAIN, MAIL FROM, RCPT TO, DATA et QUIT, et enregistre le contenu
// des messages reçus.
type fakeSMTPServer struct {
	mu       sync.Mutex
	received []string
	addr     string
	ln       net.Listener
}

func startFakeSMTPServer(t *testing.T) *fakeSMTPServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("impossible de démarrer le faux serveur SMTP: %v", err)
	}
	s := &fakeSMTPServer{addr: ln.Addr().String(), ln: ln}
	go s.serve()
	t.Cleanup(func() { ln.Close() })
	return s
}

func (s *fakeSMTPServer) serve() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.handle(conn)
	}
}

func (s *fakeSMTPServer) handle(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	write := func(format string, args ...any) {
		fmt.Fprintf(writer, format+"\r\n", args...)
		writer.Flush()
	}
	write("220 localhost ESMTP fake")

	var data strings.Builder
	inData := false

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")

		if inData {
			if line == "." {
				inData = false
				s.mu.Lock()
				s.received = append(s.received, data.String())
				s.mu.Unlock()
				data.Reset()
				write("250 OK: message queued")
				continue
			}
			data.WriteString(line)
			data.WriteString("\n")
			continue
		}

		upper := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(upper, "EHLO"):
			write("250-localhost Hello")
			write("250 AUTH PLAIN")
		case strings.HasPrefix(upper, "HELO"):
			write("250 localhost Hello")
		case strings.HasPrefix(upper, "AUTH"):
			write("235 2.7.0 Authentication successful")
		case strings.HasPrefix(upper, "MAIL FROM"):
			write("250 OK")
		case strings.HasPrefix(upper, "RCPT TO"):
			write("250 OK")
		case upper == "DATA":
			inData = true
			write("354 Start mail input; end with <CRLF>.<CRLF>")
		case upper == "QUIT":
			write("221 Bye")
			return
		default:
			write("250 OK")
		}
	}
}

// hostPort retourne l'hôte et le port séparés du serveur, pour peupler config.Config.
func (s *fakeSMTPServer) hostPort(t *testing.T) (string, int) {
	t.Helper()
	host, portStr, err := net.SplitHostPort(s.addr)
	if err != nil {
		t.Fatalf("adresse du faux serveur invalide: %v", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("port invalide: %v", err)
	}
	return host, port
}

func (s *fakeSMTPServer) receivedCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.received)
}

func (s *fakeSMTPServer) lastMessage() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.received) == 0 {
		return ""
	}
	return s.received[len(s.received)-1]
}
