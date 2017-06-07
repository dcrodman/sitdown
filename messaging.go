package main

import (
	"encoding/json"
	"github.com/pubnub/go/messaging"
	"net"
	"strings"
	"time"
)

type Command string

// Possible commands that can be sent to (or from) a controller.
const (
	// Move the desk up. Syntax: move TARGET (up|down) [duration ms]
	Move Command = "move"
	// Set the desk to a particular height. Syntax: set TARGET HEIGHT
	Set Command = "set"
	// BellToll will cause the Pi to adjust up/down on the hour. Syntax: belltoll TARGET (enable|disable).
	BellToll Command = "belltoll"
	// FixHeight will cause a desk to reset to the specified height when changed (after a small delay).
	// Syntax: fixheight TARGET (enable|disable)
	FixHeight Command = "fixheight"
	// Announce is an internal command used for discovery purposes.
	Announce Command = "announce"
)

const (
	CommandClientId = "command-client"
	sitdownChannel  = "controller"
)

type Message struct {
	// Action that the recipient should perform.
	Action Command
	// Parameters for the Action.
	Params []string
	// ID of the sender.
	ID string
	// IP Address of the sender (usually only for announce).
	IPAddr string
	// ID of the intended recipient.
	TargetID string
}

type Messenger struct {
	pubnub *messaging.Pubnub
}

func (m *Messenger) Initialize() {
	m.pubnub = messaging.NewPubnub(controller.PubKey, controller.SubKey, "", "", true, "", nil)
	m.pubnub.SetUUID(controller.ID)
}

func (m *Messenger) Cleanup() {
	successChan := make(chan []byte)
	errorChan := make(chan []byte)

	go m.pubnub.Unsubscribe(sitdownChannel, successChan, errorChan)
	select {
	case <-successChan:
		logger.Println("Unsubscribed from channel")
	case err := <-errorChan:
		logger.Println("Failed to unsubscribe from channel: " + string(err))
	case <-messaging.Timeout():
		logger.Println("Timeout while unsubcribing from channel")
	}
}

// Kick off a goroutine that will write a message to the channel with some basic
// info about the device for discovery by other controllers and the command client.
func (m Messenger) StartAnnouncing() {
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
		m.Publish(Announce, ipAddress, "all", nil)

		for {
			timer := time.NewTimer(1 * time.Minute)
			select {
			case <-timer.C:
				m.Publish(Announce, ipAddress, "all", nil)
			}
		}
	}()
}

// Subscribe to the PubNub channel and decode messages as they come in.
// Valid messages will be passed to handlerFn with the full Message struct.
func (m Messenger) StartSubscriber(handlerFn func(Message)) {
	successChan := make(chan []byte)
	errorChan := make(chan []byte)

	logger.Println("Subscribing to " + sitdownChannel)
	go m.pubnub.Subscribe(sitdownChannel, "", successChan, false, errorChan)

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

					targetID := strings.ToLower(message.TargetID)

					// Throw out messages sent from the same device or that
					// are directed to another device.
					if message.ID != controller.ID &&
						(targetID == "all" || targetID == controller.ID) {
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

// Write a message to our channel on PubNub.
func (m Messenger) Publish(command Command, sourceIP string, targetID string, params []string) {
	successChan := make(chan []byte)
	errorChan := make(chan []byte)

	cmd := &Message{
		Action:   command,
		Params:   params,
		ID:       controller.ID,
		IPAddr:   sourceIP,
		TargetID: targetID,
	}

	jsonCmd, _ := json.Marshal(cmd)
	m.pubnub.Publish(sitdownChannel, string(jsonCmd), successChan, errorChan)

	select {
	case <-successChan:
		logger.Printf("Publishing command: %+v\n", cmd)
	case err := <-errorChan:
		logger.Println("Error publishing command " + string(err))
	}
}
