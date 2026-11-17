// Command imapstub is a tiny in-process IMAP server used only for local
// end-to-end testing of the CRM inbox polling. It accepts the login
// "stub"/"stubpass" and serves two canned messages in INBOX.
package main

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend"
	"github.com/emersion/go-imap/backend/memory"
	"github.com/emersion/go-imap/server"
)

func message(uid uint32, raw string) *memory.Message {
	return &memory.Message{
		Uid:   uid,
		Date:  time.Now().UTC(),
		Flags: []string{},
		Size:  uint32(len(raw)),
		Body:  []byte(raw),
	}
}

func rawMessages() []string {
	now := time.Now().UTC().Format(time.RFC1123Z)
	return []string{
		"From: Alice Sender <alice@example.com>\r\n" +
			"To: me@company.com\r\n" +
			"Subject: Pricing question\r\n" +
			"Message-ID: <m1@example.com>\r\n" +
			"Date: " + now + "\r\n" +
			"Content-Type: text/plain; charset=UTF-8\r\n" +
			"\r\n" +
			"Hi, what does the pro plan cost?\r\n",
		"From: Bob Buyer <bob@example.com>\r\n" +
			"To: me@company.com\r\n" +
			"Subject: Re: Pricing question\r\n" +
			"Message-ID: <m2@example.com>\r\n" +
			"In-Reply-To: <m1@example.com>\r\n" +
			"References: <m1@example.com>\r\n" +
			"Date: " + now + "\r\n" +
			"Content-Type: multipart/alternative; boundary=\"bb\"\r\n" +
			"\r\n" +
			"--bb\r\n" +
			"Content-Type: text/plain; charset=UTF-8\r\n" +
			"\r\n" +
			"Thanks!\r\n" +
			"--bb\r\n" +
			"Content-Type: text/html; charset=UTF-8\r\n" +
			"\r\n" +
			"<p>Thanks!</p>\r\n" +
			"--bb--\r\n",
	}
}

// stubBackend wraps the memory backend and seeds INBOX on first login.
type stubBackend struct{ inner *memory.Backend }

func (b *stubBackend) Login(info *imap.ConnInfo, username, password string) (backend.User, error) {
	if username != "stub" || password != "stubpass" {
		return nil, fmt.Errorf("bad username or password")
	}
	u, err := b.inner.Login(info, "username", "password")
	if err != nil {
		return nil, err
	}
	mu := u.(*memory.User)
	mbox, err := mu.GetMailbox("INBOX")
	if err != nil {
		return nil, err
	}
	mm := mbox.(*memory.Mailbox)
	if len(mm.Messages) <= 1 { // memory.New() pre-seeds one message
		for i, raw := range rawMessages() {
			mm.Messages = append(mm.Messages, message(uint32(100+i), raw))
		}
		log.Printf("seeded INBOX with %d stub messages", len(mm.Messages))
	}
	return u, nil
}

func (b *stubBackend) AnonymousLogin(info *imap.ConnInfo) (backend.User, error) {
	return nil, fmt.Errorf("anonymous login disabled")
}

func main() {
	port := "1993"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}
	be := &stubBackend{inner: memory.New()}
	s := server.New(be)
	s.AllowInsecureAuth = true
	ln, err := net.Listen("tcp", "127.0.0.1:"+port)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("imapstub listening on 127.0.0.1:%s (login stub/stubpass)", port)
	_ = bytes.MinRead
	if err := s.Serve(ln); err != nil {
		log.Fatal(err)
	}
}
