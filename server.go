package main

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type Server struct {
	Ip string
	Port int
	OnlineMap map[string]*User		//在线用户对象是一个map集合userName:user
	mapLock sync.RWMutex			//锁
	Message chan string				//广播channel
}
//创建一个Server的接口
func NewServer(ip string, port int) *Server {
	server := &Server{
		Ip: ip,
		Port: port,
		OnlineMap: make(map[string]*User),
		Message: make(chan string),
	}
	return server
}

//连接操作
func (s *Server)Handle(coon net.Conn)  {
	//fmt.Println("连接成功")
	user := NewUser(coon, s)
	//用户上线
	user.Online()

	isLive := make(chan bool)

	//开启一个go协程 用于处理客户端的消息
	go func() {
		buff := make([]byte, 4096)
		for {
			n, err := coon.Read(buff)
			if n == 0 {
				//用户下线
				user.OffLine()
				return
			}
			if err != nil && err != io.EOF {
				fmt.Println("Conn Read Err: ..", err)
				return
			}
			msg := string(buff[:n-1])
			// s.BroadCast(user, msg)
			user.DoMessage(msg)
			//一旦发送消息 表示该用户活跃
			isLive <- true
		}
	}()

	select {
	case <- isLive:
		//代表当前用户是活跃的 不进行操作 进入下一行代码
	//开启一个定时器 为10秒钟
	case <-time.After(time.Second * 30):
		//已经超时
		user.sendMsg("你被踢了")
		close(user.C)
		coon.Close()
		return
	}
}

//广播消息
func (s *Server)BroadCast(user *User, msg string) {
	sendMsg := "[" + user.Addr + "]" + user.Name + msg
	s.Message <- sendMsg
}

//监听Message广播channel的goroutine 一旦有消息就发送给全部在线的User
func (s *Server) ListenMessage(){
	for {
		msg := <- s.Message
		s.mapLock.Lock()
		for _, cli := range s.OnlineMap{
			cli.C <- msg
		}
		s.mapLock.Unlock()
	}

}

func (s *Server) Start() {
	//创建连接
	listen, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.Ip, s.Port))
	if err != nil {
		fmt.Println("net.Listen is err ", err)
		return
	}
	//一直开启监听Message
	go s.ListenMessage()

	//关闭连接
	defer listen.Close()

	for {
		coon, err := listen.Accept()
		if err != nil {
			fmt.Println("listen is not accept ")
			continue
		}
		//业务操作
		go s.Handle(coon)

	}

}