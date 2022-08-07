package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	solax "github.com/hectormalot/solax-x1-rs485"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// used for flags
var (
	outputJson bool
	verbose    bool
	device     string
	address    int
	serial     []byte
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&device, "device", "d", "", "Serial device for communication")
	rootCmd.PersistentFlags().IntVarP(&address, "address", "a", 0x00, "Address on which to connect with Solax inverter (1..255)")
	rootCmd.PersistentFlags().BoolVarP(&outputJson, "json", "j", false, "Output results as JSON")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "")
	rootCmd.MarkFlagRequired("device")
	registerCmd.PersistentFlags().BytesHexVarP(&serial, "serial", "s", nil, "Inverter serial")
	registerCmd.MarkFlagRequired("serial")
	registerCmd.MarkFlagRequired("address")
	rootCmd.AddCommand(findCmd)
	rootCmd.AddCommand(registerCmd)
	rootCmd.AddCommand(unregisterCmd)
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(inverterInfoCmd)
}

func main() {
	Execute()
}

var rootCmd = &cobra.Command{
	Use:   "solax",
	Short: "Solax reads data from a Solax X1 air",
	Long: `Solax reads data from a Solax X1 air using an RS485 connection.

Typical usage:
A. You need to register your inverter first (this is 1 time per inverter)
	1. Connect your Solax device directly to your PC (usually with a USB-RS485 converter)
	2. Run 'solax -d /dev/yourserialdevicehere find'. This will return a Serial. Copy it.
	3. Register the inverter using 'solax -a <address> -s <serial> -d /dev/yourserialdevicehere register', where address is a unique number between 1 and 255

B. Get the information from your inverter:
	1. Run 'solax -d /dev/yourserialdevicehere -a <address> info'
	2. use the --json flag to output JSON
	`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var findCmd = &cobra.Command{
	Use:   "find",
	Short: "Finds the next unregistered Solax X1 device on the bus",
	Run:   Find,
}
var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Register an inverter",
	Run:   Register,
}
var unregisterCmd = &cobra.Command{
	Use:   "unregister",
	Short: "Remove the registration from an inverter",
	Run:   Unregister,
}
var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Get real-time inverter information",
	Run:   Info,
}
var inverterInfoCmd = &cobra.Command{
	Use:   "deviceinfo",
	Short: "Get inverter device details",
	Run:   DeviceInfo,
}

func Find(cmd *cobra.Command, args []string) {
	client, err := solax.NewClient(device)
	fatalIfError(err)
	inv, err := client.FindUnregisteredInverter()
	if verbose {
		log.Printf("Raw response: %X", client.LastResponse)
	}

	if err != nil {
		if errors.Is(err, solax.ErrNoInverter) {
			log.Print("No unregistered inverters found")
			os.Exit(0)
		}
		log.Fatal(err)
	}
	log.Printf("Found inverter\nSerial: %X", inv.Serial)

}
func Register(cmd *cobra.Command, args []string) {
	if address < 1 || address > 255 {
		log.Fatal("Address must be between 1-255")
	}
	if len(serial) < 10 {
		log.Fatal("You need to provide a valid serial")
	}

	client, err := solax.NewClient(device)
	fatalIfError(err)
	inv := &solax.Inverter{Serial: serial, Address: 0x00}

	err = client.RegisterInverter(inv, byte(address))
	fatalIfError(err)
	if verbose {
		log.Printf("Raw response: %X", client.LastResponse)
	}

	log.Printf("Inverter registered with address %X", address)
}
func Unregister(cmd *cobra.Command, args []string) {
	if address < 1 || address > 255 {
		log.Fatal("Address must be between 1-255")
	}

	client, err := solax.NewClient(device)
	fatalIfError(err)
	inv := &solax.Inverter{Serial: serial, Address: byte(address)}

	err = client.UnregisterInverter(inv)
	fatalIfError(err)
	if verbose {
		log.Printf("Raw response: %X", client.LastResponse)
	}

	log.Printf("Inverter with address %X no longer registered", address)
}

func Info(cmd *cobra.Command, args []string) {
	if address < 0 || address > 255 {
		log.Fatal("Address must be between 1-255")
	}

	client, err := solax.NewClient(device)
	fatalIfError(err)

	inv := &solax.Inverter{Address: byte(address)}
	info, err := client.GetInfo(inv)
	fatalIfError(err)

	if verbose {
		log.Printf("Raw response: %X", client.LastResponse)
	}
	if outputJson {
		out, err := json.Marshal(solax.NormalizeInfoResponse(*info))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(out))
		os.Exit(0)
	}
	pterm.DefaultSection.Println("Real-time inverter information:")
	pterm.DefaultTable.WithHasHeader().WithData(pterm.TableData{
		{"Parameter", "Value", "Unit"},
		{"Temperature", fmt.Sprintf("%d", info.Temperature), "Celsius"},
		{"EnergyToday", fmt.Sprintf("%.1f", float64(info.EnergyToday)/10), "kWh"},
		{"Vpv1", fmt.Sprintf("%.1f", float64(info.Vpv1)/10), "Volt"},
		{"Vpv2", fmt.Sprintf("%.1f", float64(info.Vpv2)/10), "Volt"},
		{"Apv1", fmt.Sprintf("%.1f", float64(info.Apv1)/10), "Ampere"},
		{"Apv2", fmt.Sprintf("%.1f", float64(info.Apv2)/10), "Ampere"},
		{"Iac", fmt.Sprintf("%.1f", float64(info.Iac)/10), "Ampere"},
		{"Vac", fmt.Sprintf("%.1f", float64(info.Vac)/10), "Volt"},
		{"Frequency", fmt.Sprintf("%.1f", float64(info.Frequency)/100), "Hz"},
		{"Power", fmt.Sprintf("%d", info.Power), "W"},
		{"EnergyTotal", fmt.Sprintf("%.1f", float64(info.EnergyTotal)/10), "KwH"},
		{"TimeTotal", fmt.Sprintf("%d", info.TimeTotal), "Hours"},
		{"Mode", fmt.Sprintf("%X", info.Mode), "-"},
		{"GridVoltFault", fmt.Sprintf("%.1f", float64(info.GridVoltFault)/10), "Volt"},
		{"GridFreqFault", fmt.Sprintf("%.2f", float64(info.GridFreqFault)/100), "Hz"},
		{"DCIFault", fmt.Sprintf("%d", info.DCIFault), "mA"},
		{"TemperatureFault", fmt.Sprintf("%d", info.TemperatureFault), "-"},
		{"PV1Fault", fmt.Sprintf("%.1f", float64(info.PV1Fault)/10), "Volt"},
		{"PV2Fault", fmt.Sprintf("%.1f", float64(info.PV2Fault)/10), "Volt"},
		{"GFCFault", fmt.Sprintf("%d", info.GFCFault), "mA"},
		{"ErrMessage", fmt.Sprintf("%d", info.ErrMessage), "-"},
		{"Last update", time.Now().Format("2006-01-02 15:04:05"), ""},
	}).Render()

}
func DeviceInfo(cmd *cobra.Command, args []string) {
	if address < 0 || address > 255 {
		log.Fatal("Address must be between 1-255")
	}

	client, err := solax.NewClient(device)
	fatalIfError(err)

	inv := &solax.Inverter{Address: byte(address)}
	info, err := client.GetInverterInfo(inv)
	fatalIfError(err)

	if verbose {
		log.Printf("Raw response: %X", client.LastResponse)
	}

	if outputJson {
		out, err := json.Marshal(info)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(out))
		os.Exit(0)
	}

	pterm.DefaultSection.Println("Inverter device information:")
	pterm.DefaultTable.WithHasHeader().WithData(pterm.TableData{
		{"Parameter", "Value", "Unit"},
		{"Phase", fmt.Sprintf("%d", info.Phase), "-"},
		{"RatedPower", info.RatedPower, "W"},
		{"FirmwareVersion", info.FirmwareVersion, "-"},
		{"ModuleName", info.ModuleName, "-"},
		{"FactoryName", info.FactoryName, "-"},
		{"SerialNumber", info.SerialNumber, "-"},
		{"RatedBusVoltage", info.RatedBusVoltage, "V"},
	}).Render()
}

func fatalIfError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
