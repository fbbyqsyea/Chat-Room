package common

import (
	"errors"
	"net"
)

const (
	// MessageType 表示消息类型
	ChatMessageType   = "chat"   // 聊天内容 消息类型
	IpMessageType     = "ip"     // ip 消息类型
	JoinedMessageType = "joined" // 加入房间 消息类型
	LeftMessageType   = "left"   // 离开房间 消息类型
	OtherMessageType  = "other"  // 其他消息类型
)

// 消息类型
type Message struct {
	Type    string `json:"type"`
	From    string `json:"from"`
	Message string `json:"message"`
}

func MustIPv4() string {
	ipv4, _ := IPv4()
	return ipv4
}

func IPv4() (string, error) {
	ipv4s, err := IPv4S()
	if err != nil {
		return "", err
	}
	if len(ipv4s) > 0 {
		return ipv4s[0], nil
	}
	return "", errors.New("get ipv4 failed")
}

func IPv4S() ([]string, error) {
	ipv4 := make([]string, 0)
	// 获取所有网络接口的信息
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	// 遍历网络接口
	for _, iface := range interfaces {
		// 排除回环接口和无效接口
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}
		// 获取接口的地址信息
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		// 遍历地址信息，找到IPv4地址
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if ok && !ipNet.IP.IsLoopback() && ipNet.IP.To4() != nil {
				ipv4 = append(ipv4, ipNet.IP.String())
			}
		}
	}
	return ipv4, nil
}
