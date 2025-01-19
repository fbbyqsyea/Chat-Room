package main

import (
	"bufio"
	"fmt"
	"net"
	"sync"

	"github.com/bytedance/sonic"
	"github.com/fbbyqsyea/Chat-Room/common"
	"github.com/sirupsen/logrus"
)

var (
	clients    = make(map[net.Conn]string) // 存储所有连接的客户端
	clientLock sync.Mutex                  // 用于保护clients的并发操作
	logger     = logrus.New()              // 使用 logrus 记录日志
)

func init() {
	// 设置log格式
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	// 可以将日志输出到文件，方便后期调试
	// logger.SetOutput(os.Stdout)
}

func main() {
	// 启动服务器监听
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		logger.Fatalf("Error starting server: %v", err)
		return
	}
	defer listener.Close()
	logger.Info("Chatroom server started on port 8080...")

	for {
		// 等待客户端连接
		conn, err := listener.Accept()
		if err != nil {
			logger.Errorf("Connection error: %v", err)
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

	// 处理消息
	for {
		message := common.Message{}
		data, err := reader.ReadString('\n')
		if err != nil {
			logger.Errorf("Error reading message: %v", err)
			return
		}

		err = sonic.UnmarshalString(data, &message)
		if err != nil {
			logger.Errorf("Error unmarshalling message: %v", err)
			continue
		}

		// 处理不同类型的消息
		switch message.Type {
		case common.IpMessageType:
			ip := message.Message
			// 将客户端加入到clients
			clientLock.Lock()
			clients[conn] = ip
			clientLock.Unlock()

			// 广播加入消息
			broadcast(&common.Message{
				Type:    common.JoinedMessageType,
				From:    ip,
				Message: fmt.Sprintf("%s has joined the chatroom.", ip),
			}, conn)

			logger.Infof("%s connected.", ip)

		case common.ChatMessageType:
			// 广播聊天消息
			broadcast(&message, conn)
		}
	}
}

// 广播消息给所有客户端
func broadcast(message *common.Message, sender net.Conn) {
	// 使用 goroutine 进行并行发送，避免阻塞
	for client := range clients {
		if client != sender { // 不发给发送者自己
			data, err := sonic.MarshalString(message)
			if err != nil {
				logger.Errorf("Error marshalling message: %v", err)
				continue
			}
			_, err = client.Write([]byte(data + "\n"))
			if err != nil {
				logger.Errorf("Error sending message to %s: %v", clients[client], err)
			} else {
				logger.Infof("Message sent to %s", clients[client])
			}
		}
	}
}

// 客户端断开连接
func disconnectClient(conn net.Conn) {
	// 获取断开的客户端IP
	ip := clients[conn]

	// 从clients中删除该客户端
	clientLock.Lock()
	delete(clients, conn)
	clientLock.Unlock()

	// 广播离开消息
	broadcast(&common.Message{
		Type:    common.LeftMessageType,
		From:    ip,
		Message: fmt.Sprintf("%s has left the chatroom.", ip),
	}, conn)

	logger.Infof("%s disconnected.", ip)
}
