package lsp

import (
	"bufio"
	"bytes"
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
