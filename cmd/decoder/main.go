package main

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"google.golang.org/protobuf/proto"

	generated "fyneMMQT/model/meshtastic"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Использование: go run decoder.go <raw_messages.txt>")
		os.Exit(1)
	}

	filename := os.Args[1]
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Ошибка открытия файла: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Split(line, " | ")
		if len(parts) != 3 {
			fmt.Printf("Строка %d: неверный формат (ожидается: timestamp | topic | hex_data)\n", lineNum)
			continue
		}

		timestamp := parts[0]
		topic := parts[1]
		hexData := parts[2]

		fmt.Printf("\n=== Строка %d ===\n", lineNum)
		fmt.Printf("Время: %s\n", timestamp)
		fmt.Printf("Топик: %s\n", topic)

		// Декодируем hex в байты
		data, err := hex.DecodeString(hexData)
		if err != nil {
			fmt.Printf("Ошибка декодирования hex: %v\n", err)
			continue
		}

		// Определяем тип сообщения по топику
		if strings.Contains(topic, "/map/") {
			if !decodeMapReport(data) {
				// Если не получилось декодировать как MapReport, пробуем ServiceEnvelope
				fmt.Println("Попытка декодировать как ServiceEnvelope...")
				decodeServiceEnvelope(data)
			}
		} else if strings.Contains(topic, "/e/") {
			decodeServiceEnvelope(data)
		} else {
			fmt.Printf("Неизвестный тип топика, пытаемся декодировать как ServiceEnvelope\n")
			decodeServiceEnvelope(data)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Ошибка чтения файла: %v\n", err)
	}
}

func decodeMapReport(data []byte) bool {
	var mapReport generated.MapReport
	if err := proto.Unmarshal(data, &mapReport); err != nil {
		fmt.Printf("Ошибка декодирования MapReport: %v\n", err)
		return false
	}

	fmt.Println("Тип: MapReport")
	fmt.Printf("  Long Name: %s\n", mapReport.GetLongName())
	fmt.Printf("  Short Name: %s\n", mapReport.GetShortName())
	fmt.Printf("  Role: %v\n", mapReport.GetRole())
	fmt.Printf("  Hardware Model: %v\n", mapReport.GetHwModel())
	fmt.Printf("  Firmware Version: %s\n", mapReport.GetFirmwareVersion())
	fmt.Printf("  Region: %v\n", mapReport.GetRegion())
	fmt.Printf("  Modem Preset: %v\n", mapReport.GetModemPreset())
	fmt.Printf("  Has Default Channel: %v\n", mapReport.GetHasDefaultChannel())

	lat := float64(mapReport.GetLatitudeI()) / 1e7
	lon := float64(mapReport.GetLongitudeI()) / 1e7
	if lat != 0 || lon != 0 {
		fmt.Printf("  Position: %.7f, %.7f (alt: %dm)\n", lat, lon, mapReport.GetAltitude())
	}
	fmt.Printf("  Position Precision: %d\n", mapReport.GetPositionPrecision())
	fmt.Printf("  Online Local Nodes: %d\n", mapReport.GetNumOnlineLocalNodes())
	fmt.Printf("  Opted Report Location: %v\n", mapReport.GetHasOptedReportLocation())
	return true
}

func decodeServiceEnvelope(data []byte) {
	var envelope generated.ServiceEnvelope
	if err := proto.Unmarshal(data, &envelope); err != nil {
		fmt.Printf("Ошибка декодирования ServiceEnvelope: %v\n", err)
		return
	}

	fmt.Println("Тип: ServiceEnvelope")
	fmt.Printf("  Channel ID: %s\n", envelope.GetChannelId())
	fmt.Printf("  Gateway ID: %s\n", envelope.GetGatewayId())

	packet := envelope.GetPacket()
	if packet == nil {
		fmt.Println("  Packet: отсутствует")
		return
	}

	fmt.Println("  MeshPacket:")
	fmt.Printf("    From: %d\n", packet.GetFrom())
	fmt.Printf("    To: %d\n", packet.GetTo())
	fmt.Printf("    Channel: %d\n", packet.GetChannel())
	fmt.Printf("    ID: %d\n", packet.GetId())
	fmt.Printf("    Hop Limit: %d\n", packet.GetHopLimit())
	fmt.Printf("    Want Ack: %v\n", packet.GetWantAck())
	fmt.Printf("    Priority: %v\n", packet.GetPriority())
	fmt.Printf("    Via MQTT: %v\n", packet.GetViaMqtt())
	fmt.Printf("    Transport: %v\n", packet.GetTransportMechanism())

	// Проверяем тип payload
	if decoded := packet.GetDecoded(); decoded != nil {
		fmt.Println("    Payload Type: Decoded")
		decodeData(decoded)
	} else if encrypted := packet.GetEncrypted(); encrypted != nil {
		fmt.Printf("    Payload Type: Encrypted (%d bytes)\n", len(encrypted))
		fmt.Printf("    Encrypted Data (hex): %s\n", hex.EncodeToString(encrypted))
	} else {
		fmt.Println("    Payload Type: отсутствует")
	}
}

func decodeData(data *generated.Data) {
	fmt.Printf("    Data:\n")
	fmt.Printf("      Portnum: %v (%d)\n", data.GetPortnum(), data.GetPortnum())
	fmt.Printf("      Want Response: %v\n", data.GetWantResponse())
	fmt.Printf("      Dest: %d\n", data.GetDest())
	fmt.Printf("      Source: %d\n", data.GetSource())
	fmt.Printf("      Request ID: %d\n", data.GetRequestId())
	fmt.Printf("      Reply ID: %d\n", data.GetReplyId())
	fmt.Printf("      Emoji: %d\n", data.GetEmoji())

	payload := data.GetPayload()
	if len(payload) == 0 {
		fmt.Println("      Payload: пустой")
		return
	}

	fmt.Printf("      Payload Size: %d bytes\n", len(payload))

	// Декодируем payload в зависимости от portnum
	switch data.GetPortnum() {
	case generated.PortNum_TEXT_MESSAGE_APP, generated.PortNum_TEXT_MESSAGE_COMPRESSED_APP:
		text := string(payload)
		fmt.Printf("      Text Message: %s\n", text)

	case generated.PortNum_POSITION_APP:
		var position generated.Position
		if err := proto.Unmarshal(payload, &position); err == nil {
			fmt.Println("      Position:")
			lat := float64(position.GetLatitudeI()) / 1e7
			lon := float64(position.GetLongitudeI()) / 1e7
			fmt.Printf("        Location: %.7f, %.7f\n", lat, lon)
			fmt.Printf("        Altitude: %dm\n", position.GetAltitude())
			fmt.Printf("        Time: %d\n", position.GetTime())
			fmt.Printf("        Location Source: %v\n", position.GetLocationSource())
			fmt.Printf("        Precision Bits: %d\n", position.GetPrecisionBits())
			if position.GetGroundTrack() != 0 {
				fmt.Printf("        Ground Track: %d°\n", position.GetGroundTrack())
			}
			if position.GetGroundSpeed() != 0 {
				fmt.Printf("        Ground Speed: %d\n", position.GetGroundSpeed())
			}
		} else {
			fmt.Printf("      Ошибка декодирования Position: %v\n", err)
			fmt.Printf("      Raw Payload (hex): %s\n", hex.EncodeToString(payload))
		}

	case generated.PortNum_NODEINFO_APP:
		var user generated.User
		if err := proto.Unmarshal(payload, &user); err == nil {
			fmt.Println("      User Info:")
			fmt.Printf("        ID: %s\n", user.GetId())
			fmt.Printf("        Long Name: %s\n", user.GetLongName())
			fmt.Printf("        Short Name: %s\n", user.GetShortName())
			if len(user.GetMacaddr()) > 0 {
				fmt.Printf("        Macaddr: %s\n", hex.EncodeToString(user.GetMacaddr()))
			}
			fmt.Printf("        HwModel: %v\n", user.GetHwModel())
			fmt.Printf("        Is Licensed: %v\n", user.GetIsLicensed())
		} else {
			fmt.Printf("      Ошибка декодирования User: %v\n", err)
			fmt.Printf("      Raw Payload (hex): %s\n", hex.EncodeToString(payload))
		}

	case generated.PortNum_TELEMETRY_APP:
		var telemetry generated.Telemetry
		if err := proto.Unmarshal(payload, &telemetry); err == nil {
			fmt.Println("      Telemetry:")
			if deviceMetrics := telemetry.GetDeviceMetrics(); deviceMetrics != nil {
				fmt.Printf("        Device Metrics:\n")
				fmt.Printf("          Battery Level: %d%%\n", deviceMetrics.GetBatteryLevel())
				fmt.Printf("          Voltage: %.2fV\n", deviceMetrics.GetVoltage())
				fmt.Printf("          Channel Utilization: %.2f%%\n", deviceMetrics.GetChannelUtilization())
				fmt.Printf("          Air Utilization TX: %.2f%%\n", deviceMetrics.GetAirUtilTx())
			}
			if envMetrics := telemetry.GetEnvironmentMetrics(); envMetrics != nil {
				fmt.Printf("        Environment Metrics:\n")
				fmt.Printf("          Temperature: %.1f°C\n", envMetrics.GetTemperature())
				fmt.Printf("          Relative Humidity: %.1f%%\n", envMetrics.GetRelativeHumidity())
				fmt.Printf("          Barometric Pressure: %.1f hPa\n", envMetrics.GetBarometricPressure())
				fmt.Printf("          Gas Resistance: %.1f\n", envMetrics.GetGasResistance())
			}
		} else {
			fmt.Printf("      Ошибка декодирования Telemetry: %v\n", err)
			fmt.Printf("      Raw Payload (hex): %s\n", hex.EncodeToString(payload))
		}

	case generated.PortNum_WAYPOINT_APP:
		var waypoint generated.Waypoint
		if err := proto.Unmarshal(payload, &waypoint); err == nil {
			fmt.Println("      Waypoint:")
			fmt.Printf("        ID: %d\n", waypoint.GetId())
			lat := float64(waypoint.GetLatitudeI()) / 1e7
			lon := float64(waypoint.GetLongitudeI()) / 1e7
			fmt.Printf("        Location: %.7f, %.7f\n", lat, lon)
			fmt.Printf("        Name: %s\n", waypoint.GetName())
			fmt.Printf("        Description: %s\n", waypoint.GetDescription())
		} else {
			fmt.Printf("      Ошибка декодирования Waypoint: %v\n", err)
			fmt.Printf("      Raw Payload (hex): %s\n", hex.EncodeToString(payload))
		}

	case generated.PortNum_ROUTING_APP:
		var routing generated.Routing
		if err := proto.Unmarshal(payload, &routing); err == nil {
			fmt.Println("      Routing:")
			fmt.Printf("        Variant: %v\n", routing.GetVariant())
			if errRoute := routing.GetErrorReason(); errRoute != generated.Routing_NONE {
				fmt.Printf("        Error Reason: %v\n", errRoute)
			}
		} else {
			fmt.Printf("      Ошибка декодирования Routing: %v\n", err)
			fmt.Printf("      Raw Payload (hex): %s\n", hex.EncodeToString(payload))
		}

	case generated.PortNum_ADMIN_APP:
		var admin generated.AdminMessage
		if err := proto.Unmarshal(payload, &admin); err == nil {
			fmt.Println("      Admin Message:")
			// AdminMessage имеет много вариантов, выводим общую информацию
			jsonData, _ := json.MarshalIndent(admin, "        ", "  ")
			fmt.Printf("        %s\n", string(jsonData))
		} else {
			fmt.Printf("      Ошибка декодирования AdminMessage: %v\n", err)
			fmt.Printf("      Raw Payload (hex): %s\n", hex.EncodeToString(payload))
		}

	case generated.PortNum_REMOTE_HARDWARE_APP:
		var hw generated.HardwareMessage
		if err := proto.Unmarshal(payload, &hw); err == nil {
			fmt.Println("      Remote Hardware:")
			fmt.Printf("        Type: %v\n", hw.GetType())
			fmt.Printf("        GPIO Mask: %d\n", hw.GetGpioMask())
			fmt.Printf("        GPIO Value: %d\n", hw.GetGpioValue())
		} else {
			fmt.Printf("      Ошибка декодирования HardwareMessage: %v\n", err)
			fmt.Printf("      Raw Payload (hex): %s\n", hex.EncodeToString(payload))
		}

	case generated.PortNum_MAP_REPORT_APP:
		var mapReport generated.MapReport
		if err := proto.Unmarshal(payload, &mapReport); err == nil {
			fmt.Println("      Map Report:")
			fmt.Printf("        Long Name: %s\n", mapReport.GetLongName())
			fmt.Printf("        Short Name: %s\n", mapReport.GetShortName())
			fmt.Printf("        Role: %v\n", mapReport.GetRole())
			fmt.Printf("        Hardware Model: %v\n", mapReport.GetHwModel())
			fmt.Printf("        Firmware Version: %s\n", mapReport.GetFirmwareVersion())
			fmt.Printf("        Region: %v\n", mapReport.GetRegion())
			fmt.Printf("        Modem Preset: %v\n", mapReport.GetModemPreset())
			fmt.Printf("        Has Default Channel: %v\n", mapReport.GetHasDefaultChannel())
			lat := float64(mapReport.GetLatitudeI()) / 1e7
			lon := float64(mapReport.GetLongitudeI()) / 1e7
			if lat != 0 || lon != 0 {
				fmt.Printf("        Position: %.7f, %.7f (alt: %dm)\n", lat, lon, mapReport.GetAltitude())
			}
			fmt.Printf("        Position Precision: %d\n", mapReport.GetPositionPrecision())
			fmt.Printf("        Online Local Nodes: %d\n", mapReport.GetNumOnlineLocalNodes())
			fmt.Printf("        Opted Report Location: %v\n", mapReport.GetHasOptedReportLocation())
		} else {
			fmt.Printf("      Ошибка декодирования MapReport: %v\n", err)
			fmt.Printf("      Raw Payload (hex): %s\n", hex.EncodeToString(payload))
		}

	default:
		// Для неизвестных типов выводим hex
		fmt.Printf("      Unknown Portnum, Raw Payload (hex): %s\n", hex.EncodeToString(payload))
		fmt.Printf("      Payload (first 100 bytes as text): %s\n",
			strings.ReplaceAll(string(payload[:min(100, len(payload))]), "\n", "\\n"))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
