package solaxx1rs485

import (
	"log"
	"testing"
)

func TestConnection(t *testing.T) {

	log.Println("Starting Open")
	c, _ := NewClient("/dev/tty.usbserial-A10KNFUE")
	i, err := c.FindUnregisteredInverter()
	if err != nil {
		log.Fatal(err)
	}
	log.Fatalf("%+v", i)

	// c := &serial.Config{Name: "/dev/tty.usbserial-A10KNFUE", Baud: 9600, ReadTimeout: 500 * time.Millisecond}
	// s, err := serial.OpenPort(c)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// log.Println("Starting Write")
	// n, err := s.Write([]byte("test"))
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// log.Println("Completed Write")

	// log.Println("Starting Read")
	// buf := make([]byte, 128)
	// n, err = s.Read(buf)
	// if err != nil {
	// 	log.Fatalf("got error while reading: %s", err)
	// }
	// log.Printf("%q", buf[:n])
	// log.Println("Completed Read")
	t.FailNow()
}
