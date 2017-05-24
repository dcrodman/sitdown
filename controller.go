package main

import (
	"fmt"
	"github.com/dcrodman/sitdown/desk"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

func main() {
	for _, arg := range os.Args {
		switch arg {
		case "-c":
			EnterCommandMode()
			os.Exit(0)
		case "-t", "--test":
			desk.Setup()
			defer desk.Cleanup()

			move("up", 2000)
			move("down", 2500)
			os.Exit(0)
		}
	}

	desk.Setup()
	defer desk.Cleanup()

	PubNubSubscribe()

	http.HandleFunc("/move", HandleMove)
	http.HandleFunc("/set", HandleSet)
	http.HandleFunc("/height", HandleHeight)
	fmt.Println("Starting HTTP server")
	if err := http.ListenAndServe(":8080", nil); err != nil {
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
	fmt.Fprintf(responseWriter, "Moved to %.1f", desk.Height())
}

func HandleSet(responseWriter http.ResponseWriter, request *http.Request) {
	vals, err := url.ParseQuery(request.URL.RawQuery)
	if err != nil {
		fmt.Println(err)
		return
	}
	height, err := strconv.ParseFloat(vals["height"][0], 32)
	if err != nil || height < 28.1 || height > 47.5 {
		fmt.Printf("Invalid height: %d\n", height)
		return
	}

	fmt.Printf("Received set command: %.1f\n", height)
	desk.ChangeToHeight(float32(height))
	fmt.Fprintf(responseWriter, "Changed to %.1f", desk.Height())
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
