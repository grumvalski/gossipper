package sip

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func ReadMessage(reader *bufio.Reader) (Message, error) {
	startLine, err := reader.ReadString('\n')
	if err != nil {
		if errors.Is(err, io.EOF) && strings.TrimSpace(startLine) == "" {
			return Message{}, fmt.Errorf("peer closed connection before any SIP data: %w", err)
		}
		if errors.Is(err, io.EOF) {
			return Message{}, fmt.Errorf("connection closed while reading SIP start line: %w", err)
		}
		return Message{}, err
	}
	startLine = strings.TrimRight(startLine, "\r\n")
	if strings.TrimSpace(startLine) == "" {
		return Message{}, fmt.Errorf("empty SIP start line")
	}

	var builder strings.Builder
	builder.WriteString(startLine)
	builder.WriteString("\r\n")

	headers := make(map[string][]string)
	contentLength := 0
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				return Message{}, fmt.Errorf("connection closed while reading SIP headers: %w", err)
			}
			return Message{}, err
		}
		line = strings.TrimRight(line, "\r\n")
		builder.WriteString(line)
		builder.WriteString("\r\n")
		if line == "" {
			break
		}

		name, value, ok := strings.Cut(line, ":")
		if !ok {
			return Message{}, fmt.Errorf("malformed SIP header %q", line)
		}
		name = strings.TrimSpace(name)
		value = strings.TrimSpace(value)
		headers[name] = append(headers[name], value)
		if strings.EqualFold(name, "Content-Length") {
			contentLength, _ = strconv.Atoi(value)
		}
	}

	body := make([]byte, contentLength)
	if contentLength > 0 {
		if _, err := io.ReadFull(reader, body); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return Message{}, fmt.Errorf("connection closed while reading SIP body (%d bytes): %w", contentLength, err)
			}
			return Message{}, err
		}
		builder.Write(body)
	}

	return Parse([]byte(builder.String()))
}
