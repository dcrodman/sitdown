package main

import (
	"fmt"
	"github.com/dcrodman/sitdown/desk"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"flag"
)

func main() {
	commandMode := flag.Bool("c", false, "Start the server in command mode")
	testMode := flag.Bool("t", false, "Run a quick test that moves the desk up then down")
	port := flag.String("p", "8080", "Listen on the specified port")
	flag.Parse()

	if *commandMode {
		EnterCommandMode()
		os.Exit(0)
	}
	if *testMode {
		desk.Setup()
		defer desk.Cleanup()

		move("up", 2000)
		move("down", 2500)
		os.Exit(0)
	}

	desk.Setup()
	defer desk.Cleanup()

	PubNubSubscribe()

	http.HandleFunc("/move", HandleMove)
	http.HandleFunc("/set", HandleSet)
	http.HandleFunc("/height", HandleHeight)
	fmt.Println("Starting HTTP server")
	if err := http.ListenAndServe(":" + *port, nil); err != nil {
		panic(err)
	}
}

func HandleMove(responseWriter http.ResponseWriter, request *http.Request) {
	vals, err := url.ParseQuery(request.URL.RawQuery)
	if err != nil {
		fmt.Println(err)
		return
	}
	direction := vals["direction"][0]
	if direction != "down" && direction != "up" {
		fmt.Printf("Invalid direction: %s\n", direction)
		return
	}
	duration, err := strconv.Atoi(vals["time"][0])
	if err != nil || duration < 0 || duration > 10000 {
		fmt.Printf("Invalid time: %d\n", duration)
		return
	}

	fmt.Printf("Received move command: %s %d\n", direction, duration)
	move(direction, duration)
}

func HandleSet(responseWriter http.ResponseWriter, request *http.Request) {
	vals, err := url.ParseQuery(request.URL.RawQuery)
	if err != nil {
		fmt.Println(err)
		return
	}
	height, err := strconv.Atoi(vals["height"][0])
	if err != nil || height < 0 || height > 100 {
		fmt.Printf("Invalid height: %d\n", height)
		return
	}

	fmt.Printf("Received set command: %d\n", height)
}

func HandleHeight(responseWriter http.ResponseWriter, request *http.Request) {
	fmt.Fprintf(responseWriter, "%.1f", desk.Height())
}

func move(direction string, time int) {
	switch direction {
	case "up":
		desk.RaiseForDuration(time)
	case "down":
		desk.LowerForDuration(time)
	}
}
