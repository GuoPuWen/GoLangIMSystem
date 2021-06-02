package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
)

type Client struct {
	ServerIp string
	ServerPort int
	Name string
	conn net.Conn
	flag int
}
//创建连接函数
func NewClient(ServerIp string, serverPort int) *Client{
	client := &Client{
		ServerIp: ServerIp,
		ServerPort: serverPort,
		flag: 999,
	}
	conn, err := net.Dial("tcp",fmt.Sprintf("%s:%d", ServerIp, serverPort))
	if(err != nil){
		fmt.Println("net.Dial is Err")
	}
	client.conn = conn
	return client
}

var ServerIp string
var ServerPort int

//main函数之前执行init函数
func init()  {
	flag.StringVar(&ServerIp, "ip", "127.0.0.1","设置服务器的ip地址 默认为127.0.0.1")
	flag.IntVar(&ServerPort, "port", 8890, "设置服务器的端口，默认端口为8890")
}

//菜单
func (client *Client) Menu() bool {
	var flag int
	fmt.Println("1.公聊模式")
	fmt.Println("2.私聊模式")
	fmt.Println("3.更改用户名")
	fmt.Println("0.退出")

	fmt.Scanln(&flag)

	if flag >= 0 && flag <= 3 {
		client.flag = flag
		return true
	}else {
		fmt.Println("输入合法的指令")
		return false
	}
}

func (client *Client) Run() {
	for client.flag != 0 {
		for client.Menu() != true{}
		switch client.flag {
		case 1:
			fmt.Println("公聊模式")
			client.PublicChat()
			break
		case 2:
			fmt.Println("私聊模式")
			client.PrivateChat()
			break
		case 3:
			fmt.Println("更改用户名")
			client.updateName()
			break

		}
	}
}
//修改用户名
func (client *Client) updateName() bool {
	fmt.Println(">>>> 请输入用户名")
	var name string
	fmt.Scanln(&name)
	sendMsg := "rename|" + name + "\n"
	_, err := client.conn.Write([]byte(sendMsg))
	if err != nil {
		fmt.Println("conn 写入失败")
		return false
	}
	return true

}

//公聊模式
func (client *Client) PublicChat() bool {
	fmt.Println("请输入聊天内容，exit退出")
	var chatMsg string
	fmt.Scanln(&chatMsg)
	for chatMsg != "exit" {
		if len(chatMsg) != 0 {
			sendMsg := chatMsg + "\n"
			_, err := client.conn.Write([]byte(sendMsg))
			if err != nil {
				fmt.Println("发送失败")
				return false
			}
			sendMsg = ""
			fmt.Println("请输入聊天内容，exit退出")
			fmt.Scanln(&chatMsg)
		}
	}
	return true
}

//查询用户上线情况
func (client *Client) SelectUsers() {
	sendMsg := "who"
	_, err := client.conn.Write([]byte(sendMsg))
	if err != nil {
		fmt.Println("coon write err")
		return
	}
	return
}
//发送私聊消息
func (client *Client) PrivateChat() bool {
	var remoteName string
	var chatMsg string
	go client.SelectUsers()
	fmt.Println(">>>请输入聊天对象[用户名]，exit退出")
	fmt.Scanln(&remoteName)
	for remoteName != "exit" {
		fmt.Println("请输入聊天消息，exit退出")
		fmt.Scanln(&chatMsg)
		for chatMsg != "exit" {
			if len(chatMsg) != 0 {
				sendMsg := "to|" + remoteName + "|" + chatMsg + "\n"
				_, err := client.conn.Write([]byte(sendMsg))
				if err != nil {
					fmt.Println("coon write err")
					return false
				}
			}
			chatMsg = ""
			fmt.Println("请输入聊天消息，exit退出")
			fmt.Scanln(&chatMsg)
		}
		remoteName = ""
		fmt.Println(">>>请输入聊天对象[用户名]，exit退出")
		fmt.Scanln(&remoteName)
	}
	return true
	
}

//接收serve端的消息的go程
func (client *Client) DealMessage() {
	io.Copy(os.Stdin, client.conn)
}



func main() {
	//命令行解析
	flag.Parse()

	client := NewClient(ServerIp, ServerPort)
	if client == nil {
		fmt.Println(">>>>>>>>>>>> 连接服务器失败")
		return
	}
	fmt.Println(">>>>>>>>>>>> 连接服务器成功")
	go client.DealMessage()
	client.Run()
	select {
	}


}

