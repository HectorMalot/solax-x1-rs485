package solaxx1rs485

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/tarm/serial"
)

var (
	ErrIncompleteRead  = errors.New("Failed to read read body")
	ErrIncompleteWrite = errors.New("Failed to write full body")
	ErrNoInverter      = errors.New("No inverter responded to call")
)

type Connection interface {
	io.ReadWriteCloser
	Flush() error
}

type Client struct {
	Conn         Connection
	LastResponse []byte
	WaitTime     time.Duration // Time to wait for a response after sending
}

func NewClient(device string) (*Client, error) {
	// Default connection parameters: https://tasmota.github.io/docs/_media/solax-x1/SolaxPower_Single_Phase_External_Communication_Protocol_X1_V1.7.pdf
	// Speeds: 9600bps
	// Data bit: 8
	// Parity: none
	// Stop bit: 1
	// Example device: "/dev/tty.usbserial-A10KNFUE"
	c := &serial.Config{Name: device, Baud: 9600, ReadTimeout: 500 * time.Millisecond}
	conn, err := serial.OpenPort(c)
	if err != nil {
		return nil, err
	}
	return NewClientWithConnection(conn)
}

func NewClientWithConnection(conn Connection) (*Client, error) {
	return &Client{Conn: conn, WaitTime: 250 * time.Millisecond}, nil
}

/*
To be implemented
Registration:
* [x] FindUnregisteredInverters
* RegisterInverter
* UnregisterInverters
Information (Read):
* QueryInfo
* QueryID
* QueryConfig
[Maybe Later] Write Config
* WriteConfig
*/

type Inverter struct {
	Serial  []byte
	Address byte
}

/*
-------------------------------------------------------------------------------
----- Calls related to inverter registration
-------------------------------------------------------------------------------
*/

// FindUnregisteredInverter returns the first unregistered inverter (address 0x00)
// Use RegisterInverter afterwards to set an address for the inverter
func (c *Client) FindUnregisteredInverter() (*Inverter, error) {
	err := c.Conn.Flush()
	if err != nil {
		return nil, err
	}

	// Send the request
	req, err := UnregisteredInverterRequest().Bytes()
	if err != nil {
		return nil, err
	}
	err = c.Send(req)
	if err != nil {
		return nil, err
	}

	// Get and handle the response
	resp, err := c.Read()
	if err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, ErrNoInverter
	}
	r, err := ParseUnregisteredInverterResponse(resp)
	if err != nil {
		return nil, err
	}

	return &Inverter{Serial: r.Serial}, nil
}

// RegisterInverter Sets the bus address for an unregistered inverter
func (c *Client) RegisterInverter(inverter *Inverter, address byte) error {
	if inverter == nil {
		return fmt.Errorf("Inverter must not be nil")
	}

	err := c.Conn.Flush()
	if err != nil {
		return err
	}

	// Send the request
	req, err := RegisterInverterRequest(inverter.Serial, address).Bytes()
	if err != nil {
		return err
	}
	err = c.Send(req)
	if err != nil {
		return err
	}

	// Get and handle the response
	resp, err := c.Read()
	if err != nil {
		return err
	}
	err = ParseRegisterInverterResponse(resp)
	if err != nil {
		return err
	}

	inverter.Address = address
	return nil
}

// UnregisterInverter resets the inverter address (becomes 0x00)
func (c *Client) UnregisterInverter(inverter *Inverter) error {
	if inverter == nil {
		return fmt.Errorf("Inverter must not be nil")
	}

	err := c.Conn.Flush()
	if err != nil {
		return err
	}

	// Send the request
	req, err := UnregisterInverterRequest(inverter.Serial, inverter.Address).Bytes()
	if err != nil {
		return err
	}
	err = c.Send(req)
	if err != nil {
		return err
	}

	// Get and handle the response
	resp, err := c.Read()
	if err != nil {
		return err
	}
	err = ParseUnregisterInverterResponse(resp)
	if err != nil {
		return err
	}

	inverter.Address = 0x00
	return nil
}

/*
-------------------------------------------------------------------------------
----- Calls related to inverter information
-------------------------------------------------------------------------------
*/

func (c *Client) GetInfo(inverter *Inverter) (*NormalInfoResponse, error) {
	if inverter == nil {
		return nil, fmt.Errorf("Inverter must not be nil")
	}

	err := c.Conn.Flush()
	if err != nil {
		return nil, err
	}

	// Send the request
	req, err := NormalInfoRequest(inverter.Address).Bytes()
	if err != nil {
		return nil, err
	}
	err = c.Send(req)
	if err != nil {
		return nil, err
	}

	// Get and handle the response
	resp, err := c.Read()
	if err != nil {
		return nil, err
	}
	result, err := ParseNormalInfoResponse(resp)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) GetInverterInfo(inverter *Inverter) (*InverterInfoResponse, error) {
	if inverter == nil {
		return nil, fmt.Errorf("Inverter must not be nil")
	}

	err := c.Conn.Flush()
	if err != nil {
		return nil, err
	}

	// Send the request
	req, err := InverterInfoRequest(inverter.Address).Bytes()
	if err != nil {
		return nil, err
	}
	err = c.Send(req)
	if err != nil {
		return nil, err
	}

	// Get and handle the response
	resp, err := c.Read()
	if err != nil {
		return nil, err
	}
	result, err := ParseInverterInfoResponse(resp)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

/*
-------------------------------------------------------------------------------
----- COMMUNICATION
-------------------------------------------------------------------------------
*/

func (c *Client) Send(req []byte) error {
	n, err := c.Conn.Write(req)
	if err != nil {
		return err
	}
	if n != len(req) {
		return ErrIncompleteWrite
	}
	return nil
}

func (c *Client) Read() ([]byte, error) {
	time.Sleep(c.WaitTime)
	response, err := io.ReadAll(c.Conn)
	c.LastResponse = response
	return response, err
}
