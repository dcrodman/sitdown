package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/pubnub/go/messaging"
	"os"
	"reflect"
	"strconv"
	"strings"
)

const (
	sitdownChannel = "sitdown"
	subscribeKey   = "sub-c-04ca627a-4008-11e7-86e2-02ee2ddab7fe"
	publishKey     = "pub-c-b92ac3f8-47e1-4965-a9d0-1f6b2e8b7847"
)

func EnterCommandMode() {
	fmt.Println("Entering Command Mode")
	pubnub := messaging.NewPubnub(publishKey, subscribeKey, "", "", true, "", nil)

	successChan := make(chan []byte)
	errorChan := make(chan []byte)
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Command: ")
		command, _ := reader.ReadString('\n')
		command = strings.Trim(command, "\n ")
		if strings.ToLower(command) == "exit" {
			break
		}

		pubnub.Publish(sitdownChannel, command, successChan, errorChan)

		select {
		case <-successChan:
			fmt.Println("Publishing command: " + command)
		case err := <-errorChan:
			fmt.Println("Error publishing command " + string(err))
		}
	}
}

func PubNubSubscribe() {
	pubnub := messaging.NewPubnub("", subscribeKey, "", "", true, "", nil)

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

				var command string
				switch m := reflect.TypeOf(msg[0]).Kind(); m {
				case reflect.Slice:
					command = msg[0].([]interface{})[0].(string)
					fmt.Printf("Received command: %v\n", command)
					handleCommand(command)
				default:
					fmt.Printf("Received message on success channel: %v\n", msg)
				}
			case err := <-errorChan:
				fmt.Println("Received message on error channel: " + string(err))
			}
		}
	}()
}

func handleCommand(command string) {
	if strings.Contains(command, "move") {
		splitCommand := strings.Split(command, " ")
		switch len := len(splitCommand); len {
		case 1:
			fmt.Println("Missing direction from move command (skipping)")
		case 2:
			move(splitCommand[1], 1000)
		default:
			duration, _ := strconv.ParseInt(splitCommand[2], 10, 32)
			move(splitCommand[1], int(duration))
		}
		return
	}

	switch command {
	default:
		fmt.Printf("Unrecognized command %s; skipping\n", command)
	}
}
