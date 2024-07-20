package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	fmt.Println("Logs from your program will appear here!")

	var dir string
	if len(os.Args) > 2 && os.Args[1] == "--directory" {
		dir = os.Args[2]
		if !strings.HasSuffix(dir, "/") {
			dir += "/"
		}
	}

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	for {
		con, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		go handleConnection(con, dir)
	}
}

func handleConnection(con net.Conn, dir string) {
	defer con.Close()
	con.SetDeadline(time.Now().Add(5 * time.Second))

	reader := bufio.NewReader(con)
	request, err := http.ReadRequest(reader)
	if err != nil {
		fmt.Println("Error reading request: ", err.Error())
		return
	}

	header := Header{
		StatusCode:    404,
		StatusMessage: "Not Found",
	}

	var body io.Reader
	switch {
	case request.Method == "GET" && request.URL.Path == "/":
		header.StatusCode = 200
		header.StatusMessage = "OK"

	case request.Method == "GET" && strings.HasPrefix(request.URL.Path, "/echo/"):
		header.StatusCode = 200
		header.StatusMessage = "OK"
		header.ContentLength = len(request.URL.Path) - 6
		body = strings.NewReader(request.URL.Path[6:])

	case request.Method == "GET" && request.URL.Path == "/user-agent":
		header.StatusCode = 200
		header.StatusMessage = "OK"
		header.ContentLength = len(request.UserAgent())
		body = strings.NewReader(request.UserAgent())

	case request.Method == "GET" && strings.HasPrefix(request.URL.Path, "/files/"):
		file, err := os.Open(dir + request.URL.Path[7:])
		if err != nil {
			break
		}

		fi, err := file.Stat()
		if err != nil {
			file.Close()
			file = nil
			break
		}
		size := fi.Size()
		header.StatusCode = 200
		header.StatusMessage = "OK"
		header.ContentType = "application/octet-stream"
		header.ContentLength = int(size)
		body = file

	case request.Method == "POST" && strings.HasPrefix(request.URL.Path, "/files/"):
		file, err := os.Create(dir + request.URL.Path[7:])
		if err != nil {
			break
		}
		defer file.Close()

		io := bufio.NewReader(request.Body)
		io.WriteTo(file)

		header.StatusCode = 201
		header.StatusMessage = "Created"
	}

	for _, enc := range strings.Split(request.Header.Get("Accept-Encoding"), ",") {
		if strings.HasPrefix(strings.TrimLeft(enc, " "), "gzip") {
			header.ContentEncoding = "gzip"
			var buf bytes.Buffer
			gzipper := gzip.NewWriter(&buf)
			_, _ = io.Copy(gzipper, body)
			gzipper.Close()
			if closer, ok := body.(io.Closer); ok {
				closer.Close()
			}
			body = &buf
			header.ContentLength = buf.Len()
			break
		}
	}

	_, _ = con.Write([]byte(fmt.Sprintf("HTTP/1.1 %d %s\r\n", header.StatusCode, header.StatusMessage)))
	if header.ContentEncoding != "" {
		_, _ = con.Write([]byte(fmt.Sprintf("Content-Encoding: %s\r\n", header.ContentEncoding)))
	}
	if header.ContentType != "" {
		_, _ = con.Write([]byte(fmt.Sprintf("Content-Type: %s\r\n", header.ContentType)))
	} else if header.ContentLength > 0 {
		_, _ = con.Write([]byte("Content-Type: text/plain\r\n"))
	}
	if header.ContentLength > 0 {
		_, _ = con.Write([]byte(fmt.Sprintf("Content-Length: %d\r\n", header.ContentLength)))
	}
	con.Write([]byte("\r\n"))
	if body != nil {
		reader := bufio.NewReader(body)
		_, _ = reader.WriteTo(con)
		if closer, ok := body.(io.Closer); ok {
			closer.Close()
		}
		return
	}
}

type Header struct {
	StatusCode      int
	StatusMessage   string
	ContentLength   int
	ContentType     string
	ContentEncoding string
}
