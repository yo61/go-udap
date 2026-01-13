package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"squeezebox-udap/udap"

	"github.com/chzyer/readline"
)

// CLI functions
func printUsage() {
	fmt.Println("Squeezebox UDAP Configuration Tool")
	fmt.Println("Usage:")
	fmt.Println("  discover                     - Discover devices on network")
	fmt.Println("  list                        - List discovered devices")
	fmt.Println("  read <mac>                  - Read all parameters from device")
	fmt.Println("  config <mac> get <param>    - Get configuration parameter")
	fmt.Println("  config <mac> set <param>=<value> [param2=value2 ...] - Set configuration parameters")
	fmt.Println("  save <mac>                  - Save configuration to device persistent storage")
	fmt.Println("  commit <mac>                - Save configuration and reset device")
	fmt.Println("  reset <mac>                 - Reset device (required after config changes)")
	fmt.Println("  info <mac>                  - Show device information")
	fmt.Println("  help                        - Show this help")
	fmt.Println("  quit                        - Exit the tool")
}

// parseParameters parses parameter strings respecting quotes
func parseParameters(input string) []string {
	var result []string
	var current strings.Builder
	inQuotes := false
	escapeNext := false

	for i, ch := range input {
		if escapeNext {
			current.WriteRune(ch)
			escapeNext = false
			continue
		}

		switch ch {
		case '\\':
			escapeNext = true
		case '"':
			inQuotes = !inQuotes
		case ' ':
			if !inQuotes {
				if current.Len() > 0 {
					result = append(result, current.String())
					current.Reset()
				}
			} else {
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}

		// Handle end of string
		if i == len(input)-1 && current.Len() > 0 {
			result = append(result, current.String())
		}
	}

	return result
}

// createCompleter creates an autocomplete configuration for readline
func createCompleter() *readline.PrefixCompleter {
	return readline.NewPrefixCompleter(
		readline.PcItem("help"),
		readline.PcItem("discover"),
		readline.PcItem("list"),
		readline.PcItem("read"),
		readline.PcItem("info"),
		readline.PcItem("config",
			readline.PcItem("<mac>",
				readline.PcItem("get"),
				readline.PcItem("set"),
			),
		),
		readline.PcItem("save"),
		readline.PcItem("commit"),
		readline.PcItem("reset"),
		readline.PcItem("quit"),
		readline.PcItem("exit"),
	)
}

func main() {
	client, err := udap.NewClient()
	if err != nil {
		log.Fatal("Failed to create UDAP client:", err)
	}
	defer client.Close()

	fmt.Println("Squeezebox UDAP Configuration Tool")
	fmt.Println("Type 'help' for available commands")

	// Get home directory for history file
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	historyFile := filepath.Join(homeDir, ".squeezebox_udap_history")

	// Configure readline with proper terminal handling
	rl, err := readline.NewEx(&readline.Config{
		Prompt:                 "> ",
		HistoryFile:            historyFile,
		AutoComplete:           createCompleter(),
		InterruptPrompt:        "^C",
		EOFPrompt:              "exit",
		HistorySearchFold:      true,
		VimMode:                false,
		UniqueEditLine:         true,
		DisableAutoSaveHistory: false,
		FuncIsTerminal: func() bool {
			return readline.IsTerminal(int(os.Stdin.Fd()))
		},
	})
	if err != nil {
		log.Fatal("Failed to create readline:", err)
	}
	defer rl.Close()

	for {
		line, err := rl.Readline()
		if err != nil { // io.EOF or user interrupt
			break
		}

		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}

		// Special handling for set command to preserve quotes
		var args []string
		if strings.HasPrefix(input, "config") && strings.Contains(input, "set") {
			// Parse manually to preserve quoted values
			parts := strings.Fields(input)
			if len(parts) >= 3 && parts[2] == "set" {
				// Keep first 3 parts (config, mac, set)
				args = parts[:3]
				// Find the rest after "set"
				_, after, ok := strings.Cut(input, "set")
				if ok {
					remainder := strings.TrimSpace(after)
					if remainder != "" {
						// Split by spaces but respect quotes
						paramArgs := parseParameters(remainder)
						args = append(args, paramArgs...)
					}
				}
			} else {
				args = strings.Fields(input)
			}
		} else {
			args = strings.Fields(input)
		}

		if len(args) == 0 {
			continue
		}

		command := args[0]

		switch command {
		case "help":
			printUsage()

		case "discover":
			err := client.DiscoverDevices(5 * time.Second)
			if err != nil {
				fmt.Printf("Discovery failed: %v\n", err)
			} else {
				devices := client.GetDevices()
				fmt.Printf("Discovery complete. Found %d devices.\n", len(devices))
			}

		case "list":
			devices := client.ListDevices()
			if len(devices) == 0 {
				fmt.Println("No devices found. Run 'discover' first.")
			} else {
				fmt.Println("Discovered devices:")
				for _, device := range devices {
					fmt.Printf("  %s - %s (%s) at %s\n",
						device.MAC, device.Name, device.Model, device.IP)
				}
			}

		case "read":
			if len(args) < 2 {
				fmt.Println("Usage: read <mac>")
				continue
			}

			device := client.GetDevice(args[1])
			if device == nil {
				fmt.Println("Device not found")
				continue
			}

			err := client.GetAllDeviceConfig(device)
			if err != nil {
				fmt.Printf("Failed to read device parameters: %v\n", err)
			} else {
				fmt.Printf("Device Parameters (%d total):\n", len(device.Parameters))
				for param, value := range device.Parameters {
					fmt.Printf("  %s = %s\n", param, value)
				}
			}

		case "info":
			if len(args) < 2 {
				fmt.Println("Usage: info <mac>")
				continue
			}

			device := client.GetDevice(args[1])
			if device == nil {
				fmt.Println("Device not found")
				continue
			}

			fmt.Printf("Device Information:\n")
			fmt.Printf("  MAC: %s\n", device.MAC)
			fmt.Printf("  Name: %s\n", device.Name)
			fmt.Printf("  Model: %s\n", device.Model)
			fmt.Printf("  Firmware: %s\n", device.Firmware)
			fmt.Printf("  IP: %s\n", device.IP)
			fmt.Printf("  UUID: %s\n", device.UUID)
			fmt.Printf("  Last Seen: %s\n", device.LastSeen.Format(time.RFC3339))
			if len(device.Parameters) > 0 {
				fmt.Printf("  Parameters loaded: %d\n", len(device.Parameters))
			} else {
				fmt.Printf("  Parameters: Not loaded (use 'read <mac>' to load)\n")
			}

		case "config":
			if len(args) < 3 {
				fmt.Println("Usage: config <mac> get <param> | config <mac> set <param> <value>")
				continue
			}

			device := client.GetDevice(args[1])
			if device == nil {
				fmt.Println("Device not found")
				continue
			}

			subcommand := args[2]
			switch subcommand {
			case "get":
				if len(args) < 4 {
					fmt.Println("Usage: config <mac> get <param>")
					continue
				}

				params := []string{args[3]}
				config, err := client.GetDeviceConfig(device, params)
				if err != nil {
					fmt.Printf("Failed to get config: %v\n", err)
				} else {
					for param, value := range config {
						fmt.Printf("%s = %s\n", param, value)
					}
				}

			case "set":
				if len(args) < 4 {
					fmt.Println("Usage: config <mac> set <param>=<value> [param2=value2 ...]")
					continue
				}

				// Parse all param=value pairs
				config := make(map[string]string)
				for _, arg := range args[3:] {
					parts := strings.SplitN(arg, "=", 2)
					if len(parts) != 2 {
						fmt.Printf("Invalid parameter format: %s (expected param=value)\n", arg)
						continue
					}

					param := parts[0]
					value := parts[1]

					// Remove quotes if present
					if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
						value = strings.Trim(value, "\"")
					}

					config[param] = value
				}

				if len(config) == 0 {
					fmt.Println("No valid parameters provided")
					continue
				}

				fmt.Printf("Setting %d parameters on device %s:\n", len(config), device.MAC)
				for param, value := range config {
					fmt.Printf("  %s = %s\n", param, value)
				}

				err := client.SetDeviceConfig(device, config)
				if err != nil {
					fmt.Printf("Failed to set config: %v\n", err)
				} else {
					fmt.Println("Configuration updated successfully")
					fmt.Printf("Note: Configuration changes are not persistent until you run 'save %s'\n", device.MAC)
				}

			default:
				fmt.Println("Unknown config subcommand. Use 'get' or 'set'")
			}

		case "save":
			if len(args) < 2 {
				fmt.Println("Usage: save <mac>")
				continue
			}

			device := client.GetDevice(args[1])
			if device == nil {
				fmt.Println("Device not found")
				continue
			}

			err := client.SaveDeviceConfig(device)
			if err != nil {
				fmt.Printf("Failed to save device configuration: %v\n", err)
			} else {
				fmt.Println("Device configuration saved to persistent storage")
				fmt.Printf("Note: Run 'reset %s' to apply saved configuration\n", device.MAC)
			}

		case "commit":
			if len(args) < 2 {
				fmt.Println("Usage: commit <mac>")
				continue
			}

			device := client.GetDevice(args[1])
			if device == nil {
				fmt.Println("Device not found")
				continue
			}

			fmt.Println("Committing configuration (save + reset)...")

			// First save the configuration
			err := client.SaveDeviceConfig(device)
			if err != nil {
				fmt.Printf("Failed to save device configuration: %v\n", err)
				continue
			}

			fmt.Println("Configuration saved, now resetting device...")

			// Then reset the device
			err = client.ResetDevice(device)
			if err != nil {
				fmt.Printf("Failed to reset device: %v\n", err)
			} else {
				fmt.Println("Device configuration committed and device reset")
				fmt.Println("Device should reboot with new configuration")
			}

		case "reset":
			if len(args) < 2 {
				fmt.Println("Usage: reset <mac>")
				continue
			}

			device := client.GetDevice(args[1])
			if device == nil {
				fmt.Println("Device not found")
				continue
			}

			err := client.ResetDevice(device)
			if err != nil {
				fmt.Printf("Failed to reset device: %v\n", err)
			}

		case "quit", "exit":
			fmt.Println("Goodbye!")
			return

		default:
			fmt.Printf("Unknown command: %s\n", command)
			fmt.Println("Type 'help' for available commands")
		}
	}

	fmt.Println("\nGoodbye!")
}
