package main

import (
	"bufio"
	"fmt"
	"net"
	"sync"

	"github.com/bytedance/sonic"
	"github.com/fbbyqsyea/Chat-Room/common"
)

var (
	clients    = make(map[net.Conn]string) // 存储所有连接的客户端
	clientLock sync.Mutex                  // 用于保护clients的并发操作
)

func main() {
	// 启动服务器监听
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}
	defer listener.Close()
	fmt.Println("Chatroom server started on port 8080...")

	for {
		// 等待客户端连接
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Connection error:", err)
			continue
		}
		// 为每个客户端开启一个goroutine
		go handleClient(conn)
	}
}

// 处理每个客户端的连接
func handleClient(conn net.Conn) {
	defer func() {
		disconnectClient(conn)
		conn.Close()
	}()

	// 初次连接时要求用户输入昵称
	reader := bufio.NewReader(conn)
	nickname, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading nickname:", err)
		return
	}
	nickname = nickname[:len(nickname)-1] // 去掉换行符
	if nickname == "" {
		fmt.Println("Invalid nickname.")

		return
	}

	// 将客户端加入到clients
	clientLock.Lock()
	clients[conn] = nickname
	clientLock.Unlock()

	broadcast(common.SystemMessageType, fmt.Sprintf("%s has joined the chatroom.\n", nickname), conn)
	fmt.Printf("%s connected.\n", nickname)

	// 处理消息
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("%s disconnected.\n", nickname)
			break
		}
		broadcast(common.ChatMessageType, fmt.Sprintf("%s: %s", nickname, message), conn)
	}
}

// 广播消息给所有客户端
func broadcast(messageType string, message string, sender net.Conn) {
	clientLock.Lock()
	defer clientLock.Unlock()
	for client := range clients {
		if client != sender { // 不发给发送者
			data, _ := sonic.MarshalString(common.Message{
				Type:    messageType,
				From:    clients[sender],
				Message: message,
			})
			_, _ = client.Write([]byte(data + "\n"))
		}
	}
	fmt.Print(message) // 在服务端打印广播消息
}

// 客户端断开连接
func disconnectClient(conn net.Conn) {
	clientLock.Lock()
	defer clientLock.Unlock()
	nickname := clients[conn]
	delete(clients, conn)
	broadcast(common.SystemMessageType, fmt.Sprintf("%s has left the chatroom.\n", nickname), conn)
}
