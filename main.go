package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"github.com/gorilla/websocket"
	"log"
	"os"
	"strings"
)

var (
	port   = flag.String("p", "", "port")
	engine = flag.String("e", "", "engine uuid36 or uuid32")
	secret = flag.String("s", "", "secret")
	host   = "workers.katago.vip:"
)

func main() {
	flag.Parse()

	println("katavip-client v1.0.1")
	println("https://github.com/dionren/katavip-client")

	if *port == "" || *engine == "" || *secret == "" {
		println("params error, please check.")
		return
	}

	var engineUuid36 string
	if (strings.Count(*engine, "") - 1) == 32 {
		engineUuid36 = (*engine)[0:8] + "-" + (*engine)[8:12] + "-" + (*engine)[12:16] + "-" + (*engine)[16:20] + "-" + (*engine)[20:]
	} else {
		engineUuid36 = *engine
	}

	wsUrl := "ws://" + host + *port + "/engine/operator/" + engineUuid36 + "/" + *secret

	println(wsUrl)

	ws, _, err := websocket.DefaultDialer.Dial(wsUrl, nil)

	if err != nil {
		println("WebSocket connected error.")
		return
	} else {
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

				_, err = os.Stdout.WriteString(msgServer.Str)
				if err != nil {
					return
				}
			}
		}()
	}

	var msgClient MsgClient
	reader := bufio.NewReader(os.Stdin)
	for {
		bytes, _, _ := reader.ReadLine()
		msgClient.Cmd = string(bytes)
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

type MsgServer struct {
	Code     int    `json:"code,omitempty"`
	Category string `json:"category,omitempty"`
	Str      string `json:"str,omitempty"`
	Game     string `json:"game,omitempty"`
}

type MsgClient struct {
	Category string `json:"category"`
	Cmd      string `json:"cmd"`
}
