package desk

import (
	"github.com/jacobsa/go-serial/serial"
	"github.com/stianeikeland/go-rpio"
	"io"
	"log"
	"math"
	"sync"
	"time"
)

const (
	pinButtonUp   rpio.Pin = rpio.Pin(16)
	pinButtonDown rpio.Pin = rpio.Pin(12)
	maxHeight     int      = 219
	minHeight     int      = 25
	baseHeight    float32  = 28.1
)

var (
	moveMux       = &sync.Mutex{}
	serialFile    io.ReadWriteCloser
	serialOptions = serial.OpenOptions{
		PortName:               "/dev/serial0",
		BaudRate:               9600,
		DataBits:               8,
		StopBits:               1,
		MinimumReadSize:        0,
		InterCharacterTimeout:  100,
		ParityMode:             serial.PARITY_NONE,
		Rs485Enable:            false,
		Rs485RtsHighDuringSend: false,
		Rs485RtsHighAfterSend:  false,
	}
	currentHeight float32

	logger *log.Logger
)

func Setup(log *log.Logger) {
	if err := rpio.Open(); err != nil {
		panic(err)
	}
	var err error
	if serialFile, err = serial.Open(serialOptions); err != nil {
		panic(err)
	}

	logger = log
	go heightMonitor()

	pinButtonUp.Output()
	pinButtonUp.PullUp()
	pinButtonUp.High()
	pinButtonDown.Output()
	pinButtonDown.PullUp()
	pinButtonDown.High()
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
	Stop()
}

func LowerForDuration(duration int) {
	lock()
	defer unlock()
	lower()
	sleep(duration)
	Stop()
}

func ChangeToHeight(height float32) {
	lock()
	defer unlock()
	var acceptableRange float32 = 0.75
	for math.Abs(float64(height-currentHeight)) < float64(acceptableRange) {
		acceptableRange *= 0.75
	}
	destLow := height - acceptableRange
	destHigh := height + acceptableRange
	for {
		if destLow <= currentHeight && currentHeight <= destHigh {
			Stop()
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

func Stop() {
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
		if n < 4 || err == io.EOF {
			sleep(50)
		} else if err != nil {
			panic(err)
		} else if data[1] == 1 {
			newHeight := baseHeight + float32(int(data[3])-minHeight)/10
			if newHeight != currentHeight {
				logger.Printf("Height changed to %.1f from %.1f\n", newHeight, currentHeight)
				currentHeight = newHeight
			}
		}
	}
}

func sleep(ms int) {
	time.Sleep(time.Duration(ms) * time.Millisecond)
}
