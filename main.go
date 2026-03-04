package main

import (
	"fmt"
	"net"
)

func main() {
	db := InitDB()
	defer db.Close()

	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	fmt.Println("Server running on http://localhost:8080")

	for {
		conn, _ := listener.Accept()
		go HandleClient(conn, db)
	}
}