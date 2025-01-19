package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/fbbyqsyea/Chat-Room/common"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ChatClient struct {
	host        string
	port        string
	app         *tview.Application
	ip          string
	serverConn  net.Conn
	chatHistory *tview.TextView
	systemInfo  *tview.TextView
	inputField  *tview.InputField
	clientLock  sync.Mutex
}

func NewChatClient(host, port string) *ChatClient {
	return &ChatClient{
		host: host,
		port: port,
		app:  tview.NewApplication(),
		ip:   common.MustIPv4(),
	}
}

func (c *ChatClient) initUI() {
	// Initialize UI components for chat history, system info, and input field
	c.chatHistory = tview.NewTextView().
		SetScrollable(true).
		SetDynamicColors(true).
		SetChangedFunc(func() {
			c.app.ForceDraw()
		})
	c.chatHistory.
		SetBorder(true).
		SetTitle(" Chat History ")

	c.systemInfo = tview.NewTextView().
		SetScrollable(true).
		SetDynamicColors(true).
		SetChangedFunc(func() {
			c.app.ForceDraw()
		})
	c.systemInfo.
		SetBorder(true).
		SetTitle(" System Info ")

	c.inputField = tview.NewInputField().
		SetLabel("> ").
		SetFieldBackgroundColor(tview.Styles.ContrastBackgroundColor).
		SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyEnter {
				message := c.inputField.GetText()
				if strings.TrimSpace(message) != "" {
					c.sendMessage(common.ChatMessageType, message)
					// 显示你的消息
					c.displayChatMessage(fmt.Sprintf("%s: %s\n", "Your", message))
					c.inputField.SetText("")
				}
			}
		})

	c.app.EnableMouse(true)
}

func (c *ChatClient) setupChatLayout() *tview.Flex {
	// Set up layout with chat history, system info, and input field
	return tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(c.chatHistory, 0, 4, false).
		AddItem(c.systemInfo, 0, 1, false).
		AddItem(c.inputField, 1, 0, true)
}

func (c *ChatClient) connectToServer() error {
	// Try to establish a connection to the server
	var err error
	c.serverConn, err = net.Dial("tcp", fmt.Sprintf("%s:%s", c.host, c.port))
	if err != nil {
		return fmt.Errorf("error connecting to server: %v", err)
	}
	return nil
}

func (c *ChatClient) reconnectToServer() error {
	// Retry connection after a brief delay in case of failure
	for {
		err := c.connectToServer()
		if err == nil {
			return nil
		}
		c.displaySystemMessage(fmt.Sprintf("Reconnection failed, retrying in 5 seconds: %v", err))
		time.Sleep(5 * time.Second)
	}
}

func (c *ChatClient) receiveMessages() {
	c.displaySystemMessage("Connected to server.")
	reader := bufio.NewReader(c.serverConn)

	// Continuously read messages from the server
	for {
		data, err := reader.ReadString('\n')
		if err != nil {
			c.displaySystemMessage("Disconnected from server.")
			// Try to reconnect if disconnected
			c.reconnectToServer()
			continue
		}

		// Handle incoming message
		message := common.Message{}
		if err := sonic.UnmarshalString(data, &message); err != nil {
			c.displaySystemMessage(fmt.Sprintf("Failed to parse message. err: %v", err))
			continue
		}
		c.handleReceivedMessage(message)
	}
}

func (c *ChatClient) handleReceivedMessage(message common.Message) {
	// Handle the received message based on its type
	switch message.Type {
	case common.ChatMessageType:
		c.displayChatMessage(fmt.Sprintf("%s: %s\n", message.From, message.Message))
	case common.LeftMessageType:
		c.serverConn = nil
		c.displaySystemMessage(message.Message)
	case common.JoinedMessageType, common.OtherMessageType:
		c.displaySystemMessage(message.Message)
	}
}

func (c *ChatClient) sendMessage(messageType, message string) {
	// Serialize and send the message to the server
	data, err := sonic.MarshalString(common.Message{
		Type:    messageType,
		From:    c.ip,
		Message: message,
	})
	if err != nil {
		c.displaySystemMessage("Failed to marshal message.")
		return
	}

	_, err = c.serverConn.Write([]byte(data + "\n"))
	if err != nil {
		c.displaySystemMessage("Failed to send message.")
	}
}

func (c *ChatClient) displayChatMessage(message string) {
	// Safely update the chat history in the UI
	c.clientLock.Lock()
	defer c.clientLock.Unlock()

	_, _ = c.chatHistory.Write([]byte(message))
	c.chatHistory.ScrollToEnd()
}

func (c *ChatClient) displaySystemMessage(message string) {
	// Safely update the system info in the UI
	c.clientLock.Lock()
	defer c.clientLock.Unlock()

	_, _ = c.systemInfo.Write([]byte(message + "\n"))
	c.systemInfo.ScrollToEnd()
}

func (c *ChatClient) registerClient() {
	// Send the client's IP address to the server
	c.displaySystemMessage(fmt.Sprintf("Your IP: %s", c.ip))
	c.sendMessage(common.IpMessageType, c.ip)
}

func main() {
	// Command-line flags for host and port
	host := flag.String("host", "127.0.0.1", "Server host")
	port := flag.String("port", "8080", "Server port")
	flag.Parse()

	client := NewChatClient(*host, *port)
	client.initUI()

	// Connect to the server
	if err := client.connectToServer(); err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer client.serverConn.Close()

	// Register client and run the UI
	client.registerClient()

	// Start the message receiving goroutine
	go client.receiveMessages()

	// Run the application
	if err := client.app.SetRoot(client.setupChatLayout(), true).Run(); err != nil {
		log.Fatalf("Failed to run application: %v", err)
	}
}
