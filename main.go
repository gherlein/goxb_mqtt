// goxb_mqtt

// code to read from an XBox360(tm) controller and write events to an MQTT broker
// See the README.md file for documentation

package main

import (
	"flag"
	"fmt"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	. "github.com/gherlein/xbevents"
	"github.com/google/gousb"
	"io"
	"log"
	"math"
)

// offests into the 20 byte controller packet
const (
	XB_PAD    = 2
	XB_DECK   = 3
	XB_LT     = 4
	XB_RT     = 5
	XB_LJOY1X = 6
	XB_LJOY2X = 7
	XB_LJOY1Y = 8
	XB_LJOY2Y = 9
	XB_RJOY1X = 10
	XB_RJOY2X = 11
	XB_RJOY1Y = 12
	XB_RJOY2Y = 13
)

var (
	xbe               *XBevent
	vid               gousb.ID = 0x045e
	pid               gousb.ID = 0x028e
	iface             int      = 0
	alternate         int      = 0
	endpoint          int      = 1
	config            int      = 1
	usbdebug          int      = 3
	size              int      = 64
	bufSize           int      = 0
	num               int      = 0
	lx                int16    = 0
	ly                int16    = 0
	rx                int16    = 0
	ry                int16    = 0
	deadzone          int      = 512
	support_xy        bool     = false
	support_xy_topics bool     = true
	support_vector    bool     = false
	support_buttons   bool     = true
	debug             bool     = true
	debugraw          bool     = true
	debugvector       bool     = false
	debugjoy          bool     = false
	debugtrigger      bool     = false
	debugbutton       bool     = false
	broker            string   = "tcp://localhost:1883"
	joysticks         string   = "xb/1/joysticks"
	triggers          string   = "xb/1/triggers"
	buttons           string   = "xb/1/buttons"
	qos               int      = 0
	xmult             int16    = 1
	ymult             int16    = -1
	client            MQTT.Client
)

func init() {
	flag.StringVar(&broker, "broker", "tcp://localhost:1883", "broker connection string")
	flag.IntVar(&deadzone, "deadzone", 1024, "center deadzone value for joysticks")
	flag.Parse()
}

func main() {

	var xbe *XBevent
	opts := MQTT.NewClientOptions()
	opts.AddBroker(broker)
	client = MQTT.NewClient(opts)

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	ctx := gousb.NewContext()
	defer ctx.Close()
	ctx.Debug(usbdebug)

	var rdr io.Reader = openXB(ctx)

	buf1 := make([]byte, size)
	buf2 := make([]byte, size)
	var buf []byte

	var odd bool = false

	initBuffer(buf1)
	initBuffer(buf2)

	for i := 0; num == 0 || i < num; i++ {
		if odd {
			buf = buf1
		} else {
			buf = buf2
		}
		num, err := rdr.Read(buf)

		if err != nil {
			log.Fatalf("Reading from device failed: %v", err)
		}
		if num != 20 {
			fmt.Println("did not read 20 bytes")
			continue
		}

		xbe = parseEvent(buf1, buf2, odd)

		if odd {
			odd = false
		} else {
			odd = true
		}

		if xbe.Code == LJOYX ||
			xbe.Code == LJOYY ||
			xbe.Code == RJOYX ||
			xbe.Code == RJOYY {
			if debug == true {
				fmt.Printf("%s - x: %d   y: %d\n", xbe.Name, xbe.X, xbe.Y)
			}
			send_joystick(xbe)

		} else if xbe.Code == LT || xbe.Code == RT {
			if debug {
				fmt.Printf("%s - v: %d\n", xbe.Name, xbe.X)
			}
			send_trigger(xbe)
		} else {
			if debug {
				fmt.Printf("%s\n", xbe.Name)
			}
			send_button(xbe)
		}
	}
}

func send_button(xbe *XBevent) {
	var msg string = "*"
	msg = fmt.Sprintf("%s", xbe.Name)
	if debugbutton {
		fmt.Println(msg)
	}
	token := client.Publish(buttons, byte(qos), false, msg)
	token.Wait()
}
func send_trigger(xbe *XBevent) {
	var msg string = "*|*"
	if xbe.Code == LT {
		msg = fmt.Sprintf("LT|%d", xbe.X)
	}
	if xbe.Code == RT {
		msg = fmt.Sprintf("RT|%d", xbe.X)
	}
	if support_xy {
		if debugtrigger {
			fmt.Printf("%s\n", msg)
		}
		token := client.Publish(joysticks, byte(qos), false, msg)
		token.Wait()
	}
}

func send_joystick(xbe *XBevent) {
	if support_xy {
		var msg string = "*|*|*"
		if xbe.Code == LJOYX {
			msg = fmt.Sprintf("L|X|%d|%d", xbe.X, xbe.Y)
		}
		if xbe.Code == LJOYY {
			msg = fmt.Sprintf("L|Y|%d|%d", xbe.X, xbe.Y)
		}
		if xbe.Code == RJOYX {
			msg = fmt.Sprintf("R|X|%d|%d", xbe.X, xbe.Y)
		}
		if xbe.Code == RJOYY {
			msg = fmt.Sprintf("R|Y|%d|%d", xbe.X, xbe.Y)
		}
		if debugjoy {
			fmt.Printf("%s\n", msg)
		}
		token := client.Publish(joysticks, byte(qos), false, msg)
		token.Wait()
	}
	if support_xy_topics {
		var msg string = "*"
		var topic string
		if xbe.Code == LJOYX {
			msg = fmt.Sprintf("%d", xbe.X)
			topic = fmt.Sprintf("%s/L/X", joysticks)
		}
		if xbe.Code == LJOYY {
			msg = fmt.Sprintf("%d", xbe.Y)
			topic = fmt.Sprintf("%s/L/Y", joysticks)
		}
		if xbe.Code == RJOYX {
			msg = fmt.Sprintf("%d", xbe.X)
			topic = fmt.Sprintf("%s/R/X", joysticks)
		}
		if xbe.Code == RJOYY {
			msg = fmt.Sprintf("%d", xbe.Y)
			topic = fmt.Sprintf("%s/R/Y", joysticks)
		}
		if debugjoy {
			fmt.Printf("%s\n", msg)
		}
		fmt.Printf("%s - %s\n", topic, msg)
		token := client.Publish(topic, byte(qos), false, msg)
		token.Wait()
	}

}

func parseEvent(b1 []byte, b2 []byte, odd bool) *XBevent {

	var xbe XBevent
	var debugLast bool = false
	var buf []byte
	var last []byte
	var data int16

	if odd {
		buf = b1
		last = b2
	} else {
		buf = b2
		last = b1
	}

	if debugLast {
		for i := 0; i < 20; i++ {
			fmt.Printf("%02X", buf[i])

		}
		fmt.Printf("\n")
		for i := 0; i < 20; i++ {
			fmt.Printf("%02X", last[i])
		}
		fmt.Printf("\n")
	}

	for i := 0; i < 20; i++ {
		r := buf[i] ^ last[i]
		if r == 0 {
			continue
		}
		if (r != 0) && (i == XB_PAD) {
			decodePad(last[XB_PAD], buf[XB_PAD], &xbe)
		}
		if (r != 0) && (i == XB_DECK) {
			decodeDeck(last[XB_DECK], buf[XB_DECK], &xbe)
		}
		if (r != 0) && (i == XB_LT) {
			data = int16(buf[XB_LT])
			xbe.Code = LT
			xbe.Name = "LT"
			xbe.X = data
			xbe.Y = data
		}
		if (r != 0) && (i == XB_RT) {
			data = int16(buf[XB_RT])
			xbe.Code = RT
			xbe.Name = "RT"
			xbe.X = data
			xbe.Y = data
		}
		if (r != 0) && (i == XB_LJOY1X || i == XB_LJOY2X) {
			data = int16(int16(buf[XB_LJOY1X]) | int16(buf[XB_LJOY2X])<<8)
			if math.Abs(float64(data)) <= float64(deadzone) {
				data = 0
			}
			xbe.Code = LJOYX
			xbe.Name = "LJOYX"
			xbe.X = data
			xbe.Y = ly
			lx = data
		}
		if (r != 0) && (i == XB_LJOY1Y || i == XB_LJOY2Y) {
			data = int16(int16(buf[XB_LJOY1Y]) | int16(buf[XB_LJOY2Y])<<8)
			if math.Abs(float64(data)) <= float64(deadzone) {
				data = 0
			}
			xbe.Code = LJOYY
			xbe.Name = "LJOYY"
			xbe.Y = data
			xbe.X = lx
			ly = data
		}
		if (r != 0) && (i == XB_RJOY1X || i == XB_RJOY2X) {
			data = int16(int16(buf[XB_RJOY1X]) | int16(buf[XB_RJOY2X])<<8)
			if math.Abs(float64(data)) <= float64(deadzone) {
				data = 0
			}
			xbe.Code = RJOYX
			xbe.Name = "RJOYX"
			xbe.X = data
			xbe.Y = ly
			lx = data
		}
		if (r != 0) && (i == XB_RJOY1Y || i == XB_RJOY2Y) {
			data = int16(int16(buf[XB_RJOY1Y]) | int16(buf[XB_RJOY2Y])<<8)
			if math.Abs(float64(data)) <= float64(deadzone) {
				data = 0
			}
			xbe.Code = RJOYY
			xbe.Name = "RJOYY"
			xbe.Y = data
			xbe.X = lx
			ly = data
		}

	}
	return &xbe
}

func decodePad(last, buf byte, xbe *XBevent) {
	if buf == 0 {
		// it's a button release event
		if last&0x80 != 0 {
			xbe.Name = "RJOY_UP"
			xbe.Code = RJOY_UP
		}
		if last&0x40 != 0 {
			xbe.Name = "LJOY_UP"
			xbe.Code = LJOY_UP
		}
		if last&0x20 != 0 {
			xbe.Name = "BACK_UP"
			xbe.Code = BACK_UP
		}
		if last&0x10 != 0 {
			xbe.Name = "START_UP"
			xbe.Code = PADU_UP
		}
		if last&0x08 != 0 {
			xbe.Name = "PADR_UP"
			xbe.Code = PADR_UP
		}
		if last&0x04 != 0 {
			xbe.Name = "PADL_UP"
			xbe.Code = PADL_UP
		}
		if last&0x02 != 0 {
			xbe.Name = "PADD_UP"
			xbe.Code = PADD_UP
		}
		if last&0x01 != 0 {
			xbe.Name = "PADU_UP"
			xbe.Code = PADU_UP
		}
	} else {
		// it's a button press event
		if buf&0x80 != 0 {
			xbe.Name = "RJOY_DOWN"
			xbe.Code = RJOY_DOWN
		}
		if buf&0x40 != 0 {
			xbe.Name = "LJOY_DOWN"
			xbe.Code = LJOY_DOWN
		}
		if buf&0x20 != 0 {
			xbe.Name = "BACK_DOWN"
			xbe.Code = BACK_DOWN
		}
		if buf&0x10 != 0 {
			xbe.Name = "START_DOWN"
			xbe.Code = START_DOWN
		}
		if buf&0x08 != 0 {
			xbe.Name = "PADR_DOWN"
			xbe.Code = PADR_DOWN
		}
		if buf&0x04 != 0 {
			xbe.Name = "PADL_DOWN"
			xbe.Code = PADL_DOWN
		}
		if buf&0x02 != 0 {
			xbe.Name = "PADD_DOWN"
			xbe.Code = PADD_DOWN
		}
		if buf&0x01 != 0 {
			xbe.Name = "PADU_DOWN"
			xbe.Code = PADU_DOWN
		}
	}

}

func decodeDeck(last, buf byte, xbe *XBevent) {
	if buf == 0 {
		// it's a button release event
		if last&0x80 != 0 {
			xbe.Name = "Y_UP"
			xbe.Code = Y_UP
		}
		if last&0x40 != 0 {
			xbe.Name = "X_UP"
			xbe.Code = Y_UP
		}
		if last&0x20 != 0 {
			xbe.Name = "B_UP"
			xbe.Code = B_UP
		}
		if last&0x10 != 0 {
			xbe.Name = "A_UP"
			xbe.Code = A_UP
		}
		if last&0x08 != 0 {
			xbe.Name = "DECK_08_UP"
			xbe.Code = 0
		}
		if last&0x04 != 0 {
			xbe.Name = "GUIDE_UP"
			xbe.Code = GUIDE_UP
		}
		if last&0x02 != 0 {
			xbe.Name = "RTOP_UP"
			xbe.Code = RTOP_UP
		}
		if last&0x01 != 0 {
			xbe.Name = "LTOP_UP"
			xbe.Code = LTOP_UP
		}
	} else {
		// it's a button press event
		if buf&0x80 != 0 {
			xbe.Name = "Y_DOWN"
			xbe.Code = Y_DOWN
		}
		if buf&0x40 != 0 {
			xbe.Name = "X_DOWN"
			xbe.Code = X_DOWN
		}
		if buf&0x20 != 0 {
			xbe.Name = "B_DOWN"
			xbe.Code = B_DOWN
		}
		if buf&0x10 != 0 {
			xbe.Name = "A_DOWN"
			xbe.Code = A_DOWN
		}
		if buf&0x08 != 0 {
			xbe.Name = "DECK_08_DOWN"
			xbe.Code = 0
		}
		if buf&0x04 != 0 {
			xbe.Name = "GUIDE_DOWN"
			xbe.Code = GUIDE_DOWN
		}
		if buf&0x02 != 0 {
			xbe.Name = "RTOP_DOWN"
			xbe.Code = RTOP_DOWN
		}
		if buf&0x01 != 0 {
			xbe.Name = "LTOP_DOWN"
			xbe.Code = LTOP_DOWN
		}
	}

}

func initBuffer(b []byte) {
	//00 14 00 00 00 00 00 F7 00 02 00 FE 00 00 00 00 00 00 00 00
	b[1] = 0x14
	b[7] = 0xF7
	b[9] = 0x02
	b[11] = 0xFE

	/*
		for i := 0; i < 20; i++ {
			fmt.Printf("%02X", b[i])
		}
		fmt.Printf("\n")
	*/
}

func openXB(ctx *gousb.Context) io.Reader {
	dev, err := ctx.OpenDeviceWithVIDPID(vid, pid)
	if err != nil {
		log.Fatalf("Could not open a device: %v", err)
	}

	intf, done, err := dev.DefaultInterface()
	if err != nil {
		log.Fatalf("%s.DefaultInterface(): %v", dev, err)
	}
	defer done()

	ep, err := intf.InEndpoint(endpoint)
	if err != nil {
		log.Fatalf("dev.InEndpoint(): %s", err)
	}

	var rdr io.Reader
	rdr = ep
	if bufSize > 1 {
		s, err := ep.NewStream(size, bufSize)
		if err != nil {
			log.Fatalf("ep.NewStream(): %v", err)
		}
		defer s.Close()
		rdr = s
	}
	return rdr
}
