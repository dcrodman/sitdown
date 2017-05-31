package main

import (
	"encoding/json"
	"github.com/pubnub/go/messaging"
	"net"
	"time"
)

type Command string

type Message struct {
	Action Command
	Id     string
	IPAddr string
}

const (
	// Possible commands that can be sent to (or by) the desk controllers.
	Move      Command = "move"
	SetHeight Command = "set"
	Announce  Command = "announce"

	sitdownChannel = "sitdown"
	// Demo keys that we don't really care about.
	subscribeKey = "sub-c-04ca627a-4008-11e7-86e2-02ee2ddab7fe"
	publishKey   = "pub-c-b92ac3f8-47e1-4965-a9d0-1f6b2e8b7847"

	CommandClientId = "command-client"
)

var pubnub = messaging.NewPubnub(publishKey, subscribeKey, "", "", true, "", nil)

// Write a message to our channel on PubNub.
func PublishCommand(command Command, ip string) {
	successChan := make(chan []byte)
	errorChan := make(chan []byte)

	cmd := &Message{
		Action: command,
		Id:     config.ControllerID,
		IPAddr: ip,
	}

	jsonCmd, _ := json.Marshal(cmd)
	pubnub.Publish(sitdownChannel, string(jsonCmd), successChan, errorChan)

	select {
	case <-successChan:
		logger.Println("Publishing command: " + command)
	case err := <-errorChan:
		logger.Println("Error publishing command " + string(err))
	}
}

// Kick off a goroutine that will write a message to the channel with some basic
// info about the device for discovery by other controllers and the command client.
func StartAnnouncing() {
	go func() {
		interfaces, err := net.Interfaces()
		if err != nil {
			logger.Println("ERROR: Could not announce IP address " + err.Error())
			return
		}

		var ipAddress string
		for _, iface := range interfaces {
			// Interface probably specific to the Raspberry Pi, but eh.
			if iface.Name == "wlan0" {
				addrs, err := iface.Addrs()
				if err != nil {
					logger.Printf("Could not retrieve Addrs " + err.Error())
					continue
				}

			iploop:
				for _, addr := range addrs {
					switch t := addr.(type) {
					case *net.IPNet:
						if ip4 := addr.(*net.IPNet).IP.To4(); ip4 != nil {
							ipAddress = ip4.String()
							break iploop
						}
					default:
						logger.Printf("Found Addr of type %#v\n", t)
					}
				}
			}
		}

		logger.Println("Announcing IP address: " + ipAddress)
		PublishCommand(Announce, ipAddress)

		for {
			timer := time.NewTimer(1 * time.Minute)
			select {
			case <-timer.C:
				PublishCommand(Announce, ipAddress)
			}
		}
	}()
}

// Subscribe to the PubNub channel and decode messages as they come in.
// Valid messages will be passed to handlerFn with the full Message struct.
func StartSubscriber(handlerFn func(Message)) {
	successChan := make(chan []byte)
	errorChan := make(chan []byte)

	logger.Println("Subscribing to " + sitdownChannel)
	go pubnub.Subscribe(sitdownChannel, "", successChan, false, errorChan)

	go func() {
		for {
			select {
			case response := <-successChan:
				var msg []interface{}
				if err := json.Unmarshal(response, &msg); err != nil {
					logger.Println("Could not process command: " + err.Error())
				}

				switch msg[0].(type) {
				case []interface{}:
					encoded := msg[0].([]interface{})[0].(string)
					var message Message
					json.Unmarshal([]byte(encoded), &message)

					// Throw out messages sent from the same device.
					if message.Id != config.ControllerID {
						logger.Printf("Received command: %#v\n", message)
						handlerFn(message)
					}
				default:
					logger.Printf("Ignoring message: %v\n", msg)

				}
			case err := <-errorChan:
				logger.Println("Received message on error channel: " + string(err))
			}
		}
	}()
}
