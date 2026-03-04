package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

func ServeStatic(conn net.Conn, path string) {
	data, err := os.ReadFile("." + path)
	if err != nil {
		conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
		return
	}

	contentType := "text/plain"
	if strings.HasSuffix(path, ".css") {
		contentType = "text/css"
	} else if strings.HasSuffix(path, ".js") {
		contentType = "application/javascript"
	} else if strings.HasSuffix(path, ".svg") {
		contentType = "image/svg+xml"
	}

	response := fmt.Sprintf(
		"HTTP/1.1 200 OK\r\nContent-Type: %s\r\nContent-Length: %d\r\n\r\n",
		contentType, len(data),
	)
	conn.Write([]byte(response))
	conn.Write(data)
}

func HandleClient(conn net.Conn, db *sql.DB) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	requestLine, _ := reader.ReadString('\n')
	parts := strings.Fields(requestLine)
	if len(parts) < 2 {
		return
	}

	method := parts[0]
	path := parts[1]

	var contentLength int
	for {
		line, _ := reader.ReadString('\n')
		if strings.HasPrefix(line, "Content-Length:") {
			lengthStr := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
			contentLength, _ = strconv.Atoi(lengthStr)
		}
		if line == "\r\n" {
			break
		}
	}

	// Archivos estáticos
	if strings.HasPrefix(path, "/static/") {
		ServeStatic(conn, path)
		return
	}

	// DELETE serie
	if strings.HasPrefix(path, "/delete") && method == "DELETE" {
		p := strings.SplitN(path, "?", 2)
		params, _ := url.ParseQuery(p[1])
		id := params.Get("id")

		db.Exec("DELETE FROM ratings WHERE series_id = ?", id)
		db.Exec("DELETE FROM series WHERE id = ?", id)

		conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\nok"))
		return
	}

	// UPDATE episodio +1 / -1
	if strings.HasPrefix(path, "/update") && method == "POST" {
		p := strings.SplitN(path, "?", 2)
		params, _ := url.ParseQuery(p[1])
		id := params.Get("id")
		action := params.Get("action")

		if action == "minus" {
			db.Exec(
				"UPDATE series SET current_episode = current_episode - 1 WHERE id = ? AND current_episode > 0",
				id,
			)
		} else {
			db.Exec(
				"UPDATE series SET current_episode = current_episode + 1 WHERE id = ? AND current_episode < total_episodes",
				id,
			)
		}

		conn.Write([]byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\nok"))
		return
	}

	// RATE
	if strings.HasPrefix(path, "/rate") && method == "POST" {
		p := strings.SplitN(path, "?", 2)
		params, _ := url.ParseQuery(p[1])
		id := params.Get("id")
		value := params.Get("value")

		db.Exec("DELETE FROM ratings WHERE series_id = ?", id)
		db.Exec("INSERT INTO ratings (series_id, rating) VALUES (?, ?)", id, value)

		conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\nok"))
		return
	}

	// CREATE GET
	if path == "/create" && method == "GET" {
		html := `<html>
<head>
<title>Add Series</title>
<link rel="stylesheet" href="/static/style.css">
<link rel="icon" type="image/svg+xml" href="/static/favicon.svg">
</head>
<body>
<h1>Add New Series</h1>
<form method="POST" action="/create">
<input type="text" name="series_name" placeholder="Series Name" required><br>
<input type="number" name="current_episode" min="0" value="1" required><br>
<input type="number" name="total_episodes" min="1" required><br><br>
<button type="submit">Add Series</button>
</form>
<br>
<a href="/">← Back</a>
</body>
</html>`
		response := fmt.Sprintf(
			"HTTP/1.1 200 OK\r\nContent-Type: text/html; charset=utf-8\r\nContent-Length: %d\r\n\r\n%s",
			len(html), html,
		)
		conn.Write([]byte(response))
		return
	}

	// CREATE POST
	if path == "/create" && method == "POST" {
		body := make([]byte, contentLength)
		reader.Read(body)

		values, _ := url.ParseQuery(string(body))
		name := values.Get("series_name")
		current := values.Get("current_episode")
		total := values.Get("total_episodes")

		db.Exec(
			"INSERT INTO series (name, current_episode, total_episodes) VALUES (?, ?, ?)",
			name, current, total,
		)

		conn.Write([]byte("HTTP/1.1 303 See Other\r\nLocation: /\r\n\r\n"))
		return
	}

	// MAIN PAGE
	if path == "/" {
		rows, err := db.Query(`
			SELECT s.id, s.name, s.current_episode, s.total_episodes,
			       IFNULL(r.rating, 0)
			FROM series s
			LEFT JOIN ratings r ON s.id = r.series_id
		`)
		if err != nil {
			conn.Write([]byte("HTTP/1.1 500 Internal Server Error\r\n\r\n"))
			return
		}
		defer rows.Close()

		html := `<html>
<head>
	<title>Series Tracker</title>
	<link rel="stylesheet" href="/static/style.css">
	<link rel="icon" type="image/svg+xml" href="/static/favicon.svg">
</head>
<body>
<h1>My Series Tracker</h1>
<a href="/create">+ Add Series</a>
<br><br>
<table>
<tr>
	<th>ID</th>
	<th>Name</th>
	<th>Episodes</th>
	<th>Total</th>
	<th>Progress</th>
	<th>Rating</th>
	<th>Actions</th>
</tr>
`
		for rows.Next() {
			var id, current, total, rating int
			var name string
			rows.Scan(&id, &name, &current, &total, &rating)

			badge := ""
			if current == total {
				badge = ` <span class="badge">Completed</span>`
			}

			stars := ""
			for i := 1; i <= 10; i++ {
				if i <= rating {
					stars += fmt.Sprintf(`<span onclick="rate(%d,%d)">&#11088;</span>`, id, i)
				} else {
					stars += fmt.Sprintf(`<span onclick="rate(%d,%d)">&#9734;</span>`, id, i)
				}
			}

			html += fmt.Sprintf(`
<tr>
	<td>%d</td>
	<td>%s%s</td>
	<td>%d</td>
	<td>%d</td>
	<td class="progress" data-current="%d" data-total="%d"></td>
	<td>%s</td>
	<td>
		<button onclick="updateEp(%d,'minus')">-1</button>
		<button onclick="updateEp(%d,'plus')">+1</button>
		<button class="btn-delete" onclick="deleteSeries(%d)">&#10005;</button>
	</td>
</tr>`, id, name, badge, current, total, current, total, stars, id, id, id)
		}

		html += `
</table>
<script src="/static/script.js"></script>
</body>
</html>`

		response := fmt.Sprintf(
			"HTTP/1.1 200 OK\r\nContent-Type: text/html; charset=utf-8\r\nContent-Length: %d\r\n\r\n%s",
			len(html), html,
		)
		conn.Write([]byte(response))
		return
	}

	conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\nnot found"))
}