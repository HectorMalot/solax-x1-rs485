# Solax X1 Air RS485 communication client in Go

Simple client to communicate over RS485 with Solax X1 inverters. Based on the v1.7 spec.

## Typical usage:
* You need to register your inverter first (this is 1 time per inverter)
	1. Connect your Solax device directly to your PC (usually with a USB-RS485 converter)
	2. Run `solax -d /dev/yourserialdevicehere find`. This will return a Serial. Copy it.
	3. Register the inverter using `solax -a <address> -s <serial> -d /dev/yourserialdevicehere register`, where address is a unique number between 1 and 255

* Get the information from your inverter:
	1. Run `solax -d /dev/yourserialdevicehere -a <address> info`
	2. use the `--json` flag to output JSON