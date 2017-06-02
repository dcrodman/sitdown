package main

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
	maxHeight  int     = 219
	minHeight  int     = 25
	baseHeight float32 = 28.1
)

var serialOptions = serial.OpenOptions{
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

type Desk struct {
	pinButtonUp   rpio.Pin
	pinButtonDown rpio.Pin

	moveMux       *sync.Mutex
	serialFile    io.ReadWriteCloser
	currentHeight float32
}

var desk *Desk

func (d *Desk) Setup(log *log.Logger) {
	desk = new(Desk)
	d.pinButtonUp = rpio.Pin(16)
	d.pinButtonDown = rpio.Pin(12)

	if err := rpio.Open(); err != nil {
		panic(err)
	}

	var err error
	if d.serialFile, err = serial.Open(serialOptions); err != nil {
		panic(err)
	}

	go d.heightMonitor()

	d.pinButtonUp.Output()
	d.pinButtonUp.PullUp()
	d.pinButtonUp.High()
	d.pinButtonDown.Output()
	d.pinButtonDown.PullUp()
	d.pinButtonDown.High()
}

func (d *Desk) Cleanup() {
	defer rpio.Close()
	d.pinButtonUp.PullOff()
	d.pinButtonDown.PullOff()
}

func (d Desk) lock() {
	d.moveMux.Lock()
}

func (d Desk) unlock() {
	d.moveMux.Unlock()
}

func (d Desk) RaiseForDuration(duration int) {
	d.lock()
	defer d.unlock()
	d.raise()
	sleep(duration)
	d.Stop()
}

func (d Desk) LowerForDuration(duration int) {
	d.lock()
	defer d.unlock()
	d.lower()
	sleep(duration)
	d.Stop()
}

func (d Desk) ChangeToHeight(height float32) {
	d.lock()
	defer d.unlock()
	var acceptableRange float32 = 0.75
	for math.Abs(float64(height-d.currentHeight)) < float64(acceptableRange) {
		acceptableRange *= 0.75
	}
	destLow := height - acceptableRange
	destHigh := height + acceptableRange
	for {
		if destLow <= d.currentHeight && d.currentHeight <= destHigh {
			d.Stop()
			return
		} else if d.currentHeight > height {
			d.pinButtonUp.High()
			d.pinButtonDown.Low()
		} else if d.currentHeight < height {
			d.pinButtonDown.High()
			d.pinButtonUp.Low()
		}
	}
}

func (d Desk) raise() {
	d.pinButtonUp.Low()
}

func (d Desk) lower() {
	d.pinButtonDown.Low()
}

func (d Desk) Stop() {
	d.pinButtonUp.High()
	d.pinButtonDown.High()
}

func (d Desk) Height() float32 {
	return d.currentHeight
}

func (d Desk) heightMonitor() {
	for {
		data := make([]byte, 4)
		n, err := d.serialFile.Read(data)
		if n < 4 || err == io.EOF {
			sleep(50)
		} else if err != nil {
			panic(err)
		} else if data[1] == 1 {
			newHeight := baseHeight + float32(int(data[3])-minHeight)/10
			if newHeight != d.currentHeight {
				logger.Printf("Height changed to %.1f from %.1f\n", newHeight, d.currentHeight)
				d.currentHeight = newHeight
			}
		}
	}
}

func sleep(ms int) {
	time.Sleep(time.Duration(ms) * time.Millisecond)
}