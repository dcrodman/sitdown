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
		DataBits: 8,
		StopBits: 1,
		MinimumReadSize: 0,
		InterCharacterTimeout: 100,
		ParityMode: serial.PARITY_NONE,
		Rs485Enable: false,
		Rs485RtsHighDuringSend: false,
		Rs485RtsHighAfterSend: false,
	}
)

func Setup() {
	if err := rpio.Open(); err != nil {
		panic(err)
	}
	var err error
	if serialFile, err = serial.Open(serialOptions); err != nil {
		panic(err)
	}
}

func Cleanup() {
	defer rpio.Close()
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

func Height() byte {
	buf := make([]byte, 4)
	n, err := serialFile.Read(buf)
	if err != nil {
		panic(err)
	} else {
		fmt.Println("Rx: ", hex.EncodeToString(buf[3]))
		return buf[3]
	}
}
