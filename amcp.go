// Package amcp implements the Advanced Media Control Protocol as defined
// here: https://github.com/CasparCG/help/wiki/AMCP-Protocol.
package amcp

import (
	"bufio"
	"bytes"
	"errors"
	"net"
	"net/textproto"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Return codes
const (
	ReturnInfo     = 100
	ReturnInfoLine = 101

	ReturnOkMulti = 200
	ReturnOkData  = 201
	ReturnOk      = 202

	ReturnClientError         = 400
	ReturnIllegalVideoChannel = 401
	ReturnParameterMissing    = 402
	ReturnIllegalParameter    = 403
	ReturnMediaNotFound       = 404

	ReturnServerError        = 500
	ReturnServerErrorCommand = 501
	ReturnMediaUnreachable   = 502
	ReturnAccessError        = 503
)

// A Client represents a client connection to an AMCP server.
type Client struct {
	addr string
	text *textproto.Conn
	// underlying connection
	conn net.Conn

	Timeout time.Duration
}

// Dial returns a new Client connected to an AMCP server at addr.
func Dial(addr string) (*Client, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	host, _, _ := net.SplitHostPort(addr)
	return newClient(conn, host)
}

func newClient(conn net.Conn, addr string) (*Client, error) {
	text := textproto.NewConn(conn)
	return &Client{text: text, conn: conn, addr: addr}, nil
}

// Close closes the network connection.
func (c *Client) Close() error {
	return c.text.Close()
}

// Do sends a command to the server and returns the reply.
// If the server returned multiple lines of data, data is a []string, otherwise it's a string.
func (c *Client) Do(cmd string, args ...interface{}) (code int, data interface{}, err error) {
	id, err := c.send(cmd, args...)
	if err != nil {
		return 0, "", err
	}
	c.text.StartResponse(id)
	defer c.text.EndResponse(id)
	return c.receive()
}

// sends a command request to the server.
func (c *Client) send(cmd string, args ...interface{}) (id uint, err error) {
	id = c.text.Next()
	c.text.StartRequest(id)

	var deadline time.Time
	if c.Timeout > 0 {
		deadline = time.Now().Add(c.Timeout)
	}
	c.conn.SetWriteDeadline(deadline)

	_, err = c.text.W.WriteString(formatCmd(cmd, args...))
	if err != nil {
		return 0, err
	}
	err = c.text.W.Flush()
	if err != nil {
		return 0, err
	}
	c.text.EndRequest(id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

// preallocated new lines
var crnl = []byte{'\r', '\n'}

// formats the command and its arguments to be sent.
func formatCmd(cmd string, args ...interface{}) string {
	var b strings.Builder
	b.WriteString(cmd)
	for _, arg := range args {
		b.WriteByte(' ')
		switch v := arg.(type) {
		case int:
			b.WriteString(strconv.FormatInt(int64(v), 10))
		case float32:
			b.WriteString(strconv.FormatFloat(float64(v), 'f', -1, 32))
		case float64:
			b.WriteString(strconv.FormatFloat(v, 'f', -1, 64))
		case string:
			// wrap string in quotes if it has spaces
			quote := strings.IndexFunc(v, unicode.IsSpace) != -1
			if quote {
				b.WriteByte('"')
			}
			for _, r := range v {
				// escape special marks
				switch r {
				case '"':
					b.WriteString(`\"`)
				case '\\':
					b.WriteString(`\\`)
				case '\n':
					b.WriteString(`\n`)
				default:
					b.WriteRune(r)
				}
			}
			if quote {
				b.WriteByte('"')
			}
		}
	}
	b.Write(crnl)
	return b.String()
}

// reads a response from the server. Parses the return code and data.
// data will be a []string if mutli-line data, otherwise string.
func (c *Client) receive() (code int, data interface{}, err error) {
	var deadline time.Time
	if c.Timeout > 0 {
		deadline = time.Now().Add(c.Timeout)
	}
	c.conn.SetReadDeadline(deadline)
	r := bufio.NewReader(c.conn)

	b, err := r.ReadSlice('\n')
	if err != nil {
		return 0, "", err
	}
	line := string(b[:len(b)-2])
	code, data, err = parseCodeLine(line)
	if err != nil {
		return 0, "", err
	}

	// read all lines if multi line response
	if code == ReturnOkMulti || code == ReturnOkData {
		v := []string{data.(string)}

		for {
			b, err := r.ReadSlice('\n')
			if err != nil {
				return 0, "", err
			}

			endsWithCRNL := len(b) > 0 && bytes.Compare(b[len(b)-2:], crnl) == 0
			if endsWithCRNL {
				line = string(b[:len(b)-2])
			} else if b[len(b)-1] == '\n' {
				line = string(b[:len(b)-1])
			}
			v = append(v, line)

			// Single line data ends with CRNL on the end
			if code == ReturnOkData && endsWithCRNL {
				break
			}

			// Multi line data ends has an extra line with only CRNL to signal end
			if code == ReturnOkMulti && len(b) == 2 && endsWithCRNL {
				break
			}

		}
		data = v
	}

	return code, data, nil
}

// parse a line from the server including its return code and rest of the data.
func parseCodeLine(line string) (code int, msg string, err error) {
	if len(line) < 4 || line[3] != ' ' {
		err = errors.New("short response: " + line)
		return
	}
	code, err = strconv.Atoi(line[0:3])
	if err != nil || code < 100 {
		err = errors.New("invalid response: " + line)
		return
	}
	msg = line[4:]
	return
}
