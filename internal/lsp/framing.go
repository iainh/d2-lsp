package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"strconv"
	"strings"
)

type unsupportedContentCharsetError struct {
	charset string
}

func (e unsupportedContentCharsetError) Error() string {
	return fmt.Sprintf("unsupported Content-Type charset %q", e.charset)
}

func readMessage(reader *bufio.Reader) ([]byte, error) {
	contentLength := -1
	contentType := ""

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
		} else if strings.EqualFold(strings.TrimSpace(name), "Content-Type") {
			contentType = strings.TrimSpace(value)
		}
	}

	if contentLength < 0 {
		return nil, fmt.Errorf("missing Content-Length")
	}

	if charset, ok, err := contentCharset(contentType); err != nil {
		return nil, err
	} else if ok && !isUTF8Charset(charset) {
		if _, err := io.CopyN(io.Discard, reader, int64(contentLength)); err != nil {
			return nil, err
		}
		return nil, unsupportedContentCharsetError{charset: charset}
	}

	body := make([]byte, contentLength)
	if _, err := io.ReadFull(reader, body); err != nil {
		return nil, err
	}
	return body, nil
}

func contentCharset(contentType string) (string, bool, error) {
	if contentType == "" {
		return "", false, nil
	}

	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return "", false, fmt.Errorf("invalid Content-Type %q: %w", contentType, err)
	}
	charset, ok := params["charset"]
	return charset, ok, nil
}

func isUTF8Charset(charset string) bool {
	normalized := strings.ToLower(strings.TrimSpace(charset))
	return normalized == "utf-8" || normalized == "utf8"
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

func encodeForTestWithHeader(value interface{}, header string) []byte {
	body, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}

	var buf bytes.Buffer
	if _, err := fmt.Fprintf(&buf, "Content-Length: %d\r\n%s\r\n", len(body), header); err != nil {
		panic(err)
	}
	if _, err := buf.Write(body); err != nil {
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
