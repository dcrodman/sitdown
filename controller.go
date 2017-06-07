package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

const configFilename = "controller.conf"

// Controller is the singleton for most of Sitdown's operation. It can be used as a
// central controller for other desks or the controller to do things with a specific desk.
type Controller struct {
	ID     string
	PubKey string
	SubKey string

	// Desk instance used to control the standing desk if running in control mode.
	desk *Desk
	// Map of the IDs of active desk controllers to their IP addresses.
	activeControllers map[string]string

	// Unbuffered channel specifically for killing bellToll mode.
	bellTollKill chan bool
	// Unbuffered channel specifically for killing fixed height mode.
	fixedHeightKill chan bool
}

func (c *Controller) InitFromConfig() {
	fileContents, err := ioutil.ReadFile(configFilename)
	if err != nil {
		fileContents, err = ioutil.ReadFile("/home/pi/" + configFilename)
		if err != nil {
			fmt.Printf("Unable to locate %s in local dir or /home/pi\n", configFilename)
			os.Exit(1)
		}
	}

	json.Unmarshal([]byte(fileContents), &c)
	logger.Printf("Initializing controller with ID: %s\n", c.ID)

	c.desk = new(Desk)
	c.activeControllers = make(map[string]string)
	c.bellTollKill = make(chan bool, 1)
}

// Command client mode for communicating with the desk controllers remotely. This is
// invoked with the -c command line argument from any machine. Does not have to be on
// the same network since all of the commands are passed through PubNub.
//
// list: Show all controllers that the command client is aware of
// exit: Kill the prompt
// Syntax for anything else (published to controllers): command TARGET [parameters]
func (c *Controller) EnterCommandMode() {
	fmt.Println("Entering Command Mode")
	logFile, err := os.Create("controller.log")
	if err != nil {
		fmt.Printf("Could not open controller.log")
		os.Exit(0)
	}
	// Reinitialize the logger from stdout so that we don't interfere with the prompt.
	logger = log.New(logFile, "", log.Ltime)
	c.ID = CommandClientId

	reader := bufio.NewReader(os.Stdin)
	messenger.StartSubscriber(c.handleCommandModeMessage)

loop:
	for {
		fmt.Print("Command: ")
		fullCommand, _ := reader.ReadString('\n')
		fullCommand = strings.Trim(fullCommand, "\n ")

		splitFullCommand := strings.Split(fullCommand, " ")
		action := splitFullCommand[0]

		switch strings.ToLower(action) {
		case "list":
			for id, ip := range c.activeControllers {
				logger.Printf("Controller @ %s (id: %s)]\n", ip, id)
			}
			continue
		case "exit":
			break loop
		}

		if len(splitFullCommand) < 2 {
			logger.Printf("Command is missing target; skipping")
			continue
		}

		target := splitFullCommand[1]
		if len(splitFullCommand) > 2 {
			messenger.Publish(Command(action), "", target, splitFullCommand[2:])
		} else {
			messenger.Publish(Command(action), "", target, nil)
		}
	}
	os.Exit(0)
}

// Command handler for messages received while in command mode.
func (c *Controller) handleCommandModeMessage(message Message) {
	splitCommand := strings.Split(string(message.Action), " ")
	switch Command(splitCommand[0]) {
	case Announce:
		logger.Printf("Discovered controller %s (id: %s)\n", message.IPAddr, message.ID)
		c.activeControllers[message.ID] = message.IPAddr
	}
}

// Server mode for processing requests to make a desk do funny things.
func (c *Controller) EnterDeskControlMode() {
	c.desk.Setup(logger)
	messenger.StartAnnouncing()
	messenger.StartSubscriber(c.handleDeskControllerMessage)
}

// Cleanup releases the GPIO resources for controlling the desk. Only needed for desk contol mode.
func (c Controller) Cleanup() {
	c.desk.Cleanup()
}

// Command handler that should be running on the actual desk controllers.
func (c *Controller) handleDeskControllerMessage(message Message) {
	switch Command(message.Action) {
	case Move:
		switch len(message.Params) {
		case 0:
			logger.Println("Missing parameters in Move command; skipping")
		case 1:
			c.Move(message.Params[0], 1000)
		default:
			duration, _ := strconv.ParseInt(message.Params[1], 10, 32)
			c.Move(message.Params[0], int(duration))
		}
	case Set:
		if len(message.Params) < 1 {
			logger.Println("Missing parameters in Set command; skipping")
		}
		c.SetHeight(message.Params[0])
	case BellToll:
		if len(message.Params) < 1 {
			logger.Println("Missing parameters in BellToll command; skipping")
		} else if message.Params[0] == "enable" {
			go c.EnableBellToll()
		} else {
			c.DisableBellToll()
		}
	case FixHeight:
		if len(message.Params) < 1 {
			logger.Println("Missing parameters in FixHeight command; skipping")
		} else if message.Params[0] == "enable" {
			go c.EnableFixedHeight()
		} else {
			c.desk.ResetListeners()
		}
	case Announce:
		logger.Printf("Discovered controller %s (id: %s)\n", message.IPAddr, message.ID)
		c.activeControllers[message.ID] = message.IPAddr
	default:
		logger.Printf("Unrecognized command %v; skipping\n", message.Action)
	}
}

func (c Controller) Move(direction string, time int) {
	logger.Printf("Moving desk %s for %d", direction, time)
	switch direction {
	case "up":
		c.desk.RaiseForDuration(time)
	case "down":
		c.desk.LowerForDuration(time)
	}
}

func (c Controller) SetHeight(height string) {
	logger.Println("Setting height to " + height)

	h, err := strconv.ParseFloat(height, 32)
	if err != nil || h < 28.1 || h > 47.5 {
		logger.Printf("Invalid height: %f\n", h)
		return
	}
	c.desk.ChangeToHeight(float32(h))
}

func (c *Controller) GetHeight() float32 {
	return c.desk.Height()
}

func (c Controller) EnableBellToll() {
	logger.Println("Enabling BellToll mode")
	// Start tolling at the next hour so the desk doesn't move immediately.
	// lastTolled := time.Now().Hour() % 12
loop:
	for {
		timer := time.NewTimer(10 * time.Second)
		select {
		case <-controller.bellTollKill:
			break loop
		case <-timer.C:
			thisHour := time.Now().Hour() % 12
			// if thisHour == 0 {
			// 	thisHour = 12
			// }

			// if thisHour != lastTolled {
			log.Printf("Belltoll - %d times", thisHour)
			for i := 0; i < thisHour; i++ {
				c.Move("up", 800)
				time.Sleep(time.Duration(1200) * time.Millisecond)
				c.Move("down", 850)
				time.Sleep(time.Duration(1200) * time.Millisecond)
			}
			// lastTolled = thisHour
			// }
		}
	}
}

func (c Controller) DisableBellToll() {
	logger.Println("Disabling BellToll mode")
	c.bellTollKill <- true
}

func (c Controller) EnableFixedHeight() {
	c.desk.AddListener(new(FixedHeightListener))
}

// FixedHeightListener is a listener that will reset the desk to a configured height.
type FixedHeightListener struct {
	EmptyListener
	height float32
}

func (listener FixedHeightListener) HeightChanged(newHeight float32) {
	logger.Println("Listener notified of height change")
}
