package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
)

var (
	// Global logger that should be used for any output.
	logger = log.New(os.Stdin, "", log.Ltime)
	// Controller instance for the currently running sitdown process.
	controller *Controller
)

func main() {
	commandMode := flag.Bool("c", false, "Start the server in command mode")
	port := flag.String("p", "8080", "Listen on the specified port")
	flag.Parse()

	controller = &Controller{
		activeControllers: make(map[string]string),
		bellTollKill:      make(chan bool, 1),
	}
	controller.initFromConfig()

	registerSignalHandlers()

	InitializePubNub()
	defer CleanupPubNub()

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
		CleanupPubNub()
		os.Exit(0)
	}()
}
