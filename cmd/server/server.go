package main

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
)

func main() {
	fmt.Println("Logs from your program will appear here!")

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	con, err := l.Accept()
	if err != nil {
		fmt.Println("Error accepting connection: ", err.Error())
		os.Exit(1)
	}
	defer con.Close()

	reader := bufio.NewReader(con)
	request, err := http.ReadRequest(reader)
	if err != nil {
		fmt.Println("Error reading request: ", err.Error())
		os.Exit(1)
	}
	switch {
	case request.Method == "GET" && request.URL.Path == "/":
		con.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))

	case request.Method == "GET" && strings.HasPrefix(request.URL.Path, "/echo/"):
		con.Write([]byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(request.URL.Path)-6, request.URL.Path[6:])))

	default:
		con.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
	}
}
