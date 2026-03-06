package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

func main() {

	db, err := sql.Open("sqlite", "file:series.db")
	if err != nil {
		log.Fatal("DB connection error:", err)
	}
	defer db.Close()

	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal("Server listen error:", err)
	}
	defer listener.Close()

	log.Println("Listening on port 8080")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Connection error:", err)
			continue
		}
		go handleClient(conn, db)
	}
}

func handleClient(conn net.Conn, db *sql.DB) {

	defer conn.Close()
	reader := bufio.NewReader(conn)

	requestLine, _ := reader.ReadString('\n')
	parts := strings.Fields(requestLine)
	if len(parts) < 2 {
		return
	}

	method := parts[0]
	path := parts[1]

	pathParts := strings.SplitN(path, "?", 2)
	route := pathParts[0]

	var queryParams url.Values
	if len(pathParts) > 1 {
		queryParams, _ = url.ParseQuery(pathParts[1])
	}

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

	var body string
	if contentLength > 0 {
		bodyBytes := make([]byte, contentLength)
		io.ReadFull(reader, bodyBytes)
		body = string(bodyBytes)
	}

	if route == "/create" && method == "GET" {

		html := `
<html>
<body>
<h1>Agregar Serie</h1>
<form method="POST" action="/create">
Nombre: <input type="text" name="series_name" required><br><br>
Episodio Actual: <input type="number" name="current_episode" min="1" value="1" required><br><br>
Total Episodios: <input type="number" name="total_episodes" min="1" required><br><br>
<button type="submit">Enviar</button>
</form>
<a href="/">Volver</a>
</body>
</html>`

		conn.Write([]byte("HTTP/1.1 200 OK\r\nContent-Type: text/html\r\n\r\n" + html))
		return
	}

	if route == "/create" && method == "POST" {

		values, _ := url.ParseQuery(body)

		name := values.Get("series_name")
		currentStr := values.Get("current_episode")
		totalStr := values.Get("total_episodes")

		current, _ := strconv.Atoi(currentStr)
		total, _ := strconv.Atoi(totalStr)

		if name == "" || current < 1 || total < 1 || current > total {
			conn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\nDatos invalidos"))
			return
		}

		_, err := db.Exec(
			"INSERT INTO series (name, current_episode, total_episodes) VALUES (?, ?, ?)",
			name, current, total,
		)
		if err != nil {
			log.Println("DB INSERT error:", err)
			conn.Write([]byte("HTTP/1.1 500 Internal Server Error\r\n\r\nDatabase error"))
			return
		}

		conn.Write([]byte("HTTP/1.1 303 See Other\r\nLocation: /\r\n\r\n"))
		return
	}

	if route == "/update" && method == "POST" {

		id := queryParams.Get("id")

		_, err := db.Exec(`UPDATE series 
			SET current_episode = current_episode + 1
			WHERE id = ? AND current_episode < total_episodes`, id)

		if err != nil {
			log.Println("DB UPDATE error:", err)
			conn.Write([]byte("HTTP/1.1 500 Internal Server Error\r\n\r\nDatabase error"))
			return
		}

		conn.Write([]byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\nok"))
		return
	}

	if route == "/decrement" && method == "POST" {

		id := queryParams.Get("id")

		_, err := db.Exec(`UPDATE series 
			SET current_episode = current_episode - 1
			WHERE id = ? AND current_episode > 1`, id)

		if err != nil {
			log.Println("DB DECREMENT error:", err)
			conn.Write([]byte("HTTP/1.1 500 Internal Server Error\r\n\r\nDatabase error"))
			return
		}

		conn.Write([]byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\nok"))
		return
	}

	if route == "/delete" && method == "DELETE" {

		id := queryParams.Get("id")

		_, err := db.Exec("DELETE FROM series WHERE id = ?", id)
		if err != nil {
			log.Println("DB DELETE error:", err)
			conn.Write([]byte("HTTP/1.1 500 Internal Server Error\r\n\r\nDatabase error"))
			return
		}

		conn.Write([]byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\nDeleted"))
		return
	}

	if route == "/rate" && method == "POST" {

		values, _ := url.ParseQuery(body)
		id := values.Get("id")
		scoreStr := values.Get("score")

		score, err := strconv.Atoi(scoreStr)
		if err != nil || score < 0 || score > 10 {
			conn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\nScore invalido"))
			return
		}

		_, err = db.Exec("INSERT INTO ratings (series_id, score) VALUES (?, ?)", id, score)
		if err != nil {
			log.Println("DB RATING error:", err)
			conn.Write([]byte("HTTP/1.1 500 Internal Server Error\r\n\r\nDatabase error"))
			return
		}

		conn.Write([]byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\nok"))
		return
	}

	if route != "/" {
		conn.Write([]byte("HTTP/1.1 404 Not Found\r\nContent-Type: text/html\r\n\r\n<h1>404 Not Found</h1>"))
		return
	}

	rows, err := db.Query("SELECT id, name, current_episode, total_episodes FROM series")
	if err != nil {
		log.Println("DB QUERY error:", err)
		return
	}
	defer rows.Close()

	html := `
<html>
<head>
<style>
.progress { width:100px; background:#ddd; }
.bar { height:10px; background:green; }
.completed-text { color:green; font-weight:bold; }
</style>
<script>
async function nextEpisode(id){
	await fetch("/update?id="+id,{method:"POST"});
	location.reload();
}
async function prevEpisode(id){
	await fetch("/decrement?id="+id,{method:"POST"});
	location.reload();
}
async function deleteSerie(id){
	await fetch("/delete?id="+id,{method:"DELETE"});
	location.reload();
}
async function rate(id){
	let score = prompt("Score 0-10:");
	await fetch("/rate",{
		method:"POST",
		headers:{"Content-Type":"application/x-www-form-urlencoded"},
		body:"id="+id+"&score="+score
	});
	location.reload();
}
</script>
</head>
<body>

<h1>Track de Series</h1>
<a href="/create">Agregar Serie</a>

<table border="1">
<tr>
<th>ID</th>
<th>Nombre</th>
<th>Episodios</th>
<th>Progreso</th>
<th>Rating</th>
<th>Accion</th>
</tr>
`

	for rows.Next() {

		var id, current, total int
		var name string

		rows.Scan(&id, &name, &current, &total)

		percent := (current * 100) / total

		completed := ""
		if current == total {
			completed = "<span class='completed-text'>(COMPLETA)</span>"
		}

		var avgRating sql.NullFloat64
		db.QueryRow("SELECT AVG(score) FROM ratings WHERE series_id = ?", id).Scan(&avgRating)

		ratingText := "Sin rating"
		if avgRating.Valid {
			ratingText = fmt.Sprintf("%.1f / 10", avgRating.Float64)
		}

		html += fmt.Sprintf(`
<tr>
<td>%d</td>
<td>%s %s</td>
<td>%d/%d</td>
<td>
<div class="progress">
<div class="bar" style="width:%d%%"></div>
</div>
</td>
<td>%s</td>
<td>
<button onclick="prevEpisode(%d)">-1</button>
<button onclick="nextEpisode(%d)">+1</button>
<button onclick="rate(%d)">Rate</button>
<button onclick="deleteSerie(%d)">Delete</button>
</td>
</tr>`,
			id, name, completed,
			current, total,
			percent,
			ratingText,
			id, id, id, id)
	}

	html += "</table></body></html>"

	conn.Write([]byte("HTTP/1.1 200 OK\r\nContent-Type: text/html\r\n\r\n" + html))
}
