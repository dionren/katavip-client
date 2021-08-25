package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"flag"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

var (
	port   = flag.String("p", "", "port")
	engine = flag.String("e", "", "engine uuid36 or uuid32")
	secret = flag.String("s", "", "secret")
	accessKey = flag.String("k", "", "access key")
	host   = "workers.katago.vip:"
)

func main() {
	flag.Parse()

	println("katavip-client v1.0.3")
	println("https://github.com/dionren/katavip-client")

	var wsUrl string

	if *port != "" && *engine != "" && *secret != "" {
		var engineUuid36 string
		if (strings.Count(*engine, "") - 1) == 32 {
			engineUuid36 = (*engine)[0:8] + "-" + (*engine)[8:12] + "-" + (*engine)[12:16] + "-" + (*engine)[16:20] + "-" + (*engine)[20:]
		} else {
			engineUuid36 = *engine
		}
		wsUrl = "ws://" + host + *port + "/engine/operator/" + engineUuid36 + "/" + *secret
	} else if *accessKey != "" {
		resp, err := http.Get("https://api.katavip.cn/api/v1/katavip/go/ws/operator?accessKey=" + *accessKey)
		if err != nil {
			println("https://api.katavip.cn offline temporary")
			return
		}
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			println("https://api.katavip.cn offline temporary")
			return
		}
		if resp.StatusCode != 200 {
			println(string(b))
			return
		}
		wsUrl = string(b)
	} else {
		println("params error, please check.")
		return
	}

	println(wsUrl)

	ws, _, err := websocket.DefaultDialer.Dial(wsUrl, nil)

	if err != nil {
		println("WebSocket connected error.")
		return
	} else {
		// 从WSS接收数据
		go func() {
			var msgServer MsgServer
			for {
				_, data, err := ws.ReadMessage()
				if err != nil {
					log.Fatal(err)
				}

				err = json.Unmarshal(data, &msgServer)
				if err != nil {
					return
				}

				if msgServer.Zip == 1 {
					msgServer.Str, _ = unGzipBase64(msgServer.Str)
				}

				_, err = os.Stdout.WriteString(msgServer.Str)
				if err != nil {
					return
				}
			}
		}()
	}

	var msgClient MsgClient

	// 发送压缩指令
	msgClient.Category = "ext"
	msgClient.Cmd = "zip"
	payload, _ := json.Marshal(msgClient)
	err = ws.WriteMessage(websocket.TextMessage, payload)
	if err != nil {
		log.Fatal(err)
	}

	// 从STD不断的轮询输入数据并通过WSS发送至服务器
	reader := bufio.NewReader(os.Stdin)
	for {
		byteArray, _, _ := reader.ReadLine()
		msgClient.Cmd = string(byteArray)
		msgClient.Category = "gtp"

		if msgClient.Cmd == "quit" {
			break
		}

		payload, _ := json.Marshal(msgClient)
		err = ws.WriteMessage(websocket.TextMessage, payload)
		if err != nil {
			log.Fatal(err)
		}
	}

	err = ws.Close()
	if err != nil {
		log.Fatal(err)
	}
}

func unGzip(in []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(in))

	if err != nil {
		var out []byte
		return out, err
	}

	defer reader.Close()
	return ioutil.ReadAll(reader)
}

// 从Base64数据解析成byte数组，再解压缩，再转换成字符串
func unGzipBase64(in string) (string, error) {
	bytesOut, err := base64.StdEncoding.DecodeString(in)
	if err != nil {
		return "", err
	}

	bytesUnGzip, err := unGzip(bytesOut)
	if err != nil {
		return "", err
	}

	return string(bytesUnGzip), nil
}

type MsgServer struct {
	Code     int    `json:"code,omitempty"`
	Zip      int    `json:"zip,omitempty"`
	Category string `json:"category,omitempty"`
	Str      string `json:"str,omitempty"`
	Game     string `json:"game,omitempty"`
}

type MsgClient struct {
	Category string `json:"category"`
	Cmd      string `json:"cmd"`
}
