package desk

import (
	"github.com/stianeikeland/go-rpio"
	"github.com/jacobsa/go-serial/serial"
	"sync"
	"math"
	"fmt"
	"io"
	"time"
)

const (
	pinButtonUp   rpio.Pin = rpio.Pin(16)
	pinButtonDown rpio.Pin = rpio.Pin(12)
	maxHeight     int = 219
	minHeight     int = 25
	baseHeight    float32 = 28.1
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
	currentHeight float32 
)

func Setup() {
	if err := rpio.Open(); err != nil {
		panic(err)
	}
	var err error
	if serialFile, err = serial.Open(serialOptions); err != nil {
		panic(err)
	}
	go heightMonitor()
	pinButtonUp.Output()
	pinButtonUp.PullUp()
	pinButtonDown.Output()
	pinButtonDown.PullUp()
}

func Cleanup() {
	defer rpio.Close()
	pinButtonUp.PullOff()
	pinButtonDown.PullOff()
}

func lock() {
	moveMux.Lock()
}

func unlock() {
	moveMux.Unlock()
}

func RaiseForDuration(duration int) {
	lock()
	defer unlock()
	raise()
	sleep(duration)
	stop()
}

func LowerForDuration(duration int) {
	lock()
	defer unlock()
	raise()
	sleep(duration)
	stop()
}

func ChangeToHeight(height float32) {
	lock()
	defer unlock()
	acceptableRange := 0.75
	for math.Abs(height - currentHeight) < acceptableRange {
		acceptableRange *= 0.75
	}
	destLow := height - acceptableRange 
	destHigh := height + acceptableRange 
	for {
		if destLow <= currentHeight && currentHeight <= destHigh {
			stop()
			return
		} else if currentHeight > height {
			pinButtonUp.High()
			pinButtonDown.Low()
		} else if currentHeight < height {
			pinButtonDown.High()
			pinButtonUp.Low()
		}
	}
}

func raise() {
	pinButtonUp.Low()
}

func lower() {
	pinButtonDown.Low()
}

func stop() {
	pinButtonUp.High()
	pinButtonDown.High()
}

func Height() float32 {
	return currentHeight
}

func heightMonitor() {
	for {
		data := make([]byte, 4)
		n, err := serialFile.Read(data)
		if n < 4 || err != nil {
			panic(err)
		} else {
			newHeight := baseHeight + float32(int(data[3]) - minHeight) / 10
			if newHeight != currentHeight {
				fmt.Printf("Height changed to %.1f from %.1f\n", newHeight, currentHeight)
				currentHeight = newHeight
			}
		}
	}
}

func sleep(ms int) {
	time.Sleep(time.Duration(ms) * time.Millisecond)
}
