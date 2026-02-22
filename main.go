package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"net"
	"strings"

	_"github.com/mattn/go-sqlite3"
)

func handleClient(conn net.Conn, db *sql.DB) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// Leer primera línea del request
	requestLine, _ := reader.ReadString('\n')
	parts := strings.Fields(requestLine)

	path := "/"
	if len(parts) > 1 {
		path = parts[1]
	}

	// Leer headers (ignorarlos)
	for {
		line, _ := reader.ReadString('\n')
		if line == "\r\n" {
			break
		}
	}

	var html string

	if path == "/" {
		rows, err := db.Query("SELECT id, name, current_episode, total_episodes FROM series")
		if err != nil {
			html = "<h1>Error querying database</h1>"
		} else {
			defer rows.Close()

			html = `
			<html>
			<head>
				<title>My Series Tracker</title>
				<style>
					body { font-family: Arial; background: #111; color: white; }
					table { border-collapse: collapse; width: 60%; margin: auto; }
					th, td { border: 1px solid white; padding: 10px; text-align: center; }
					th { background: #333; }
					h1 { text-align: center; }
				</style>
			</head>
			<body>
			<h1>My Series Tracker</h1>
			<table>
			<tr>
				<th>#</th>
				<th>Name</th>
				<th>Current</th>
				<th>Total</th>
			</tr>
			`

			for rows.Next() {
				var id int
				var name string
				var current int
				var total int

				rows.Scan(&id, &name, &current, &total)

				html += fmt.Sprintf(`
				<tr>
					<td>%d</td>
					<td>%s</td>
					<td>%d</td>
					<td>%d</td>
				</tr>
				`, id, name, current, total)
			}

			html += `
			</table>
			</body>
			</html>
			`
		}
	} else {
		html = fmt.Sprintf("<h1>Hello! You requested: %s</h1>", path)
	}

	// Construir respuesta HTTP
	response := fmt.Sprintf(
		"HTTP/1.1 200 OK\r\nContent-Type: text/html\r\nContent-Length: %d\r\n\r\n%s",
		len(html),
		html,
	)

	conn.Write([]byte(response))
}

func main() {
	// Abrir DB
	db, err := sql.Open("sqlite3", "series.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Crear servidor TCP
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	fmt.Println("Server running on http://localhost:8080")

	for {
		conn, _ := listener.Accept()
		go handleClient(conn, db)
	}
}