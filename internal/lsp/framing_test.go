package lsp

import (
	"bufio"
	"bytes"
	"errors"
	"testing"
)

func TestReadMessageReadsContentLengthFramedBody(t *testing.T) {
	body := []byte(`{"jsonrpc":"2.0","method":"test"}`)
	input := append([]byte("Content-Length: 33\r\n\r\n"), body...)

	got, err := readMessage(bufio.NewReader(bytes.NewReader(input)))
	if err != nil {
		t.Fatalf("read message: %v", err)
	}
	if !bytes.Equal(got, body) {
		t.Fatalf("got %q, want %q", got, body)
	}
}

func TestReadMessageAcceptsUTF8ContentType(t *testing.T) {
	body := []byte(`{"jsonrpc":"2.0","method":"test"}`)
	input := append([]byte("Content-Length: 33\r\nContent-Type: application/vscode-jsonrpc; charset=utf8\r\n\r\n"), body...)

	got, err := readMessage(bufio.NewReader(bytes.NewReader(input)))
	if err != nil {
		t.Fatalf("read message: %v", err)
	}
	if !bytes.Equal(got, body) {
		t.Fatalf("got %q, want %q", got, body)
	}
}

func TestReadMessageRejectsUnsupportedContentTypeCharset(t *testing.T) {
	body := []byte(`{"jsonrpc":"2.0","method":"test"}`)
	input := append([]byte("Content-Length: 33\r\nContent-Type: application/vscode-jsonrpc; charset=utf-16\r\n\r\n"), body...)
	reader := bufio.NewReader(bytes.NewReader(input))

	got, err := readMessage(reader)
	if got != nil {
		t.Fatalf("expected no message body, got %q", got)
	}
	var charsetErr unsupportedContentCharsetError
	if !errors.As(err, &charsetErr) {
		t.Fatalf("expected unsupported charset error, got %v", err)
	}
	if charsetErr.charset != "utf-16" {
		t.Fatalf("expected utf-16 charset, got %q", charsetErr.charset)
	}
	if reader.Buffered() != 0 {
		t.Fatalf("expected unsupported message body to be consumed, buffered %d bytes", reader.Buffered())
	}
}

func TestWriteJSONWritesContentLengthFrame(t *testing.T) {
	var buf bytes.Buffer
	if err := writeJSON(&buf, map[string]string{"jsonrpc": "2.0"}); err != nil {
		t.Fatalf("write json: %v", err)
	}

	want := "Content-Length: 17\r\n\r\n{\"jsonrpc\":\"2.0\"}"
	if buf.String() != want {
		t.Fatalf("got %q, want %q", buf.String(), want)
	}
}
