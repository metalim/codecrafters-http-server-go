package main

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	fmt.Println("Logs from your program will appear here!")

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
		go handleConnection(con)
	}
}

func handleConnection(con net.Conn) {
	defer con.Close()
	con.SetDeadline(time.Now().Add(5 * time.Second))

	reader := bufio.NewReader(con)
	request, err := http.ReadRequest(reader)
	if err != nil {
		fmt.Println("Error reading request: ", err.Error())
		return
	}

	var res string
	switch {
	case request.Method == "GET" && request.URL.Path == "/":
		res = "HTTP/1.1 200 OK\r\n\r\n"

	case request.Method == "GET" && strings.HasPrefix(request.URL.Path, "/echo/"):
		res = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(request.URL.Path)-6, request.URL.Path[6:])

	case request.Method == "GET" && request.URL.Path == "/user-agent":
		res = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(request.UserAgent()), request.UserAgent())

	default:
		res = "HTTP/1.1 404 Not Found\r\n\r\n"
	}
	con.Write([]byte(res))
}
