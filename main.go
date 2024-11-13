package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)


const (
	CONN_TYPE = "tcp"

	CMD_PREFIX = "/"
	CMD_CREATE = CMD_PREFIX + "create"
	CMD_LIST   = CMD_PREFIX + "list"
	CMD_JOIN   = CMD_PREFIX + "join"
	CMD_LEAVE  = CMD_PREFIX + "leave"
	CMD_HELP   = CMD_PREFIX + "help"
	CMD_NAME   = CMD_PREFIX + "name"
	CMD_QUIT   = CMD_PREFIX + "quit"
	CMD_INIT   = CMD_PREFIX + "init"

	MSG_CONNECT = "Welcome to the server! Type \"/help\" to get a list of commands.\n"
	MSG_FULL    = "Server is full. Please try reconnecting later."
)


type Client struct {
	name string
	chatRoom *ChatRoom
	incoming chan *Message
	outgoing chan string
	conn net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
}

func (client *Client) Quit() {
	client.conn.Close()
}

func NewClient(chatRoom *ChatRoom, conn net.Conn) *Client {
	log.Println("client")
	client := &Client {
		name:     "",
		chatRoom: chatRoom,
		incoming: make(chan *Message),
		outgoing: make(chan string),
		conn:     conn,
		reader:   nil,
		writer:   nil,
	}
	client.reader = bufio.NewReader(conn)
	client.writer = bufio.NewWriter(conn)
	client.Listen()
	return client
}

func (client *Client) Read() {
	go func() {
		for message := range client.incoming {
			switch {
			default:
				fmt.Print(message.String())
			case strings.HasPrefix(message.text, CMD_INIT):
				client.name = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(message.text, CMD_INIT+" "), "\n"))
			}
		}
	}()
}

func (chatRoom *ChatRoom) Broadcast(message string) {
	for _, client := range chatRoom.clients {
		client.outgoing <- message
	}
}

func (client *Client) Write() {
	for message := range client.outgoing {

		_, err := client.writer.WriteString(message)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		err = client.writer.Flush()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
}

func (client *Client) Listen() {
	go client.Read()
	go client.Write()
	go client.ClientRead(client.conn)
	go client.ClientWrite(client.conn)
}

func NewMessage(time time.Time, client *Client, text string) *Message {
	return &Message{
		time:   time,
		client: client,
		text:   text,
	}
}

type ChatRoom struct {
	myName string
	clients  []*Client
	incoming chan *Message
	join chan *Client
}

func NewChatRoom() *ChatRoom {

	chatRoom := &ChatRoom{
		myName: "",
		clients: make([]*Client, 0),
		incoming: make(chan *Message),
		join: make(chan *Client),
	}

	return chatRoom
}

func (chatRoom *ChatRoom) Join(client *Client) {
	chatRoom.clients = append(chatRoom.clients, client)
}

type Message struct {
	time time.Time
	client *Client
	text string
}

func (message *Message) String() string {
	return fmt.Sprintf("%s - %s: %s\n", message.time.Format(time.Kitchen), message.client.name, message.text)
}

func (client *Client)ClientRead(conn net.Conn) {
	reader := bufio.NewReader(conn)
	for {
		str, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		message := NewMessage(time.Now(), client, strings.TrimSuffix(str, "\n"))
		client.incoming <- message
	}
}

// Reads from Stdin, and outputs to the socket.
func (client *Client)ClientWrite(conn net.Conn) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println(client.chatRoom.clients)

	client.chatRoom.Broadcast(CMD_INIT + " " + client.chatRoom.myName)

	for {
		str, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		client.chatRoom.Broadcast(str)
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	if len(os.Args) <= 1 {
		log.Println("required server port as parameter 1 and port to connect to as parameter 2(optional)")
		os.Exit(1)
	}

	connPort := os.Args[1]
	chatRoom := NewChatRoom()

	listener, err := net.Listen(CONN_TYPE, "localhost" + connPort)
	if err != nil {
		log.Println("Error", err)
		os.Exit(1)
	}
	defer listener.Close()
	log.Println("listening on port", "localhost" + connPort)
	
	reader := bufio.NewReader(os.Stdin)
	
	log.Print("Enter name: ")
	text, err := reader.ReadString('\n')
	if err != nil {
		log.Println(err)
	}

	chatRoom.myName = text

	if len(os.Args) >= 3 {

		conn, err := net.Dial(CONN_TYPE, os.Args[2])
		if err != nil {
			fmt.Println(err)
		}
		chatRoom.Join(NewClient(chatRoom, conn))
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Error: ", err)
			continue
		}
		chatRoom.Join(NewClient(chatRoom, conn))
	}
}