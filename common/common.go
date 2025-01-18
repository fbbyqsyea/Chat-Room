package common

const (
	// MessageType 表示消息类型
	ChatMessageType   = "chat"
	SystemMessageType = "system"
)

// 消息类型
type Message struct {
	Type    string `json:"type"`
	From    string `json:"from"`
	Message string `json:"message"`
}
