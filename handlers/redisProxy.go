package handlers

import (
	"bufio"
	"context"
	"fmt"
	"github.com/labstack/echo/v4"
	"net"
	"redis-proxy/controllers"
	"strconv"
)

var (
	// ErrInvalidCommand is returned when an invalid command is received
	ErrInvalidCommand = "-Error %s\r\n"
)

type RedisProxy struct {
	controller controllers.RedisProxy
}

func NewRedisProxy(controller controllers.RedisProxy) RedisProxy {
	return RedisProxy{controller: controller}
}

// GetHTTP Handles incoming HTTP requests.
func (h RedisProxy) GetHTTP(c echo.Context) error {
	key := c.QueryParam("key")
	val, err := h.controller.Get(c.Request().Context(), key)
	if err != nil {
		return c.String(500, err.Error())
	}
	return c.JSON(200, val)
}

// GetRESP Handles incoming RESP requests.
// only supports GET command currently
// expects the following format: *2\r\n\$3\r\nGET\r\n\$3\r\nhello\r\n, where here 'hello' is the key
func (h RedisProxy) GetRESP(conn net.Conn) string {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// parse the request and validate it
	key, err := parseAndValidateRESPRequest(reader)
	if err != nil {
		conn.Write([]byte(fmt.Sprintf(ErrInvalidCommand, err.Error())))
		return fmt.Sprintf(ErrInvalidCommand, err.Error())
	}
	val, err := h.controller.Get(context.Background(), key)
	if err != nil {
		conn.Write([]byte(fmt.Sprintf(ErrInvalidCommand, err.Error())))
		return fmt.Sprintf(ErrInvalidCommand, err.Error())
	}
	// write the value to the client in the format $<length>\r\n<value>\r\n
	returnVal := fmt.Sprintf("$%d\r\n%s\r\n", len(val), val)
	conn.Write([]byte(returnVal))

	return returnVal
}

func parseAndValidateRESPRequest(reader *bufio.Reader) (string, error) {
	c, err := reader.ReadByte()
	if err != nil {
		return "", err
	}
	var vals []string
	// expecting an array ('*') of length 2 in the request
	if c == '*' {
		// next value is the length of the array
		arrayLength, err := readInt(reader)
		if err != nil {
			return "", err
		}
		if arrayLength != 2 {
			return "", fmt.Errorf("invalid command, expected 2 arguments")
		}

		// parse the array
		vals = make([]string, arrayLength)
		for i := 0; i < arrayLength; i++ {
			c, err = reader.ReadByte()
			if err != nil {
				return "", err
			}
			// expecting a bulk string ('$') as the next value
			if c == '$' {
				// parse the bulk string and store it
				strLength, err := readInt(reader)
				if err != nil {
					return "", err
				}
				b := make([]byte, strLength+2)
				_, err = reader.Read(b)
				if err != nil {
					return "", err
				}
				vals[i] = string(b[:strLength])
			} else {
				return "", fmt.Errorf("invalid command")
			}
		}
	} else {
		return "", fmt.Errorf("invalid command, expected an array")
	}

	// this error should be caught earlier if the client passes in an accurate array length
	if len(vals) != 2 {
		return "", fmt.Errorf("invalid command, expected 2 arguments")
	}

	if vals[0] != "GET" {
		return "", fmt.Errorf("invalid command, only GET is supported")
	}
	return vals[1], nil
}

func readInt(reader *bufio.Reader) (int, error) {
	line, _, err := reader.ReadLine()
	if err != nil {
		return 0, err
	}
	i64, err := strconv.ParseInt(string(line), 10, 64)
	if err != nil {
		return 0, err
	}
	return int(i64), nil
}
