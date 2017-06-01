package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/dcrodman/sitdown/desk"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"
)

type configuration struct {
	ControllerID string
	PubKey       string
	SubKey       string
}

const configFilename = "controller.conf"

var (
	// Global map of the IDs of active desk controllers to their IP addresses. This isn't
	// explicitly threadsafe but is only ever modified by one thread.
	activeControllers = make(map[string]string)
	// ID for the current instance of sitdown.
	config configuration
	// Global logger that should be used for any output.
	logger = log.New(os.Stdin, "", log.Ltime)

	// Channel specifically for killing BellToll mode.
	bellTollKill = make(chan bool, 1)
)

func main() {
	commandMode := flag.Bool("c", false, "Start the server in command mode")
	port := flag.String("p", "8080", "Listen on the specified port")
	flag.Parse()

	readConfig()
	registerSignalHandlers()

	InitializePubNub()
	defer CleanupPubNub()

	if *commandMode {
		EnterCommandMode()
	} else {
		desk.Setup(logger)
		defer desk.Cleanup()

		StartSubscriber(DeskCommandSubscriberHandler)
		StartAnnouncing()

		StartHTTPEndpoint(*port)
	}
}

func readConfig() {
	fileContents, err := ioutil.ReadFile(configFilename)
	if err != nil {
		fileContents, err = ioutil.ReadFile("/home/pi/" + configFilename)
		if err != nil {
			fmt.Printf("Unable to locate %s in local dir or /home/pi\n", configFilename)
			os.Exit(1)
		}
	}

	json.Unmarshal([]byte(fileContents), &config)
	logger.Printf("Initializing Pi with config: %#v\n", config)
}

// Attempt to cover all of our bases for cleanup.
func registerSignalHandlers() {
	killChan := make(chan os.Signal)
	signal.Notify(killChan, os.Interrupt, os.Kill)

	go func() {
		<-killChan
		logger.Println("Cleaning up from signal handler")
		desk.Cleanup()
		CleanupPubNub()
		os.Exit(0)
	}()
}

// Command client mode for communicating with the desk controllers remotely. This is
// invoked with the -c command line argument from any machine. Does not have to be on
// the same network since all of the commands are passed through PubNub.
//
// list: Show all controllers that the command client is aware of
// exit: Kill the prompt
// Anything else will be published directly to the controllers
func EnterCommandMode() {
	fmt.Println("Entering Command Mode (syntax: cmd target [params])")
	logFile, err := os.Create("controller.log")
	if err != nil {
		fmt.Printf("Could not open controller.log")
		os.Exit(0)
	}
	// Reassign the logger from stdout so that we don't interfere with the prompt.
	logger = log.New(logFile, "", log.Ltime)
	config.ControllerID = CommandClientId

	reader := bufio.NewReader(os.Stdin)
	StartSubscriber(CommandModeSubscribeHandler)

loop:
	for {
		fmt.Print("Command: ")
		fullCommand, _ := reader.ReadString('\n')
		fullCommand = strings.Trim(fullCommand, "\n ")

		splitFullCommand := strings.Split(fullCommand, " ")
		action := splitFullCommand[0]

		switch strings.ToLower(action) {
		case "list":
			for id, ip := range activeControllers {
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
			PublishCommand(Command(action), "", target, splitFullCommand[2:])
		} else {
			PublishCommand(Command(action), "", target, nil)
		}
	}
	os.Exit(0)
}

// Command handler for messages received while in command mode.
func CommandModeSubscribeHandler(message Message) {
	splitCommand := strings.Split(string(message.Action), " ")
	switch Command(splitCommand[0]) {
	case Announce:
		logger.Printf("Discovered controller %s (id: %s)\n", message.IPAddr, message.ID)
		addKnownController(message.ID, message.IPAddr)
	}
}

// Command handler that should be running on the actual desk controllers.
func DeskCommandSubscriberHandler(message Message) {
	switch Command(message.Action) {
	case Move:
		switch len(message.Params) {
		case 0:
			logger.Println("Missing direction from move command (skipping)")
		case 1:
			move(message.Params[0], 1000)
		default:
			duration, _ := strconv.ParseInt(message.Params[1], 10, 32)
			move(message.Params[0], int(duration))
		}
	case SetHeight:
		if len(message.Params) <= 1 {
			logger.Println("Missing height for set command (skipping)")
		}
		setHeight(message.Params[0])
	case BellToll:
		if len(message.Params) < 1 {
			logger.Println("Missing enable/disable; skipping")
		} else if message.Params[0] == "enable" {
			logger.Println("Enabling BellToll mode")
			go bellToll()
		} else {
			logger.Println("Disabling BellToll mode")
			bellTollKill <- true
		}
	case Announce:
		logger.Printf("Discovered controller %s (id: %s)\n", message.IPAddr, message.ID)
		addKnownController(message.ID, message.IPAddr)
	default:
		logger.Printf("Unrecognized command %v; skipping\n", message.Action)
	}
}

func bellToll() {
	// Start tolling at the next hour so the desk doesn't move immediately.
	lastTolled := time.Now().Hour() % 12
loop:
	for {
		timer := time.NewTimer(10 * time.Second)
		select {
		case <-bellTollKill:
			break loop
		case <-timer.C:
			thisHour := time.Now().Hour() % 12
			if thisHour != lastTolled {
				for i := 0; i < thisHour; i++ {
					move("up", 1000)
					move("down", 1050)
				}
				lastTolled = thisHour
			}
		}
	}
}

func setHeight(height string) {
	h, err := strconv.ParseFloat(height, 32)
	if err != nil || h < 28.1 || h > 47.5 {
		logger.Printf("Invalid height: %f\n", h)
		return
	}
	desk.ChangeToHeight(float32(h))
}

func move(direction string, time int) {
	switch direction {
	case "up":
		desk.RaiseForDuration(time)
	case "down":
		desk.LowerForDuration(time)
	}
}

func addKnownController(id, ipAddr string) {
	activeControllers[id] = ipAddr
}
