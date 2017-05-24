package main

import (
	"bufio"
	"fmt"
	"github.com/pubnub/go/messaging"
	"os"
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
			case msg := <-successChan:
				fmt.Println("Received message on success channel: " + string(msg))
			case err := <-errorChan:
				fmt.Println("Received message on error channel: " + string(err))
			}
		}
	}()
}
