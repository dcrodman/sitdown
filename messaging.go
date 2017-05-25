package main

import (
	"encoding/json"
	"fmt"
	"github.com/pubnub/go/messaging"
	// "reflect"
	"strconv"
	"strings"
)

type Command string

type Message struct {
	Action Command
	Id     string
	IPAddr string
}

const (
	Move      Command = "move"
	SetHeight Command = "set"

	sitdownChannel = "sitdown"
	subscribeKey   = "sub-c-04ca627a-4008-11e7-86e2-02ee2ddab7fe"
	publishKey     = "pub-c-b92ac3f8-47e1-4965-a9d0-1f6b2e8b7847"

	commandClientId = "command-client"
)

var pubnub = messaging.NewPubnub(publishKey, subscribeKey, "", "", true, "", nil)

func PublishCommand(command Command, id string, ip string) {
	successChan := make(chan []byte)
	errorChan := make(chan []byte)

	cmd := &Message{
		Action: command,
		Id:     id,
		IPAddr: ip,
	}

	jsonCmd, _ := json.Marshal(cmd)
	pubnub.Publish(sitdownChannel, string(jsonCmd), successChan, errorChan)

	select {
	case <-successChan:
		fmt.Println("Publishing command: " + command)
	case err := <-errorChan:
		fmt.Println("Error publishing command " + string(err))
	}
}

func StartAnnouncing() {
	go func() {
		// pubnub.Publish(sitdownChannel, command, successChan, errorChan)

		// for {
		// 	timer := time.NewTimer(1 * time.Minute)
		// 	select {
		// 	case <-timer.C:
		// 		pubnub.Publish(sitdownChannel, command, successChan, errorChan)
		// 	}
		// }
	}()
}

func SubscribeToChannel() {
	successChan := make(chan []byte)
	errorChan := make(chan []byte)

	fmt.Println("Subscribing to " + sitdownChannel)
	go pubnub.Subscribe(sitdownChannel, "", successChan, false, errorChan)

	go func() {
		for {
			select {
			case response := <-successChan:
				var msg []interface{}
				if err := json.Unmarshal(response, &msg); err != nil {
					fmt.Println("Could not process command: " + err.Error())
				}

				fmt.Printf("Received message: %v\n", msg)

				switch msg[0].(type) {
				case []interface{}:
					encoded := msg[0].([]interface{})[0].(string)
					var message Message
					json.Unmarshal([]byte(encoded), &message)

					fmt.Printf("Received command: %v\n", message)
					handleCommand(message)
				default:
				}
			case err := <-errorChan:
				fmt.Println("Received message on error channel: " + string(err))
			}
		}
	}()
}

func handleCommand(message Message) {
	splitCommand := strings.Split(string(message.Action), " ")
	switch Command(splitCommand[0]) {
	case Move:
		switch len := len(splitCommand); len {
		case 1:
			fmt.Println("Missing direction from move command (skipping)")
		case 2:
			move(splitCommand[1], 1000)
		default:
			duration, _ := strconv.ParseInt(splitCommand[2], 10, 32)
			move(splitCommand[1], int(duration))
		}
	case SetHeight:
		if len := len(splitCommand); len <= 1 {
			fmt.Println("Missing height for set command (skipping)")
		}
		setHeight(splitCommand[1])
	default:
		fmt.Printf("Unrecognized command %v; skipping\n", message.Action)
	}
}
