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

// Desk is the singleton controller for the GPIO pins that control the desk.
type Desk struct {
	pinButtonUp   rpio.Pin
	pinButtonDown rpio.Pin

	moveMux       *sync.Mutex
	serialFile    io.ReadWriteCloser
	currentHeight float32

	listeners []DeskListener
}

func (d *Desk) Setup(log *log.Logger) {
	d.pinButtonUp = rpio.Pin(16)
	d.pinButtonDown = rpio.Pin(12)
	d.moveMux = new(sync.Mutex)

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
	d.Stop()
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

	for _, listener := range d.listeners {
		listener.DeskRaised()
	}
}

func (d Desk) LowerForDuration(duration int) {
	d.lock()
	defer d.unlock()
	d.lower()
	sleep(duration)
	d.Stop()

	for _, listener := range d.listeners {
		listener.DeskLowered()
	}
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
			for _, listener := range d.listeners {
				listener.HeightSet(destHigh)
			}
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
	d.pinButtonUp.PullDown()
	d.pinButtonUp.Low()
}

func (d Desk) lower() {
	d.pinButtonDown.PullDown()
	d.pinButtonDown.Low()
}

func (d Desk) Stop() {
	d.pinButtonUp.PullUp()
	d.pinButtonUp.High()
	d.pinButtonDown.PullUp()
	d.pinButtonDown.High()
}

func (d Desk) Height() float32 {
	return d.currentHeight
}

func (d *Desk) heightMonitor() {
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
				d.currentHeight = newHeight
				for _, listener := range d.listeners {
					listener.HeightChanged(newHeight)
				}
			}
		}
	}
}

func (d *Desk) AddListener(listener DeskListener) {
	d.listeners = append(d.listeners, listener)
}

func (d *Desk) ResetListeners() {
	d.listeners = nil
}

func sleep(ms int) {
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

// DeskListener is an interface for functions that want to be notified of
// some change of state in the desk.
type DeskListener interface {
	// DeskRaised is called when the desk has been raised to a new height.
	DeskRaised()
	// DeskRaised is called when the desk has been lowered to a new height.
	DeskLowered()
	// HeightSet is called after the desk has been set to a specified height.
	HeightSet(newHeight float32)
	// HeightChanged is called on a DeskListener whenever the height of the desk changes.
	// Note that this is called by the serial stream monitor and will be hit often.
	HeightChanged(newHeight float32)
}

// EmptyListener is a no-op listener that can be embedded for convenience.
type EmptyListener struct{}

func (listener *EmptyListener) DeskRaised()                     {}
func (listener *EmptyListener) DeskLowered()                    {}
func (listener *EmptyListener) HeightChanged(newHeight float32) {}
func (listener *EmptyListener) HeightSet(newHeight float32)     {}
