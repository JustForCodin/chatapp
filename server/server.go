package server

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

type ChatMessage struct {
	Username string
	Text     string
}

var clients = make(map[*websocket.Conn]bool)
var broadcaster = make(chan *ChatMessage)
var upgrader = websocket.Upgrader{
	WriteBufferSize: 1024,
	ReadBufferSize:  1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func handleConnection(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("New client connected successfully")

	defer ws.Close()
	clients[ws] = true

	for {
		var msg *ChatMessage
		err := ws.ReadJSON(&msg)
		if err != nil {
			delete(clients, ws)
			break
		}
		broadcaster <- msg
	}
}

func handleMessage() {
	for {
		msg := <-broadcaster
		storeInDB(msg)
		sendMessageToAll(*msg)
	}
}

func storeInDB(msg *ChatMessage) {
	err := godotenv.Load(".env")

	dbConn, err := sql.Open("mysql", os.Getenv("DB_URL"))
	table := os.Getenv("TABLE_NAME")
	if err != nil {
		log.Fatal(err)
	}
	defer dbConn.Close()

	// var ms ChatMessage
	insert, err := dbConn.Query("INSERT INTO " + table + "(username, text)" +
		" VALUES(" + "\"" + msg.Username + "\"" + ", " + "\"" + msg.Text + "\"" + ");")

	if err != nil {
		log.Fatal(err)
	}
	defer insert.Close()
	fmt.Println("New record was successfully added to MySQL DB.")
}

func sendMessageToClient(client *websocket.Conn, msg ChatMessage) {
	err := client.WriteJSON(msg)
	if err != nil {
		log.Println(err)
		client.Close()
		delete(clients, client)
	}
}

func sendMessageToAll(msg ChatMessage) {
	for client := range clients {
		sendMessageToClient(client, msg)
	}
}

func disPlayPreviousMessages(dbConn *sql.DB, src string) {
	res, _ := dbConn.Query("SELECT * FROM " + src)
	fmt.Println(res)
}

func index(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Chat web server"))
}

func foo() {

}

func Start() {
	port := ":7334"

	// setup routes
	http.HandleFunc("/", index)
	http.HandleFunc("/ws", handleConnection)
	go handleMessage()
	log.Print("Server started on port:" + port)

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal(err)
	}
}
