package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/dcrodman/sitdown/desk"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
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
	// List of paired Pis to which to send commands.

	logger = log.New(os.Stdin, "", log.Ltime)
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
	}

	desk.Setup()
	defer desk.Cleanup()

	StartSubscriber(DeskCommandHandler)
	StartAnnouncing()

	http.HandleFunc("/move", HandleMove)
	http.HandleFunc("/set", HandleSet)
	http.HandleFunc("/height", HandleHeight)
	logger.Println("Starting HTTP server")
	if err := http.ListenAndServe(":"+*port, nil); err != nil {
		panic(err)
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
		CleanupPubNub()
		desk.Cleanup()
		logger.Println("Cleaning up from signal handler")
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
	fmt.Println("Entering Command Mode")
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
		command, _ := reader.ReadString('\n')
		command = strings.Trim(command, "\n ")

		switch strings.ToLower(command) {
		case "list":
			for id, ip := range activeControllers {
				logger.Printf("Controller @ %s (id: %s)]\n", ip, id)
			}
			continue
		case "exit":
			break loop
		}

		PublishCommand(Command(command), "")
	}
	os.Exit(0)
}

// Message handler specifically for command mode.
func CommandModeSubscribeHandler(message Message) {
	splitCommand := strings.Split(string(message.Action), " ")
	switch Command(splitCommand[0]) {
	case Announce:
		logger.Printf("Discovered controller %s (id: %s)\n", message.IPAddr, message.Id)
		addKnownController(message.Id, message.IPAddr)
	}
}

// Handler method for HTTP requests sent to /move.
func HandleMove(responseWriter http.ResponseWriter, request *http.Request) {
	vals, err := url.ParseQuery(request.URL.RawQuery)
	if err != nil {
		logger.Println(err)
		return
	}
	direction := vals["direction"][0]
	if direction != "down" && direction != "up" {
		logger.Printf("Invalid direction: %s\n", direction)
		return
	}
	duration, err := strconv.Atoi(vals["time"][0])
	if err != nil || duration < 0 || duration > 10000 {
		logger.Printf("Invalid time: %d\n", duration)
		return
	}

	logger.Printf("Received move command: %s %d\n", direction, duration)
	move(direction, duration)
	fmt.Fprintf(responseWriter, "Moved to %.1f", desk.Height())
}

// Handler method for HTTP requests sent to /set.
func HandleSet(responseWriter http.ResponseWriter, request *http.Request) {
	vals, err := url.ParseQuery(request.URL.RawQuery)
	if err != nil {
		logger.Println(err)
		return
	}
	setHeight(vals["height"][0])
	fmt.Fprintf(responseWriter, "Changed to %.1f", desk.Height())
}

// Handler method for HTTP requests sent to /height.
func HandleHeight(responseWriter http.ResponseWriter, request *http.Request) {
	fmt.Fprintf(responseWriter, "%.1f", desk.Height())
}

// Command handler that should be running on the actual desk controllers.
func DeskCommandHandler(message Message) {
	splitCommand := strings.Split(string(message.Action), " ")
	switch Command(splitCommand[0]) {
	case Move:
		switch len := len(splitCommand); len {
		case 1:
			logger.Println("Missing direction from move command (skipping)")
		case 2:
			move(splitCommand[1], 1000)
		default:
			duration, _ := strconv.ParseInt(splitCommand[2], 10, 32)
			move(splitCommand[1], int(duration))
		}
	case SetHeight:
		if len := len(splitCommand); len <= 1 {
			logger.Println("Missing height for set command (skipping)")
		}
		setHeight(splitCommand[1])
	case Announce:
		logger.Printf("Discovered controller %s (id: %s)\n", message.IPAddr, message.Id)
		addKnownController(message.Id, message.IPAddr)
	default:
		logger.Printf("Unrecognized command %v; skipping\n", message.Action)
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
