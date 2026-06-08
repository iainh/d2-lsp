package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func readMessage(reader *bufio.Reader) ([]byte, error) {
	contentLength := -1

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}

		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}

		name, value, ok := strings.Cut(line, ":")
		if !ok {
			return nil, fmt.Errorf("invalid header line %q", line)
		}

		if strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
			contentLength, err = strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return nil, fmt.Errorf("invalid Content-Length %q: %w", value, err)
			}
		}
	}

	if contentLength < 0 {
		return nil, fmt.Errorf("missing Content-Length")
	}

	body := make([]byte, contentLength)
	if _, err := io.ReadFull(reader, body); err != nil {
		return nil, err
	}
	return body, nil
}

func writeJSON(writer io.Writer, value interface{}) error {
	body, err := json.Marshal(value)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(writer, "Content-Length: %d\r\n\r\n", len(body)); err != nil {
		return err
	}
	_, err = writer.Write(body)
	return err
}

func encodeForTest(value interface{}) []byte {
	var buf bytes.Buffer
	if err := writeJSON(&buf, value); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func encodeForTestRaw(body []byte) []byte {
	var buf bytes.Buffer
	if _, err := fmt.Fprintf(&buf, "Content-Length: %d\r\n\r\n", len(body)); err != nil {
		panic(err)
	}
	if _, err := buf.Write(body); err != nil {
		panic(err)
	}
	return buf.Bytes()
}
