package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

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
