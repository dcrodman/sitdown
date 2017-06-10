package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
)

var (
	// Global logger that should be used for any output.
	logger = log.New(os.Stdin, "", log.Ltime)
	// Controller instance for the currently running sitdown process.
	controller *Controller
	// Messenger instance responsible for PubNub communication.
	messenger *Messenger
)

func main() {
	commandMode := flag.Bool("c", false, "Start the server in command mode")
	resetMode := flag.Bool("r", false, "Reset the pins to HIGH in case they're stuck")
	port := flag.String("p", "8080", "Listen on the specified port")
	flag.Parse()

	controller = new(Controller)
	controller.InitFromConfig()

	// Only set the pins back to HIGH and then exit.
	if *resetMode {
		controller.desk.Setup(logger)
		controller.Cleanup()
		return
	}

	registerSignalHandlers()

	messenger = new(Messenger)
	messenger.Initialize()
	defer messenger.Cleanup()

	if *commandMode {
		controller.EnterCommandMode()
	} else {
		controller.EnterDeskControlMode()
		defer controller.Cleanup()
		StartHTTPEndpoint(*port)
	}
}

// Attempt to cover all of our bases for cleanup.
func registerSignalHandlers() {
	killChan := make(chan os.Signal)
	signal.Notify(killChan, os.Interrupt, os.Kill)

	go func() {
		<-killChan
		logger.Println("Cleaning up from signal handler")
		controller.Cleanup()
		messenger.Cleanup()
		os.Exit(0)
	}()
}

// Start and block on an HTTP client listening for commands from the network.
func StartHTTPEndpoint(port string) {
	http.HandleFunc("/move", HandleMove)
	http.HandleFunc("/set", HandleSet)
	http.HandleFunc("/height", HandleHeight)
	logger.Println("Starting HTTP server")

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		panic(err)
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
	controller.Move(direction, duration)
	fmt.Fprintf(responseWriter, "Moved to %.1f", controller.GetHeight())
}

// Handler method for HTTP requests sent to /set.
func HandleSet(responseWriter http.ResponseWriter, request *http.Request) {
	vals, err := url.ParseQuery(request.URL.RawQuery)
	if err != nil {
		logger.Println(err)
		return
	}
	controller.SetHeight(vals["height"][0])
	fmt.Fprintf(responseWriter, "Changed to %.1f", controller.GetHeight())
}

// Handler method for HTTP requests sent to /height.
func HandleHeight(responseWriter http.ResponseWriter, request *http.Request) {
	fmt.Fprintf(responseWriter, "%.1f", controller.GetHeight())
}
