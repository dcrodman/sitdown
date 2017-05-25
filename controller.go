package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/dcrodman/sitdown/desk"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

func main() {
	commandMode := flag.Bool("c", false, "Start the server in command mode")
	port := flag.String("p", "8080", "Listen on the specified port")
	flag.Parse()

	if *commandMode {
		EnterCommandMode()
		os.Exit(0)
	}

	desk.Setup()
	defer desk.Cleanup()

	SubscribeToChannel()

	http.HandleFunc("/move", HandleMove)
	http.HandleFunc("/set", HandleSet)
	http.HandleFunc("/height", HandleHeight)
	fmt.Println("Starting HTTP server")
	if err := http.ListenAndServe(":"+*port, nil); err != nil {
		panic(err)
	}
}

func EnterCommandMode() {
	fmt.Println("Entering Command Mode")
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Command: ")
		command, _ := reader.ReadString('\n')
		command = strings.Trim(command, "\n ")
		if strings.ToLower(command) == "exit" {
			break
		}

		PublishCommand(Command(command), commandClientId, "")
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
	fmt.Fprintf(responseWriter, "Moved to %.1f", desk.Height())
}

func HandleSet(responseWriter http.ResponseWriter, request *http.Request) {
	vals, err := url.ParseQuery(request.URL.RawQuery)
	if err != nil {
		fmt.Println(err)
		return
	}
	setHeight(vals["height"][0])
	fmt.Fprintf(responseWriter, "Changed to %.1f", desk.Height())
}

func HandleHeight(responseWriter http.ResponseWriter, request *http.Request) {
	fmt.Fprintf(responseWriter, "%.1f", desk.Height())
}

func setHeight(height string) {
	h, err := strconv.ParseFloat(height, 32)
	if err != nil || h < 28.1 || h > 47.5 {
		fmt.Printf("Invalid height: %f\n", h)
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
