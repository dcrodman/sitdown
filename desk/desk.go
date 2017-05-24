package desk

import (
	"github.com/stianeikeland/go-rpio"
	"github.com/jacobsa/go-serial/serial"
	"sync"
	"fmt"
	"io"
	"time"
)

const (
	pinButtonUp   rpio.Pin = rpio.Pin(16)
	pinButtonDown rpio.Pin = rpio.Pin(12)
	maxHeight     int = 219
	minHeight     int = 25
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
	currentHeight int
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

func Height() int {
	return currentHeight
}

func heightMonitor() {
	height := -1
	for {
		data := make([]byte, 4)
		n, err := serialFile.Read(data)
		if n < 4 || err != nil {
			panic(err)
		} else {
			height = 100 * (int(data[3]) - minHeight) / (maxHeight - minHeight)
			if height != currentHeight {
				fmt.Printf("Height changed to %d%% from %d%%\n", height, currentHeight)
				currentHeight = height
			}
		}
	}
}

func sleep(ms int) {
	time.Sleep(time.Duration(ms) * time.Millisecond)
}
