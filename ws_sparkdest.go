package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/url"
	"strings"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Client struct {
	ID            string
	Connection    *websocket.Conn
	LastHeartbeat time.Time
}

var (
	hostUrl   = "ws://spark-api.xf-yun.com/v3.1/chat"
	appid     = ""
	apiSecret = ""
	apiKey    = ""

)

func Request(client *Client,ask string) string {
	// fmt.Println(HmacWithShaTobase64("hmac-sha256", "hello\nhello", "hello"))
	// st := time.Now()
	d := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}
	//握手并建立websocket 连接
	conn, resp, err := d.Dial(assembleAuthUrl1(hostUrl, apiKey, apiSecret), nil)
	if err != nil {
		panic(readResp(resp) + err.Error())
		return "error"
	} else if resp.StatusCode != 101 {
		panic(readResp(resp) + err.Error())
	}

	go func() {

		data := genParams1(appid, ask)
		conn.WriteJSON(data)

	}()

	var answer = ""
	//获取返回的数据
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("read message error:", err)
			break
		}

		var data map[string]interface{}
		err1 := json.Unmarshal(msg, &data)
		if err1 != nil {
			fmt.Println("Error parsing JSON:", err)
			return "error"
		}
		fmt.Println(string(msg))
		//解析数据
		payload := data["payload"].(map[string]interface{})
		choices := payload["choices"].(map[string]interface{})
		header := data["header"].(map[string]interface{})
		code := header["code"].(float64)

		if code != 0 {
			fmt.Println(data["payload"])
			return "error"
		}
		status := choices["status"].(float64)
		fmt.Println(status)
		text := choices["text"].([]interface{})
		content := text[0].(map[string]interface{})["content"].(string)
		err = client.Connection.WriteMessage(websocket.TextMessage, []byte(content))
		// if err != nil {
		// 	log.Println(err)
		// 	client.Connection.Close()
		// 	delete(clients, client.Connection)
		// 	break
		// }
		if status != 2 {
			answer += content
		} else {
			fmt.Println("收到最终结果")
			answer += content
			usage := payload["usage"].(map[string]interface{})
			temp := usage["text"].(map[string]interface{})
			totalTokens := temp["total_tokens"].(float64)
 			fmt.Println("total_tokens:", totalTokens)
			conn.Close()
			break
		}

	}
	//输出返回结果
	fmt.Println(answer)
	return answer
	//time.Sleep(1 * time.Second)
}

// 生成参数
func genParams1(appid, question string) map[string]interface{} { // 根据实际情况修改返回的数据结构和字段名


	messages := []Message{
		{Role: "user", Content: question},
	}

	data := map[string]interface{}{ // 根据实际情况修改返回的数据结构和字段名
		"header": map[string]interface{}{ // 根据实际情况修改返回的数据结构和字段名
			"app_id": appid, // 根据实际情况修改返回的数据结构和字段名
		},
		"parameter": map[string]interface{}{ // 根据实际情况修改返回的数据结构和字段名
			"chat": map[string]interface{}{ // 根据实际情况修改返回的数据结构和字段名
				"domain":      "generalv3",    // 根据实际情况修改返回的数据结构和字段名
				"temperature": float64(0.8), // 根据实际情况修改返回的数据结构和字段名
				"top_k":       int64(6),     // 根据实际情况修改返回的数据结构和字段名
				"max_tokens":  int64(2048),  // 根据实际情况修改返回的数据结构和字段名
				"auditing":    "default",    // 根据实际情况修改返回的数据结构和字段名
			},
		},
		"payload": map[string]interface{}{ // 根据实际情况修改返回的数据结构和字段名
			"message": map[string]interface{}{ // 根据实际情况修改返回的数据结构和字段名
				"text": messages, // 根据实际情况修改返回的数据结构和字段名
			},
		},
	}
	return data // 根据实际情况修改返回的数据结构和字段名
}

// 创建鉴权url  apikey 即 hmac username
func assembleAuthUrl1(hosturl string, apiKey, apiSecret string) string {
	ul, err := url.Parse(hosturl)
	if err != nil {
		fmt.Println(err)
	}
	//签名时间
	date := time.Now().UTC().Format(time.RFC1123)
	//date = "Tue, 28 May 2019 09:10:42 MST"
	//参与签名的字段 host ,date, request-line
	signString := []string{"host: " + ul.Host, "date: " + date, "GET " + ul.Path + " HTTP/1.1"}
	//拼接签名字符串
	sgin := strings.Join(signString, "\n")
	// fmt.Println(sgin)
	//签名结果
	sha := HmacWithShaTobase64("hmac-sha256", sgin, apiSecret)
	// fmt.Println(sha)
	//构建请求参数 此时不需要urlencoding
	authUrl := fmt.Sprintf("hmac username=\"%s\", algorithm=\"%s\", headers=\"%s\", signature=\"%s\"", apiKey,
		"hmac-sha256", "host date request-line", sha)
	//将请求参数使用base64编码
	authorization := base64.StdEncoding.EncodeToString([]byte(authUrl))

	v := url.Values{}
	v.Add("host", ul.Host)
	v.Add("date", date)
	v.Add("authorization", authorization)
	//将编码后的字符串url encode后添加到url后面
	callurl := hosturl + "?" + v.Encode()
	return callurl
}

func HmacWithShaTobase64(algorithm, data, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(data))
	encodeData := mac.Sum(nil)
	return base64.StdEncoding.EncodeToString(encodeData)
}

func readResp(resp *http.Response) string {
	if resp == nil {
		return ""
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("code=%d,body=%s", resp.StatusCode, string(b))
}


type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func main() {
	clients := make(map[*websocket.Conn]*Client)
	clientID := 1

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Upgrade error:", err)
			return
		}

		// 检查连接数是否已达到最大值
		if len(clients) >= 10 {
			conn.Close()
			return
		}

		// 分配客户端ID
		id := fmt.Sprintf("client-%d", clientID)
		clientID++

		// 创建客户端对象
		client := &Client{
			ID:            id,
			Connection:    conn,
			LastHeartbeat: time.Now(),
		}

		// 添加到客户端集合
		clients[conn] = client

		// 处理接收消息
		go handleMessages(client, clients)

		// 处理心跳检测
		go checkHeartbeat(client, clients)
	})

	log.Println("WebSocket server is running on port 3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}

func handleMessages(client *Client, clients map[*websocket.Conn]*Client) {
	for {
		_, message, err := client.Connection.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)

			// 关闭连接并从客户端集合中删除
			client.Connection.Close()
			delete(clients, client.Connection)
			break
		}

		log.Printf("Received message from client %s: %s\n", client.ID, string(message))
		Request(client,string(message))
	}
}

func checkHeartbeat(client *Client, clients map[*websocket.Conn]*Client) {
	for {
		time.Sleep(30 * time.Second)

		// 检查最后心跳时间
		/*if time.Since(client.LastHeartbeat) > 30*time.Second {
			log.Printf("Closing connection for client %s due to inactivity\n", client.ID)

			// 关闭连接并从客户端集合中删除
			client.Connection.Close()
			delete(clients, client.Connection)
			break
		}*/
	}
}


