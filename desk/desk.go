package desk

import (
	"github.com/stianeikeland/go-rpio"
	"github.com/jacobsa/go-serial/serial"
	"sync"
	"encoding/hex"
	"fmt"
	"io"
)

const (
	pinButtonUp   rpio.Pin = rpio.Pin(16)
	pinButtonDown rpio.Pin = rpio.Pin(12)
)

var (
	moveMux = &sync.Mutex{}
	serialFile io.ReadWriteCloser 
	serialOptions = serial.OpenOptions{
		PortName: "/dev/serial0",
		BaudRate: 9600,
	}
)

func Setup() {
	if err := rpio.Open(); err != nil {
		panic(err)
	}
	if serialFile, err := serial.Open(serialOptions); err != nil {
		panic(err)
	}
}

func Cleanup() {
	rpio.Close()
	pinButtonUp.PullOff()
	pinButtonDown.PullOff()
}

func Lock() {
	moveMux.Lock()
}

func Unlock() {
	moveMux.Unlock()
}

func Raise() {
	pinButtonUp.Output()
	pinButtonUp.PullUp()
	pinButtonUp.Low()
}

func StopRaising() {
	pinButtonUp.Output()
	pinButtonUp.PullUp()
	pinButtonUp.High()
}

func Lower() {
	pinButtonDown.Output()
	pinButtonDown.PullUp()
	pinButtonDown.Low()
}

func StopLowering() {
	pinButtonDown.Output()
	pinButtonDown.PullUp()
	pinButtonDown.High()
}

func Height() []byte {
	buf := make([]byte, 32)
	n, err := serialFile.Read(buf)
	if err != nil {
		panic(err)
	} else {
		buf = buf[:n]
		fmt.Println("Rx: ", hex.EncodeToString(buf))
		return buf
	}
}
