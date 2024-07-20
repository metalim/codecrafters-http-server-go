package main

import (
	"bufio"
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

	header := "HTTP/1.1 404 Not Found\r\n"
	var body io.Reader
	switch {
	case request.Method == "GET" && request.URL.Path == "/":
		header = "HTTP/1.1 200 OK\r\n"

	case request.Method == "GET" && strings.HasPrefix(request.URL.Path, "/echo/"):
		header = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n", len(request.URL.Path)-6)
		body = strings.NewReader(request.URL.Path[6:])

	case request.Method == "GET" && request.URL.Path == "/user-agent":
		header = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n", len(request.UserAgent()))
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
		header = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n", size)
		body = file

	case request.Method == "POST" && strings.HasPrefix(request.URL.Path, "/files/"):
		file, err := os.Create(dir + request.URL.Path[7:])
		if err != nil {
			break
		}
		defer file.Close()

		io := bufio.NewReader(request.Body)
		io.WriteTo(file)
		header = "HTTP/1.1 201 Created\r\n"
	}

	for _, enc := range strings.Split(request.Header.Get("Accept-Encoding"), ",") {
		if strings.HasPrefix(strings.TrimLeft(enc, " "), "gzip") {
			header += "Content-Encoding: gzip\r\n"
			break
		}
	}

	_, _ = con.Write([]byte(header))
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
