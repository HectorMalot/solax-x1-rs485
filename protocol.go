package solaxx1rs485

import (
	"errors"
	"fmt"
)

const (
	ControlCodeRegister byte = 0x10
	ControlCodeRead     byte = 0x11
	ControlCodeWrite    byte = 0x12
	ControlCodeExecute  byte = 0x13

	StatusACK   byte = 0x06
	StatusNOACK byte = 0x15
)

var (
	ErrMaxDataSizeExceeded    = errors.New("Exceeded max data length is 255 bytes")
	ErrInvalidBody            = errors.New("Could not parse body into valid packet")
	ErrUnexpectedControlCode  = errors.New("Unexpected Control Code")
	ErrUnexpectedFunctionCode = errors.New("Unexpected Function Code")
)

/*
Default packet format for Solax X1 air.
It uses an RS485 protocol that with some resemblance to modbus

## Packet format:
Header				2		0xAA55  // Always same value
Source Address		2		0xXX00  // Typically 0x0000
Destination Address	2		0x00XX  // As set, default 0x0000
Control code		1		0xXX    // Depends on command
Function code 		1		0xXX	// Depends on command
Data lenght			1		0xXX	// Data length, can be 0
Data0..N			N
Checksum			2		sum of bytes above
*/
type Packet struct {
	Header       uint16
	Source       uint16
	Destination  uint16
	ControlCode  byte
	FunctionCode byte
	Data         []byte
}

func (p *Packet) Bytes() ([]byte, error) {
	if len(p.Data) > 0xFF { // max data length is 255 bytes because length field is 1 byte
		return nil, ErrMaxDataSizeExceeded
	}

	body := []byte{
		bytesFromUint16(p.Header)[0],
		bytesFromUint16(p.Header)[1],
		bytesFromUint16(p.Source)[0],
		bytesFromUint16(p.Source)[1],
		bytesFromUint16(p.Destination)[0],
		bytesFromUint16(p.Destination)[1],
		p.ControlCode,
		p.FunctionCode,
		uint8(len(p.Data)),
	}
	body = append(body, p.Data...)
	body = append(body, bytesFromUint16(checksum(body))...)
	return body, nil
}

func DefaultPacket() *Packet {
	return &Packet{
		Header:      0xAA55,
		Source:      0x0000,
		Destination: 0x0000,
		Data:        []byte{},
	}
}

func ParsePacket(res []byte) (*Packet, error) {
	if len(res) < 11 {
		return nil, fmt.Errorf("%w: minimum packet size is 11 bytes, got %d bytes", ErrInvalidBody, len(res))
	}
	dataLength := int(res[8])
	if len(res) != int(dataLength)+11 {
		return nil, fmt.Errorf("%w: packet specifies length of %d, but got %d instead", ErrInvalidBody, dataLength+11, len(res))
	}
	cs := *(*[2]byte)(res[len(res)-2:])
	if checksum(res[:len(res)-2]) != uint16FromBytes(cs) {
		return nil, fmt.Errorf("%w: checksum mismatch: Packet specifies checksum of %X, got %X instead", ErrInvalidBody, uint16FromBytes(cs), checksum(res[:len(res)-2]))
	}
	if uint16FromBytes(*(*[2]byte)(res[0:2])) != 0xAA55 {
		return nil, fmt.Errorf("%w: header mismatch: Expected 0xAA55, got %X", ErrInvalidBody, uint16FromBytes(*(*[2]byte)(res[0:2])))
	}

	p := &Packet{}
	p.Header = uint16FromBytes(*(*[2]byte)(res[0:2]))
	p.Source = uint16FromBytes(*(*[2]byte)(res[2:4]))
	p.Destination = uint16FromBytes(*(*[2]byte)(res[4:6]))
	p.ControlCode = res[6]
	p.FunctionCode = res[7]
	p.Data = res[9 : 9+dataLength]
	return p, nil
}

func checksum(body []byte) uint16 {
	var cs uint16 = 0
	for _, b := range body {
		cs += uint16(b)
	}
	return cs
}

/*
-------------------------------------------------------------------------------
----- Calls related to inverter registration
-------------------------------------------------------------------------------


These commands are all related to setting up a new inverter.
Function codes:
0x00 	Client -> Inverter 	QUERY UNREGISTERED
0x80 	Inverter -> Client 	RESPOND WITH SERIAL
0x01	Client -> Inverter  SET ADDRESS
0x81	Inverter -> Client	CONFIRM ADDRESS
0x02	Client -> Inverter	REMOVE ADDRESS
0x82	Inverter -> Client	CONFIRM REMOVAL
0x03	Client -> Inverter	RECONNECT REMOVED (?)
0x04	Client -> Inverter  REREGISTER (?)
*/

// 0x00
func UnregisteredInverterRequest() *Packet {
	p := DefaultPacket()
	p.ControlCode = ControlCodeRegister
	p.FunctionCode = 0x00
	return p
}

// 0x80
func ParseUnregisteredInverterResponse(body []byte) (UnregisteredInverterResponse, error) {
	p, err := ParsePacket(body)
	if err != nil {
		return UnregisteredInverterResponse{}, err
	}

	// Control code should be 0x10
	if p.ControlCode != ControlCodeRegister {
		return UnregisteredInverterResponse{}, fmt.Errorf("%w: Expected 0x10, got %X", ErrUnexpectedControlCode, p.ControlCode)
	}
	// Function code should be 0x80
	if p.FunctionCode != 0x80 {
		return UnregisteredInverterResponse{}, fmt.Errorf("%w: Expected 0x80, got %X", ErrUnexpectedFunctionCode, p.FunctionCode)
	}

	return UnregisteredInverterResponse{Serial: p.Data}, nil
}

type UnregisteredInverterResponse struct {
	Serial []byte
}

// 0x01
func RegisterInverterRequest(serial []byte, address byte) *Packet {
	p := DefaultPacket()
	p.ControlCode = ControlCodeRegister
	p.FunctionCode = 0x01
	p.Data = append(serial, address)
	return p
}

//0x81
func ParseRegisterInverterResponse(body []byte) error {
	p, err := ParsePacket(body)
	if err != nil {
		return err
	}

	// Control code should be 0x10
	if p.ControlCode != ControlCodeRegister {
		return fmt.Errorf("%w: Expected 0x10, got %X", ErrUnexpectedControlCode, p.ControlCode)
	}
	// Function code should be 0x81
	if p.FunctionCode != 0x81 {
		return fmt.Errorf("%w: Expected 0x81, got %X", ErrUnexpectedFunctionCode, p.FunctionCode)
	}
	// Response should be ACK
	if len(p.Data) != 1 {
		return fmt.Errorf("%w: Expected data length 1, got %d", ErrInvalidBody, len(p.Data))
	}
	if p.Data[0] == StatusNOACK {
		return fmt.Errorf("got NOACK")
	}
	if p.Data[0] != StatusACK {
		return fmt.Errorf("%w: expected ACK (%X) or NOACK (%X), got %X", ErrInvalidBody, StatusACK, StatusNOACK, p.Data[0])
	}

	return nil
}

// 0x02
func UnregisterInverterRequest(serial []byte, address byte) *Packet {
	p := DefaultPacket()
	p.ControlCode = ControlCodeRegister
	p.FunctionCode = 0x02
	p.Data = append(serial, address)
	return p
}

//0x82
func ParseUnregisterInverterResponse(body []byte) error {
	p, err := ParsePacket(body)
	if err != nil {
		return err
	}

	// Control code should be 0x10
	if p.ControlCode != ControlCodeRegister {
		return fmt.Errorf("%w: Expected 0x10, got %X", ErrUnexpectedControlCode, p.ControlCode)
	}
	// Function code should be 0x81
	if p.FunctionCode != 0x82 {
		return fmt.Errorf("%w: Expected 0x81, got %X", ErrUnexpectedFunctionCode, p.FunctionCode)
	}
	// Response should be ACK
	if len(p.Data) != 1 {
		return fmt.Errorf("%w: Expected data length 1, got %d", ErrInvalidBody, len(p.Data))
	}
	if p.Data[0] == StatusNOACK {
		return fmt.Errorf("got NOACK")
	}
	if p.Data[0] != StatusACK {
		return fmt.Errorf("%w: expected ACK (%X) or NOACK (%X), got %X", ErrInvalidBody, StatusACK, StatusNOACK, p.Data[0])
	}

	return nil
}

/*
-------------------------------------------------------------------------------
----- Calls related to inverter information
-------------------------------------------------------------------------------
Control code: 0x11
Function codes

0x02	Client -> Inverter	Query normal info
0x82	Inverter -> Client	normal info response
0x03	Client -> Inverter	Query inverter specifications
0x83	Inverter -> Client  inverter specifications response
0x04	Client -> Inverter	Query config
0x84	Inverter -> Client	config response
*/

// 0x02
func NormalInfoRequest(address byte) *Packet {
	p := DefaultPacket()
	p.Destination = uint16FromBytes([2]byte{0x00, address})
	p.ControlCode = ControlCodeRead
	p.FunctionCode = 0x02
	return p
}

// 0x82
func ParseNormalInfoResponse(body []byte) (NormalInfoResponse, error) {
	p, err := ParsePacket(body)
	if err != nil {
		return NormalInfoResponse{}, err
	}

	// Control code should be 0x11
	if p.ControlCode != ControlCodeRead {
		return NormalInfoResponse{}, fmt.Errorf("%w: Expected %X, got %X", ErrUnexpectedControlCode, ControlCodeRead, p.ControlCode)
	}
	// Function code should be 0x82
	if p.FunctionCode != 0x82 {
		return NormalInfoResponse{}, fmt.Errorf("%w: Expected 0x82, got %X", ErrUnexpectedFunctionCode, p.FunctionCode)
	}

	return NormalInfoResponseFromData(p.Data)
}

type NormalInfoResponse struct {
	Temperature      uint16 // Celsius
	EnergyToday      uint16 // 0.1kWh
	Vpv1             uint16 // 0.1V
	Vpv2             uint16 // 0.1V
	Apv1             uint16 // 0.1A
	Apv2             uint16 // 0.1A
	Iac              uint16 // 0.1A
	Vac              uint16 // 0.1V
	Frequency        uint16 // 0.01Hz
	Power            uint16 // 1W
	_                uint16 // Unused
	EnergyTotal      uint32 // 0.1kWh
	TimeTotal        uint32 // hours
	Mode             uint16 // Inverter mode (0: Wait, 1: Check, 2: Normal, 3: Fault, 4: Permanent Fault, 5: Update, 6: Selftest)
	GridVoltFault    uint16 // 0.1V Grid voltage fault value
	GridFreqFault    uint16 // 0.01Hz Grid frequency fault value
	DCIFault         uint16 // mA, DJ injection fault value
	TemperatureFault uint16 // Temperature fault value
	PV1Fault         uint16 // 0.1V PV1 voltage fault value
	PV2Fault         uint16 // 0.1V PV2 voltage fault value
	GFCFault         uint16 // mA, GFC fault value
	ErrMessage       uint32 // Error message code
}

type NormalizedNormalInfoResponse struct {
	Temperature      uint16   // Celsius
	EnergyToday      float64  // 0.1kWh -> kWh
	Vpv1             float64  // 0.1V -> V
	Vpv2             float64  // 0.1V -> V
	Apv1             float64  // 0.1A -> A
	Apv2             float64  // 0.1A -> A
	Iac              float64  // 0.1A -> A
	Vac              float64  // 0.1V -> V
	Frequency        float64  // 0.01Hz -> Hz
	Power            uint16   // 1W
	_                uint16   // Unused
	EnergyTotal      float64  // 0.1kWh
	TimeTotal        uint32   // hours
	Mode             string   // Inverter mode (0: Wait, 1: Check, 2: Normal, 3: Fault, 4: Permanent Fault, 5: Update, 6: Selftest)
	GridVoltFault    float64  // 0.1V Grid voltage fault value -> V
	GridFreqFault    float64  // 0.01Hz Grid frequency fault value -> Hz
	DCIFault         float64  // mA, DJ injection fault value -> A
	TemperatureFault float64  // Temperature fault value
	PV1Fault         float64  // 0.1V PV1 voltage fault value -> V
	PV2Fault         float64  // 0.1V PV2 voltage fault value -> V
	GFCFault         float64  // mA, GFC fault value -> A
	ErrMessage       []string // Error message code
}

/*
ErrMessage details:
//BYTE0
Uint16 TzProtectFault:1;//0
Uint16 MainsLostFault:1;//1
Uint16 GridVoltFault:1;//2
Uint16 GridFreqFault:1;//3
Uint16 PLLLostFault:1;//4
Uint16 BusVoltFault:1;//5
Uint16 BIT06:1;//6
Uint16 OciFault:1;//7 OciFault;
//BYTE1
Uint16 Dci_OCP_Fault:1;//8
Uint16 ResidualCurrentFault:1;//9
Uint16 PvVoltFault:1;//10
Uint16 Ac10Mins_Voltage_Fault:1;//11
Uint16 IsolationFault:1;//12
Uint16 TemperatureOverFault:1;//13
Uint16 FanFault:1;//14
Uint16 bit15:1;//15
//BYTE2
Uint16 SpiCommsFault:1;//16
Uint16 SciCommsFault:1;//17
Uint16 BIT18:1;//18
Uint16 InputConfigFault:1;//19
Uint16 EepromFault:1;//20
Uint16 RelayFault:1;//21
Uint16 SampleConsistenceFault:1;//22
Uint16 ResidualCurrent_DeviceFault:1;//23
//BYTE3
Uint16 BIT24:1;//24
Uint16 BIT25:1;//25
Uint16 BIT26:1;//26
Uint16 BIT27:1;//27
Uint16 BIT28:1;//28
Uint16 DCI_DeviceFault:1;//29
Uint16 OtherDeviceFault:1;//30
Uint16 BIT31:1;//31
*/

func NormalizeInfoResponse(in NormalInfoResponse) NormalizedNormalInfoResponse {
	modes := map[uint16]string{0: "Wait", 1: "Check", 2: "Normal", 3: "Fault", 4: "Permanent Fault", 5: "Update", 6: "Selftest"}
	res := NormalizedNormalInfoResponse{
		Temperature:      in.Temperature,
		EnergyToday:      float64(in.EnergyToday) / 10,
		Vpv1:             float64(in.Vpv1) / 10,
		Vpv2:             float64(in.Vpv2) / 10,
		Apv1:             float64(in.Apv1) / 10,
		Apv2:             float64(in.Apv2) / 10,
		Iac:              float64(in.Iac) / 10,
		Vac:              float64(in.Vac) / 10,
		Frequency:        float64(in.Frequency) / 100,
		Power:            in.Power,
		EnergyTotal:      float64(in.EnergyTotal) / 10,
		TimeTotal:        in.TimeTotal,
		Mode:             modes[in.Mode],
		GridVoltFault:    float64(in.GridVoltFault) / 10,
		GridFreqFault:    float64(in.GridFreqFault) / 100,
		DCIFault:         float64(in.DCIFault) / 1000,
		TemperatureFault: float64(in.TemperatureFault),
		PV1Fault:         float64(in.PV1Fault) / 10,
		PV2Fault:         float64(in.PV2Fault) / 10,
		GFCFault:         float64(in.GFCFault) / 1000,
		ErrMessage:       []string{},
	}

	// parse error messages
	if in.ErrMessage&uint32(2147483648) == uint32(2147483648) {
		res.ErrMessage = append(res.ErrMessage, "TzProtectFault")
	}
	if in.ErrMessage&(uint32(2147483648)>>1) == uint32(2147483648)>>1 {
		res.ErrMessage = append(res.ErrMessage, "MainsLostFault")
	}
	if in.ErrMessage&(uint32(2147483648)>>2) == uint32(2147483648)>>2 {
		res.ErrMessage = append(res.ErrMessage, "GridVoltFault")
	}
	if in.ErrMessage&(uint32(2147483648)>>3) == uint32(2147483648)>>3 {
		res.ErrMessage = append(res.ErrMessage, "GridFreqFault")
	}
	if in.ErrMessage&(uint32(2147483648)>>4) == uint32(2147483648)>>4 {
		res.ErrMessage = append(res.ErrMessage, "PLLLostFault")
	}
	if in.ErrMessage&(uint32(2147483648)>>5) == uint32(2147483648)>>5 {
		res.ErrMessage = append(res.ErrMessage, "BusVoltFault")
	}
	if in.ErrMessage&(uint32(2147483648)>>6) == uint32(2147483648)>>6 {
		res.ErrMessage = append(res.ErrMessage, "BIT06")
	}
	if in.ErrMessage&(uint32(2147483648)>>7) == uint32(2147483648)>>7 {
		res.ErrMessage = append(res.ErrMessage, "OciFault")
	}
	if in.ErrMessage&(uint32(2147483648)>>8) == uint32(2147483648)>>8 {
		res.ErrMessage = append(res.ErrMessage, "Dci_OCP_Fault")
	}
	if in.ErrMessage&(uint32(2147483648)>>9) == uint32(2147483648)>>9 {
		res.ErrMessage = append(res.ErrMessage, "ResidualCurrentFault")
	}
	if in.ErrMessage&(uint32(2147483648)>>10) == uint32(2147483648)>>10 {
		res.ErrMessage = append(res.ErrMessage, "PvVoltFault")
	}
	if in.ErrMessage&(uint32(2147483648)>>11) == uint32(2147483648)>>11 {
		res.ErrMessage = append(res.ErrMessage, "Ac10Mins_Voltage_Fault")
	}
	if in.ErrMessage&(uint32(2147483648)>>12) == uint32(2147483648)>>12 {
		res.ErrMessage = append(res.ErrMessage, "IsolationFault")
	}
	if in.ErrMessage&(uint32(2147483648)>>13) == uint32(2147483648)>>13 {
		res.ErrMessage = append(res.ErrMessage, "TemperatureOverFault")
	}
	if in.ErrMessage&(uint32(2147483648)>>14) == uint32(2147483648)>>14 {
		res.ErrMessage = append(res.ErrMessage, "FanFault")
	}
	if in.ErrMessage&(uint32(2147483648)>>15) == uint32(2147483648)>>15 {
		res.ErrMessage = append(res.ErrMessage, "bit15")
	}
	if in.ErrMessage&(uint32(2147483648)>>16) == uint32(2147483648)>>16 {
		res.ErrMessage = append(res.ErrMessage, "SpiCommsFault")
	}
	if in.ErrMessage&(uint32(2147483648)>>17) == uint32(2147483648)>>17 {
		res.ErrMessage = append(res.ErrMessage, "SciCommsFault")
	}
	if in.ErrMessage&(uint32(2147483648)>>18) == uint32(2147483648)>>18 {
		res.ErrMessage = append(res.ErrMessage, "BIT18")
	}
	if in.ErrMessage&(uint32(2147483648)>>19) == uint32(2147483648)>>19 {
		res.ErrMessage = append(res.ErrMessage, "InputConfigFault")
	}
	if in.ErrMessage&(uint32(2147483648)>>20) == uint32(2147483648)>>20 {
		res.ErrMessage = append(res.ErrMessage, "EepromFault")
	}
	if in.ErrMessage&(uint32(2147483648)>>21) == uint32(2147483648)>>21 {
		res.ErrMessage = append(res.ErrMessage, "RelayFault")
	}
	if in.ErrMessage&(uint32(2147483648)>>22) == uint32(2147483648)>>22 {
		res.ErrMessage = append(res.ErrMessage, "SampleConsistenceFault")
	}
	if in.ErrMessage&(uint32(2147483648)>>23) == uint32(2147483648)>>23 {
		res.ErrMessage = append(res.ErrMessage, "ResidualCurrent_DeviceFault")
	}
	if in.ErrMessage&(uint32(2147483648)>>24) == uint32(2147483648)>>24 {
		res.ErrMessage = append(res.ErrMessage, "BIT24")
	}
	if in.ErrMessage&(uint32(2147483648)>>25) == uint32(2147483648)>>25 {
		res.ErrMessage = append(res.ErrMessage, "BIT25")
	}
	if in.ErrMessage&(uint32(2147483648)>>26) == uint32(2147483648)>>26 {
		res.ErrMessage = append(res.ErrMessage, "BIT26")
	}
	if in.ErrMessage&(uint32(2147483648)>>27) == uint32(2147483648)>>27 {
		res.ErrMessage = append(res.ErrMessage, "BIT27")
	}
	if in.ErrMessage&(uint32(2147483648)>>28) == uint32(2147483648)>>28 {
		res.ErrMessage = append(res.ErrMessage, "BIT28")
	}
	if in.ErrMessage&(uint32(2147483648)>>29) == uint32(2147483648)>>29 {
		res.ErrMessage = append(res.ErrMessage, "DCI_DeviceFault")
	}
	if in.ErrMessage&(uint32(2147483648)>>30) == uint32(2147483648)>>30 {
		res.ErrMessage = append(res.ErrMessage, "OtherDeviceFault")
	}
	if in.ErrMessage&(uint32(2147483648)>>31) == uint32(2147483648)>>31 {
		res.ErrMessage = append(res.ErrMessage, "BIT31")
	}
	return res
}

func NormalInfoResponseFromData(data []byte) (NormalInfoResponse, error) {
	if len(data) != 50 {
		return NormalInfoResponse{}, ErrInvalidBody
	}
	result := NormalInfoResponse{
		Temperature:      uint16FromBytes(*(*[2]byte)(data[0:2])),
		EnergyToday:      uint16FromBytes(*(*[2]byte)(data[2:4])),
		Vpv1:             uint16FromBytes(*(*[2]byte)(data[4:6])),
		Vpv2:             uint16FromBytes(*(*[2]byte)(data[6:8])),
		Apv1:             uint16FromBytes(*(*[2]byte)(data[8:10])),
		Apv2:             uint16FromBytes(*(*[2]byte)(data[10:12])),
		Iac:              uint16FromBytes(*(*[2]byte)(data[12:14])),
		Vac:              uint16FromBytes(*(*[2]byte)(data[14:16])),
		Frequency:        uint16FromBytes(*(*[2]byte)(data[16:18])),
		Power:            uint16FromBytes(*(*[2]byte)(data[18:20])),
		EnergyTotal:      uint32FromBytes(*(*[4]byte)(data[22:26])),
		TimeTotal:        uint32FromBytes(*(*[4]byte)(data[26:30])),
		Mode:             uint16FromBytes(*(*[2]byte)(data[30:32])),
		GridVoltFault:    uint16FromBytes(*(*[2]byte)(data[32:34])),
		GridFreqFault:    uint16FromBytes(*(*[2]byte)(data[34:36])),
		DCIFault:         uint16FromBytes(*(*[2]byte)(data[36:38])),
		TemperatureFault: uint16FromBytes(*(*[2]byte)(data[38:40])),
		PV1Fault:         uint16FromBytes(*(*[2]byte)(data[40:42])),
		PV2Fault:         uint16FromBytes(*(*[2]byte)(data[42:44])),
		GFCFault:         uint16FromBytes(*(*[2]byte)(data[44:46])),
		ErrMessage:       uint32FromBytes(*(*[4]byte)(data[46:50])),
	}
	return result, nil
}

// 0x03
func InverterInfoRequest(address byte) *Packet {
	p := DefaultPacket()
	p.Destination = uint16FromBytes([2]byte{0x00, address})
	p.ControlCode = ControlCodeRead
	p.FunctionCode = 0x03
	return p
}

// 0x83
func ParseInverterInfoResponse(body []byte) (InverterInfoResponse, error) {
	p, err := ParsePacket(body)
	if err != nil {
		return InverterInfoResponse{}, err
	}

	// Control code should be 0x11
	if p.ControlCode != ControlCodeRead {
		return InverterInfoResponse{}, fmt.Errorf("%w: Expected %X, got %X", ErrUnexpectedControlCode, ControlCodeRead, p.ControlCode)
	}
	// Function code should be 0x83
	if p.FunctionCode != 0x83 {
		return InverterInfoResponse{}, fmt.Errorf("%w: Expected 0x83, got %X", ErrUnexpectedFunctionCode, p.FunctionCode)
	}

	return InverterInfoResponseFromData(p.Data)
}

type InverterInfoResponse struct {
	Phase           byte
	RatedPower      string
	FirmwareVersion string
	ModuleName      string
	FactoryName     string
	SerialNumber    string
	RatedBusVoltage string
}

func InverterInfoResponseFromData(data []byte) (InverterInfoResponse, error) {
	if len(data) < 67 {
		return InverterInfoResponse{}, fmt.Errorf("%w: expected length of 67, got %d", ErrInvalidBody, len(data))
	}
	result := InverterInfoResponse{
		Phase:           data[9],
		RatedPower:      string(data[10:16]),
		FirmwareVersion: string(data[16:21]),
		ModuleName:      string(data[21:35]),
		FactoryName:     string(data[35:49]),
		SerialNumber:    string(data[49:63]),
		RatedBusVoltage: string(data[63:67]),
	}
	return result, nil
}
