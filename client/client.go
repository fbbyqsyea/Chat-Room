package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/bytedance/sonic"
	"github.com/fbbyqsyea/Chat-Room/common"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	host         = "127.0.0.1"
	port         = "8080"
	app          = tview.NewApplication()
	nickname     string
	isNicknameOk = false
	serverConn   net.Conn
	chatHistory  *tview.TextView
	systemInfo   *tview.TextView
	inputField   *tview.InputField
	clientLock   sync.Mutex
)

func init() {
	flag.StringVar(&host, "host", host, "Server host")
	flag.StringVar(&port, "port", port, "Server port")
	flag.Parse()
}

func main() {
	var err error
	// 连接到服务器
	serverConn, err = net.Dial("tcp", fmt.Sprintf("%s:%s", host, port))
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		return
	}
	defer serverConn.Close()

	// 创建昵称输入界面
	nicknameInput := tview.NewInputField()
	nicknameInput.
		SetLabel("Enter your nickname: ").
		SetFieldBackgroundColor(tview.Styles.ContrastBackgroundColor).
		SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyEnter {
				for !isNicknameOk {
					nickname = strings.TrimSpace(nicknameInput.GetText())
					if nickname != "" {
						// 发送昵称到服务器
						fmt.Fprintf(serverConn, "%s\n", nickname)
					}
				}
				setupChatUI() // 设置聊天界面
				app.SetRoot(setupChatLayout(), true)
			}
		})

	// 设置昵称输入为根布局
	if err := app.SetRoot(nicknameInput, true).Run(); err != nil {
		panic(err)
	}
}

// 设置聊天界面
func setupChatUI() {
	chatHistory = tview.NewTextView()
	chatHistory.
		SetDynamicColors(true).
		SetBorder(true).
		SetTitle(" Chat History ")

	systemInfo = tview.NewTextView()
	systemInfo.
		SetDynamicColors(true).
		SetBorder(true).
		SetTitle(" System Info ")

	inputField = tview.NewInputField().
		SetLabel("> ").
		SetFieldBackgroundColor(tview.Styles.ContrastBackgroundColor).
		SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyEnter {
				message := inputField.GetText()
				if strings.TrimSpace(message) != "" {
					sendMessage(message)
					inputField.SetText("")
				}
			}
		})

	go receiveMessages() // 开启消息接收
}

// 聊天界面布局
func setupChatLayout() *tview.Flex {
	return tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(chatHistory, 0, 4, false).
		AddItem(systemInfo, 0, 1, false).
		AddItem(inputField, 1, 0, true)
}

// 接收消息
func receiveMessages() {
	reader := bufio.NewReader(serverConn)
	for {
		data, err := reader.ReadString('\n')
		if err != nil {
			displaySystemMessage("Disconnected from server.")
			app.Stop()
			os.Exit(0)
		}
		message := common.Message{}
		err = sonic.UnmarshalString(data, &message)
		if err != nil {
			displaySystemMessage(fmt.Sprintf("Failed to parse message. err: %v", err))
		}
		switch message.Type {
		case common.ChatMessageType:
			displayChatMessage(fmt.Sprintf("[%s]: %s\n", message.From, message.Message))
		case common.SystemMessageType:
			displaySystemMessage(fmt.Sprintf("%s\n", message.Message))
		}
	}
}

// 发送消息
func sendMessage(message string) {
	_, err := serverConn.Write([]byte(message + "\n"))
	if err != nil {
		displaySystemMessage("Failed to send message.")
	}
}

// 显示聊天消息
func displayChatMessage(message string) {
	clientLock.Lock()
	defer clientLock.Unlock()

	_, _ = chatHistory.Write([]byte(message))
	chatHistory.ScrollToEnd()
}

// 显示系统消息
func displaySystemMessage(message string) {
	clientLock.Lock()
	defer clientLock.Unlock()

	_, _ = systemInfo.Write([]byte(message + "\n"))
	systemInfo.ScrollToEnd()
}
