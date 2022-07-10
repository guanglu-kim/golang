package server

import (
	"center/util"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

var nodeHost = "http://192.168.1.205:3405/"

type registerData struct {
	id   string
	name string
}

func (reg *registerData) fromJson(json map[string]string) {
	reg.id = json["id"]
	reg.name = json["name"]
}

type fromData struct {
	id   string
	name string
}
type groupData struct {
	id   string
	name string
}
type userData struct {
	id   string
	name string
}
type toData struct {
	group groupData
	user  userData
}
type bodyData struct {
	Type    string
	content map[string]interface{}
}
type MessageData struct {
	from fromData
	to   toData
	body bodyData
}

func MapToJson(param map[string]interface{}) string {
	dataType, _ := json.Marshal(param)
	dataString := string(dataType)
	return dataString
}

func (msg *MessageData) fromJson(json map[string]interface{}) {
	from := json["from"].(map[string]interface{})
	to := json["to"].(map[string]interface{})
	group := to["group"].(map[string]interface{})
	body := json["body"].(map[string]interface{})
	msg.from = fromData{
		id:   from["id"].(string),
		name: from["name"].(string),
	}
	msg.to = toData{
		group: groupData{
			id:   group["id"].(string),
			name: group["name"].(string),
		},
	}
	msgtype := body["type"].(string) + "Content"

	msg.body = bodyData{
		Type:    body["type"].(string),
		content: body[msgtype].(map[string]interface{}),
	}
}

func (msg *MessageData) toJson() map[string]interface{} {
	jsondata := map[string]interface{}{}
	jsondata["from"] = msg.from
	jsondata["to"] = msg.to
	jsondata["body"] = msg.body
	return jsondata
}

type UserInfo struct {
	id       string
	name     string
	groups   []string
	subPoint bool
	conn     *WebSocketConn
}

var userlist = make([]UserInfo, 0)

type SFUServerConfig struct {
	Host          string
	Port          int
	CertFile      string
	KeyFile       string
	HTMLRoot      string
	WebSocketPath string
	NodeHost      string
}

func reguser(user UserInfo) {
	fmt.Println("muqian：：", userlist)
	for index, value := range userlist {
		if user.id == value.id {
			fmt.Println("重复：：", user.id)
			userlist[index] = user
			return
		}
	}

	userlist = append(userlist, user)
}

func DefaultConfig() SFUServerConfig {
	return SFUServerConfig{
		Host:          "0.0.0.0",
		Port:          8000,
		HTMLRoot:      "html",
		WebSocketPath: "/ws",
	}
}

type SFUServer struct {
	upgrader websocket.Upgrader
}

func NewSFUServer() *SFUServer {
	var server = &SFUServer{}
	server.upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	return server
}

func (server *SFUServer) handleWebSocketRequest(writer http.ResponseWriter, request *http.Request) {
	util.Infof("Handle Request")
	responseHeader := http.Header{}

	socket, err := server.upgrader.Upgrade(writer, request, responseHeader)
	if err != nil {
		util.Panicf("%v", err)
	}
	ws := NewWebSocketConn(socket)
	server.handleWebSocket(ws, request)
	ws.Read()
}

func Transformation(response *http.Response) map[string]interface{} {
	var result map[string]interface{}
	body, err := ioutil.ReadAll(response.Body)
	if err == nil {
		json.Unmarshal([]byte(string(body)), &result)
	}
	return result
}

func userRegist(id string, conn *WebSocketConn) {
	fmt.Println(id)
	response, err := http.Get(nodeHost + "account/" + id)
	if err != nil {
		util.Panicf("%v", err)
	}

	data := Transformation(response)["data"].(map[string]interface{})
	if data["groups"] == nil {
		user := UserInfo{
			id:     data["id"].(string),
			name:   data["name"].(string),
			groups: make([]string, 0),
			conn:   conn,
		}
		fmt.Println(user)
		reguser(user)
	} else {
		groups := data["groups"].([]interface{})
		arr := make([]string, 1)
		for _, value := range groups {
			valuea := value.(map[string]interface{})
			arr = append(arr, valuea["id"].(string))
		}
		user := UserInfo{
			id:     data["id"].(string),
			name:   data["name"].(string),
			groups: arr,
			conn:   conn,
		}
		fmt.Println(user)
		reguser(user)
	}

}

func sendToAll(msg map[string]interface{}, excptId string) {
	fmt.Println("kkkkkkkkkkkkkkkkkkk")
	fmt.Println(userlist)
	//userlist[0].conn.Send("ssssssssssssss")
	for _, user := range userlist {
		if user.conn != nil {
			user.conn.Send(MapToJson(msg))
		}

	}
}

func sendToGroup(groupId string, msg map[string]interface{}, excptId string) {

	for _, user := range userlist {
		for _, group := range user.groups {
			if group == groupId {
				println("msgmsgmsgmsgmsg", user.name)
				user.conn.Send(MapToJson(msg))
			}
		}

	}
	//response, err := http.Get("http://192.168.1.205:3405/group/" + groupId)
	//if err != nil {
	//	util.Panicf("%v", err)
	//} else {
	//	data := Transformation(response)["data"].(map[string]interface{})
	//	members := data["members"].([]interface{})
	//
	//	ids := make([]string, 0)
	//	for _, value := range members {
	//		value := value.(map[string]interface{})
	//		ids = append(ids, value["id"].(string))
	//		// todo: 是否需要排除发送者
	//
	//
	//	}
	// 写入数据库
	http.PostForm(nodeHost+"msgrecode/", url.Values{"data": {MapToJson(msg)}})

	//for index, value := range userlist {
	//	fmt.Printf("index:%d,value:%d\n", index, value)
	//}
}

func JsonToMap(jsonStr string) (map[string]string, error) {
	m := make(map[string]string)
	err := json.Unmarshal([]byte(jsonStr), &m)
	if err != nil {
		fmt.Printf("Unmarshal with error: %+v\n", err)
		return nil, err
	}

	for k, v := range m {
		fmt.Printf("%v: %v\n", k, v)
	}

	return m, nil
}

func createNewGroup(group string) bool {
	// 写入数据库
	response, err := http.PostForm(nodeHost+"group/bySocket", url.Values{"data": {group}})
	defer response.Body.Close() //在回复后必须关闭回复的主体
	body, err := ioutil.ReadAll(response.Body)
	if err == nil {
		fmt.Println(string(body))
		if find := strings.Contains(string(body), "\"success\":true"); find {
			return true
		}

	}
	return false

}

func filterUser() {
	newuserlist := make([]UserInfo, 0)
	for _, value := range userlist {
		if value.conn != nil {
			newuserlist = append(newuserlist, value)
		}
	}
	userlist = newuserlist
}

func findAndRegister(conn *WebSocketConn, Type string) {
	for index, value := range userlist {
		if value.conn == conn {
			if Type == "subPoint" {
				userlist[index].subPoint = true
				conn.Send(MapToJson(map[string]interface{}{
					"type": "subPointRes",
					"data": map[string]interface{}{
						"success": true,
						"info":    "注册获取坐标信令成功",
					},
				}))
				return
			}
		}
	}
	conn.Send(MapToJson(map[string]interface{}{
		"type": "subPointRes",
		"data": map[string]interface{}{
			"success": false,
			"info":    "注册获取坐标信令失败",
		},
	}))
}

func findAndpubPoint(conn *WebSocketConn, msg string) {
	for _, value := range userlist {
		if value.subPoint {
			fmt.Println(value.name + "注册了point，发送")
			value.conn.Send(msg)
		}
	}
}

func (server *SFUServer) handleWebSocket(conn *WebSocketConn, request *http.Request) {
	//
	//conlist = append(conlist, ConInfo{
	//	id:      query["userId"][0],
	//	groupId: gid,
	//	conn:    conn,
	//})
	//util.HttpGet("group/group002", map[string]string{"a": "1"})
	conn.On("message", func(message []byte) {
		util.Debugf(string(message))
		fmt.Println(userlist)
		request, err := util.Unmarshal(string(message))
		if err != nil {
			util.Errorf("解析Json数据Unmarshal错误 %v", err)
			return
		}

		switch request["type"] {
		case "bgyx":
			//room := room.NewRoom("bgyx001")
			//user := finduser(conn)
			//room.AddUser(user)
		case "ces":
			conn.Send(MapToJson(map[string]interface{}{"type": "ces", "data": "崔晨"}))
		case "subPoint":
			findAndRegister(conn, "subPoint")
		case "pubPoint":
			//data := request["data"].(map[string]interface{})
			findAndpubPoint(conn, MapToJson(request))
		case "register":
			util.Warnf("消息请求 %v", "注册")
			fmt.Println(request["data"])
			data := request["data"].(map[string]interface{})

			user := UserInfo{
				id:   data["id"].(string),
				name: data["name"].(string),
				conn: conn,
			}

			conn.Send(MapToJson(map[string]interface{}{"type": "registerRes", "data": map[string]interface{}{
				"success": true,
				"info":    user.name + "注册成功，您已接入信令服务通讯",
			}}))

			userRegist(data["id"].(string), conn)
			//reguser(user)
			fmt.Println(userlist)
		case "message":
			util.Warnf("消息请求 %T", request["body"])
			content := request["data"].(map[string]interface{})
			fmt.Println(content)
			to := content["to"].(map[string]interface{})
			group := to["group"].(map[string]interface{})
			groupId := group["id"].(string)
			fmt.Println(groupId)
			fmt.Println(request["data"])
			sendToGroup(groupId, request, "")
		case "createGroup":
			util.Warnf("新建群 %v", request)
			content := request["data"].(map[string]interface{})
			fmt.Println(content)

			group := content["group"].(map[string]interface{})
			//groupId := group["id"].(string)
			groupName := group["name"].(string)
			members := group["members"]
			fmt.Println(groupName, members)
			fmt.Println(request["data"])
			success := createNewGroup(MapToJson(content))
			if success {
				conn.Send(MapToJson(map[string]interface{}{
					"type": "createGroupSuccess",
					"data": group,
				}))
				sendToAll(map[string]interface{}{
					"type": "createGroup",
					"data": group,
				}, "")
			}
			//
		case "system":
			util.Warnf("消息请求SYS %T", request["body"])
			content := request["data"].(map[string]interface{})
			fmt.Println(content)
			to := content["to"].(map[string]interface{})
			group := to["group"].(map[string]interface{})
			groupId := group["id"].(string)
			fmt.Println(groupId)
			fmt.Println(request["data"])
			sendToGroup(groupId, request, "")
		default:
			{
				util.Warnf("未知的请求 %v", request)
			}
			break
		}

	})

	conn.On("close", func(code int, text string) {
		util.Infof("连接关闭 %v", conn)
		conn.Send(MapToJson(map[string]interface{}{"body": "您已退出，断开链接"}))
		newuserlist := make([]UserInfo, 1)
		user := UserInfo{id: "def"}
		for _, value := range userlist {
			if value.conn == conn {
				fmt.Println("ddddddddddddddddddddddddddddd")
				user = value
				break
			}
		}
		fmt.Println("FINDFIND", user)
		for _, value := range userlist {
			if value.conn != conn {
				if value.conn != nil {
					newuserlist = append(newuserlist, value)
				}
			}
		}
		fmt.Println(newuserlist)
		userlist = newuserlist
		if user.id != "def" {
			sendToAll(map[string]interface{}{"type": "userLogout", "data": map[string]interface{}{
				"user": map[string]interface{}{
					"id":   user.id,
					"name": user.name,
				},
				"text": user.name + "退出",
			}}, "")
		}
	})
}

func (server *SFUServer) Bind(cfg SFUServerConfig) {
	http.HandleFunc(cfg.WebSocketPath, server.handleWebSocketRequest)
	http.Handle("/", http.FileServer(http.Dir(cfg.HTMLRoot)))
	util.Infof("SFU Server listening on: %s:%d", cfg.Host, cfg.Port)
	nodeHost = cfg.NodeHost
	panic(http.ListenAndServeTLS(cfg.Host+":"+strconv.Itoa(cfg.Port), cfg.CertFile, cfg.KeyFile, nil))
	//panic(http.ListenAndServe(cfg.Host+":"+strconv.Itoa(cfg.Port), nil))
}
