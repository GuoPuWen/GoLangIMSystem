后端人员快速入门go语言，上手项目。是一个通信系统，覆盖了go语言的基本语法以及go程的使用

# 一、构造基本Serve

构造基本的Serve，就是Socket编程，同时在处理操作时，开启go的协程处理

serve.go

```go
package main

import (
	"fmt"
	"net"
)

type Server struct {
	Ip string
	Port int
}
//创建一个Server的接口
func NewServer(ip string, port int) *Server {
	server := &Server{
		Ip: ip,
		Port: port,
	}
	return server
}

func (s *Server)Handle(coon net.Conn)  {
	fmt.Println("连接成功")
}

func (s *Server) Start() {
	//创建连接
	listen, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.Ip, s.Port))
	if err != nil {
		fmt.Println("net.Listen is err")
		return
	}

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
```

main.go

```go
package main

func main() {
   server := NewServer("127.0.0.1", 8888)
   server.Start()
}
```

# 二、上线通知

v2版本需要做的事情是当一个用户连接的时候，需要广播这个用户上线的消息

思路是在对每一个连接的用户首先封装一个类，同时这个类绑定一个Channel。同时在服务端需要维护一个OnlineMap对象以及Message的Channel，当用户上线的时候，添加进OnlineMap，同时广播消息。

广播消息在服务端需要做的事情就是将消息写入到Message这个Channel中，同时一直监听Message这个状态，如果发现有消息，就遍历OnlineMap对象，将消息写入到对应的用户的Channel中

user.go

```go
package main

import (
	"net"
)

//创建一个User对象 封装客户端信息
type User struct {
	Name string
	Addr string
	C chan string
	Conn net.Conn
}

//创建一个User对象
func NewUser(conn net.Conn) *User{
	userAddr := conn.RemoteAddr().String()
	user := &User{
		Name: userAddr,
		Addr: userAddr,
		C: make(chan string),
		Conn: conn,
	}
	//当User被创建的时候就要进行监听消息
	go user.listenMessge()
	return user
}
//监听消息
func (u *User)listenMessge() {
	for {
		msg := <- u.C
		u.Conn.Write([]byte(msg + "\n"))
	}
}

```

server.go

```go
package main

import (
	"fmt"
	"net"
	"sync"
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

func (s *Server)Handle(coon net.Conn)  {
	//fmt.Println("连接成功")
	user := NewUser(coon)
	s.mapLock.Lock()
	s.OnlineMap[user.Name] = user
	s.mapLock.Unlock()
	//广播消息
	s.BroadCast(user, "已上线")
	select {
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
		fmt.Println("net.Listen is err")
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
```

# 三、用户消息广播

思路：服务端需要接受客户端发送来的消息，收到消息之后，进行广播转发即可

```go
//开启一个go协程 用于处理客户端的消息
	go func() {
		buff := make([]byte, 4096)
		for {
			n, err := coon.Read(buff)
			if n == 0 {
				s.BroadCast(user, "Be offLine")
				return
			}
			if err != nil && err != io.EOF {
				fmt.Println("Conn Read Err: ..", err)
				return
			}
			msg := string(buff)
			s.BroadCast(user, msg)
		}
	}()
```

# 四、业务层分离

上面代码在编写的过程中，将用户上线，用户下线以及消息处理都写在了serve.go的代码中，现在将其分离处理，将这些代码写在user.go文件中，只需要在user.go中新增一个server的变量，每次处理消息的时候都调用server端的代码

user.go

```go
package main

import (
	"net"
)

//创建一个User对象 封装客户端信息
type User struct {
	Name string
	Addr string
	C chan string
	Conn net.Conn
	server *Server
}

//创建一个User对象
func NewUser(conn net.Conn, server *Server) *User{
	userAddr := conn.RemoteAddr().String()
	user := &User{
		Name: userAddr,
		Addr: userAddr,
		C: make(chan string),
		Conn: conn,
		server: server,
	}
	//当User被创建的时候就要进行监听消息
	go user.listenMessge()
	return user
}
//监听消息
func (u *User)listenMessge() {
	for {
		msg := <- u.C
		u.Conn.Write([]byte(msg + "\n"))
	}
}
//用户上线消息
func (u *User) Online(){
	u.server.mapLock.Lock()
	u.server.OnlineMap[u.Name] = u
	u.server.mapLock.Unlock()

	u.server.BroadCast(u, "已上线")
}

//用户下线
func (u *User) OffLine(){
	u.server.mapLock.Lock()
	delete(u.server.OnlineMap, u.Name)
	u.server.mapLock.Unlock()

	u.server.BroadCast(u, "已下线")
}

//处理消息
func (u *User) DoMessage(msg string){
	u.server.BroadCast(u, msg)
}
```

server.go

```go
package main

import (
	"fmt"
	"io"
	"net"
	"sync"
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
		}
	}()
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
		fmt.Println("net.Listen is err")
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
```

# 五、在线用户查询

指定消息格式 who 当客户端发送who指令时，向该客户端发送当前在线的用户，只需要将OnlineMap查询出来即可

```go
//处理消息
func (u *User) DoMessage(msg string){
	if msg == "who" {
		//查询当前在线用户
		u.server.mapLock.Lock()
		for _, user := range u.server.OnlineMap {
			onlineMsg := "[" + user.Addr + "]" + user.Name + ":" + "在线..\n"
			u.sendMsg(onlineMsg)
		}
		u.server.mapLock.Unlock()
	} else {
		u.server.BroadCast(u, msg)
	}
	
}
```

# 六、修改用户名

用户消息格式 "rename|用户名"，有了前面的基础就很好写出代码

```go
//处理消息
func (u *User) DoMessage(msg string){
	if msg == "who" {
		//查询当前在线用户
		u.server.mapLock.Lock()
		for _, user := range u.server.OnlineMap {
			onlineMsg := "[" + user.Addr + "]" + user.Name + ":" + "在线..\n"
			u.sendMsg(onlineMsg)
		}
		u.server.mapLock.Unlock()
	}else if len(msg) > 7 && msg[:7] == "rename|" {
		newName := strings.Split(msg, "|")[1]
		_, ok := u.server.OnlineMap[newName]
		if ok {
			u.sendMsg("该用户名已被占用")
		}else {
			u.server.mapLock.Lock()
			delete(u.server.OnlineMap, u.Name)
			u.Name = newName
			u.server.OnlineMap[newName] = u
			u.server.mapLock.Unlock()
			u.sendMsg("已经成功修改" + u.Name + "\n")
		}

	}else {
		u.server.BroadCast(u, msg)
	}
	
}
```

![image-20210601151455402](http://cdn.noteblogs.cn/image-20210601151455402.png)

# 七、私聊模式

同样是在DoMessage方法中处理

```java
func (u *User) DoMessage(msg string){
	if msg == "who" {
		//查询当前在线用户
		u.server.mapLock.Lock()
		for _, user := range u.server.OnlineMap {
			onlineMsg := "[" + user.Addr + "]" + user.Name + ":" + "在线..\n"
			u.sendMsg(onlineMsg)
		}
		u.server.mapLock.Unlock()
	}else if len(msg) > 7 && msg[:7] == "rename|" {
		newName := strings.Split(msg, "|")[1]
		_, ok := u.server.OnlineMap[newName]
		if ok {
			u.sendMsg("该用户名已被占用")
		}else {
			u.server.mapLock.Lock()
			delete(u.server.OnlineMap, u.Name)
			u.Name = newName
			u.server.OnlineMap[newName] = u
			u.server.mapLock.Unlock()
			u.sendMsg("已经成功修改" + u.Name + "\n")
		}
	//处理发送私聊消息
	} else if len(msg) > 4 && msg[:3] == "to|"{
		remoteName := strings.Split(msg, "|")[1]
		if remoteName == "" {
			u.sendMsg("消息格式不正确")
			return
		}
		remoteUser, ok := u.server.OnlineMap[remoteName]
		if !ok {
			u.sendMsg("当前用户名不存在")
			return
		}
		content := strings.Split(msg, "|")[2]
		if content == "" {
			u.sendMsg("内容不能为空")
			return
		}
		remoteUser.sendMsg(u.Name + "对你说：" + content)
	} else {
		u.server.BroadCast(u, msg)
	}
	
}
```



![image-20210601155951716](http://cdn.noteblogs.cn/image-20210601155951716.png)

# 八、超时强踢功能



```go
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
	case <-time.After(time.Second * 10):
		//已经超时
		user.sendMsg("你被踢了")
		close(user.C)
		coon.Close()
		return 
	}
}
```

![image-20210601161029953](http://cdn.noteblogs.cn/image-20210601161029953.png)

# 九、客户端

v1 ：简单版的客户端连接功能

```go
package main

import (
	"fmt"
	"net"
)

type Client struct {
	ServerIp string
	ServerPort int
	Name string
	conn net.Conn
}
//创建连接函数
func NewClient(ServerIp string, serverPort int) *Client{
	client := &Client{
		ServerIp: ServerIp,
		ServerPort: serverPort,
	}
	conn, err := net.Dial("tcp",fmt.Sprintf("%s:%d", ServerIp, serverPort))
	if(err != nil){
		fmt.Println("net.Dial is Err")
	}
	client.conn = conn
	return client
}



func main() {
	client := NewClient("127.0.0.1", 8890)
	if client == nil {
		fmt.Println(">>>>>>>>>>>> 连接服务器失败")
		return
	}
	fmt.Println(">>>>>>>>>>>> 连接服务器成功")
	select {
	}

}
```

![image-20210602092649762](http://cdn.noteblogs.cn/image-20210602092649762.png)

v2版本：使用命令行

```go
var ServerIp string
var ServerPort int

//main函数之前执行init函数
func init()  {
	flag.StringVar(&ServerIp, "ip", "127.0.0.1","设置服务器的ip地址 默认为127.0.0.1")
	flag.IntVar(&ServerPort, "port", 8890, "设置服务器的端口，默认端口为8890")
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
	select {
	}

}
```



![image-20210602093246823](http://cdn.noteblogs.cn/image-20210602093246823.png)

v3版本：菜单显示

```go
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
			break
		case 2:
			fmt.Println("私聊模式")
			break
		case 3:
			fmt.Println("更改用户名")
			break
			
		}
	}
}
```

v4版本：修改用户名

需要使用一个修改名字的方法，然后开启一个go程用于接收消息

```go
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

//接收serve端的消息的go程
func (client *Client) DealMessage() {
	io.Copy(os.Stdin, client.conn)
}
```



![image-20210602100019702](http://cdn.noteblogs.cn/image-20210602100019702.png)

v5版本：公聊模式

```go
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
```

v6：私聊模式

```go
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
```

