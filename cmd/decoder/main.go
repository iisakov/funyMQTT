package main

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"google.golang.org/protobuf/proto"

	generated "fyneMMQT/model/meshtastic"
)

// CSVRecord представляет одну строку CSV файла
type CSVRecord struct {
	Timestamp     string
	Topic         string
	MessageType   string
	ChannelID     string
	GatewayID     string
	From          string
	To            string
	PacketID      string
	Channel       string
	HopLimit      string
	WantAck       string
	Priority      string
	ViaMQTT       string
	Transport     string
	PayloadType   string
	Portnum       string
	PortnumName   string
	PayloadSize   string
	EncryptedData string

	// Position fields
	Latitude       string
	Longitude      string
	Altitude       string
	PositionTime   string
	LocationSource string
	PrecisionBits  string
	GroundTrack    string
	GroundSpeed    string

	// Text message
	TextMessage string

	// User info
	UserID         string
	UserLongName   string
	UserShortName  string
	UserMacaddr    string
	UserHwModel    string
	UserIsLicensed string

	// Telemetry
	BatteryLevel       string
	Voltage            string
	ChannelUtilization string
	AirUtilTx          string
	Temperature        string
	RelativeHumidity   string
	BarometricPressure string
	GasResistance      string

	// Map Report
	MapLongName            string
	MapShortName           string
	MapRole                string
	MapHwModel             string
	MapFirmwareVersion     string
	MapRegion              string
	MapModemPreset         string
	MapHasDefaultChannel   string
	MapPositionPrecision   string
	MapOnlineLocalNodes    string
	MapOptedReportLocation string

	// Waypoint
	WaypointID          string
	WaypointName        string
	WaypointDescription string

	// Routing
	RoutingVariant     string
	RoutingErrorReason string

	// Remote Hardware
	HwType      string
	HwGpioMask  string
	HwGpioValue string

	// Error
	Error string
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Использование: go run cmd/decoder/main.go <raw_messages.txt> [output.csv]")
		fmt.Println("Или: ./decoder <raw_messages.txt> [output.csv]")
		fmt.Println("По умолчанию выходной файл: decoded_messages.csv")
		os.Exit(1)
	}

	inputFile := os.Args[1]
	outputFile := "decoded_messages.csv"
	if len(os.Args) >= 3 {
		outputFile = os.Args[2]
	}

	// Открываем входной файл
	file, err := os.Open(inputFile)
	if err != nil {
		fmt.Printf("Ошибка открытия файла: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// Создаем CSV файл
	csvFile, err := os.Create(outputFile)
	if err != nil {
		fmt.Printf("Ошибка создания CSV файла: %v\n", err)
		os.Exit(1)
	}
	defer csvFile.Close()

	writer := csv.NewWriter(csvFile)
	defer writer.Flush()

	// Записываем заголовки
	headers := []string{
		"Timestamp", "Topic", "MessageType", "ChannelID", "GatewayID",
		"From", "To", "PacketID", "Channel", "HopLimit", "WantAck", "Priority",
		"ViaMQTT", "Transport", "PayloadType", "Portnum", "PortnumName", "PayloadSize",
		"EncryptedData", "Latitude", "Longitude", "Altitude", "PositionTime",
		"LocationSource", "PrecisionBits", "GroundTrack", "GroundSpeed",
		"TextMessage", "UserID", "UserLongName", "UserShortName", "UserMacaddr",
		"UserHwModel", "UserIsLicensed", "BatteryLevel", "Voltage", "ChannelUtilization",
		"AirUtilTx", "Temperature", "RelativeHumidity", "BarometricPressure", "GasResistance",
		"MapLongName", "MapShortName", "MapRole", "MapHwModel", "MapFirmwareVersion",
		"MapRegion", "MapModemPreset", "MapHasDefaultChannel", "MapPositionPrecision",
		"MapOnlineLocalNodes", "MapOptedReportLocation", "WaypointID", "WaypointName",
		"WaypointDescription", "RoutingVariant", "RoutingErrorReason", "HwType",
		"HwGpioMask", "HwGpioValue", "Error",
	}
	if err := writer.Write(headers); err != nil {
		fmt.Printf("Ошибка записи заголовков: %v\n", err)
		os.Exit(1)
	}

	scanner := bufio.NewScanner(file)
	lineNum := 0
	processed := 0

	fmt.Printf("Обработка файла %s...\n", inputFile)

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Split(line, " | ")
		if len(parts) != 3 {
			record := CSVRecord{
				Timestamp: parts[0],
				Topic:     parts[1],
				Error:     fmt.Sprintf("Неверный формат строки %d", lineNum),
			}
			writeRecord(writer, record)
			continue
		}

		timestamp := parts[0]
		topic := parts[1]
		hexData := parts[2]

		// Декодируем hex в байты
		data, err := hex.DecodeString(hexData)
		if err != nil {
			record := CSVRecord{
				Timestamp: timestamp,
				Topic:     topic,
				Error:     fmt.Sprintf("Ошибка декодирования hex: %v", err),
			}
			writeRecord(writer, record)
			continue
		}

		var record CSVRecord
		record.Timestamp = timestamp
		record.Topic = topic

		// Определяем тип сообщения по топику
		if strings.Contains(topic, "/map/") {
			if !decodeMapReport(data, &record) {
				// Если не получилось декодировать как MapReport, пробуем ServiceEnvelope
				decodeServiceEnvelope(data, &record)
			}
		} else if strings.Contains(topic, "/e/") {
			decodeServiceEnvelope(data, &record)
		} else {
			decodeServiceEnvelope(data, &record)
		}

		writeRecord(writer, record)
		processed++

		if processed%100 == 0 {
			fmt.Printf("Обработано %d сообщений...\n", processed)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Ошибка чтения файла: %v\n", err)
	}

	fmt.Printf("Готово! Обработано %d сообщений. Результаты сохранены в %s\n", processed, outputFile)
}

func decodeMapReport(data []byte, record *CSVRecord) bool {
	var mapReport generated.MapReport
	if err := proto.Unmarshal(data, &mapReport); err != nil {
		record.Error = fmt.Sprintf("Ошибка декодирования MapReport: %v", err)
		return false
	}

	record.MessageType = "MapReport"
	record.MapLongName = mapReport.GetLongName()
	record.MapShortName = mapReport.GetShortName()
	record.MapRole = mapReport.GetRole().String()
	record.MapHwModel = mapReport.GetHwModel().String()
	record.MapFirmwareVersion = mapReport.GetFirmwareVersion()
	record.MapRegion = mapReport.GetRegion().String()
	record.MapModemPreset = mapReport.GetModemPreset().String()
	record.MapHasDefaultChannel = boolToString(mapReport.GetHasDefaultChannel())

	lat := float64(mapReport.GetLatitudeI()) / 1e7
	lon := float64(mapReport.GetLongitudeI()) / 1e7
	if lat != 0 || lon != 0 {
		record.Latitude = fmt.Sprintf("%.7f", lat)
		record.Longitude = fmt.Sprintf("%.7f", lon)
		record.Altitude = fmt.Sprintf("%d", mapReport.GetAltitude())
	}

	record.MapPositionPrecision = fmt.Sprintf("%d", mapReport.GetPositionPrecision())
	record.MapOnlineLocalNodes = fmt.Sprintf("%d", mapReport.GetNumOnlineLocalNodes())
	record.MapOptedReportLocation = boolToString(mapReport.GetHasOptedReportLocation())
	return true
}

func decodeServiceEnvelope(data []byte, record *CSVRecord) {
	var envelope generated.ServiceEnvelope
	if err := proto.Unmarshal(data, &envelope); err != nil {
		record.Error = fmt.Sprintf("Ошибка декодирования ServiceEnvelope: %v", err)
		return
	}

	record.MessageType = "ServiceEnvelope"
	record.ChannelID = envelope.GetChannelId()
	record.GatewayID = envelope.GetGatewayId()

	packet := envelope.GetPacket()
	if packet == nil {
		record.Error = "Packet отсутствует"
		return
	}

	record.From = fmt.Sprintf("%d", packet.GetFrom())
	record.To = fmt.Sprintf("%d", packet.GetTo())
	record.Channel = fmt.Sprintf("%d", packet.GetChannel())
	record.PacketID = fmt.Sprintf("%d", packet.GetId())
	record.HopLimit = fmt.Sprintf("%d", packet.GetHopLimit())
	record.WantAck = boolToString(packet.GetWantAck())
	record.Priority = packet.GetPriority().String()
	record.ViaMQTT = boolToString(packet.GetViaMqtt())
	record.Transport = packet.GetTransportMechanism().String()

	// Проверяем тип payload
	if decoded := packet.GetDecoded(); decoded != nil {
		record.PayloadType = "Decoded"
		decodeData(decoded, record)
	} else if encrypted := packet.GetEncrypted(); encrypted != nil {
		record.PayloadType = "Encrypted"
		record.EncryptedData = hex.EncodeToString(encrypted)
		record.PayloadSize = fmt.Sprintf("%d", len(encrypted))

		// Пытаемся расшифровать (если функция расшифровки реализована)
		// Для полной поддержки расшифровки установите библиотеку github.com/meshtastic/go
		if decryptedData := tryDecrypt(packet, encrypted); decryptedData != nil {
			var data generated.Data
			if err := proto.Unmarshal(decryptedData, &data); err == nil {
				// Успешно расшифровано!
				record.PayloadType = "Decrypted"
				decodeData(&data, record)
			}
		}
	} else {
		record.PayloadType = "Отсутствует"
	}
}

func decodeData(data *generated.Data, record *CSVRecord) {
	record.Portnum = fmt.Sprintf("%d", data.GetPortnum())
	record.PortnumName = data.GetPortnum().String()
	record.PayloadSize = fmt.Sprintf("%d", len(data.GetPayload()))

	payload := data.GetPayload()
	if len(payload) == 0 {
		return
	}

	// Декодируем payload в зависимости от portnum
	switch data.GetPortnum() {
	case generated.PortNum_TEXT_MESSAGE_APP, generated.PortNum_TEXT_MESSAGE_COMPRESSED_APP:
		record.TextMessage = string(payload)

	case generated.PortNum_POSITION_APP:
		var position generated.Position
		if err := proto.Unmarshal(payload, &position); err == nil {
			lat := float64(position.GetLatitudeI()) / 1e7
			lon := float64(position.GetLongitudeI()) / 1e7
			if lat != 0 || lon != 0 {
				record.Latitude = fmt.Sprintf("%.7f", lat)
				record.Longitude = fmt.Sprintf("%.7f", lon)
			}
			if position.GetAltitude() != 0 {
				record.Altitude = fmt.Sprintf("%d", position.GetAltitude())
			}
			record.PositionTime = fmt.Sprintf("%d", position.GetTime())
			record.LocationSource = position.GetLocationSource().String()
			record.PrecisionBits = fmt.Sprintf("%d", position.GetPrecisionBits())
			if position.GetGroundTrack() != 0 {
				record.GroundTrack = fmt.Sprintf("%d", position.GetGroundTrack())
			}
			if position.GetGroundSpeed() != 0 {
				record.GroundSpeed = fmt.Sprintf("%d", position.GetGroundSpeed())
			}
		} else {
			record.Error = fmt.Sprintf("Ошибка декодирования Position: %v", err)
		}

	case generated.PortNum_NODEINFO_APP:
		var user generated.User
		if err := proto.Unmarshal(payload, &user); err == nil {
			record.UserID = user.GetId()
			record.UserLongName = user.GetLongName()
			record.UserShortName = user.GetShortName()
			if len(user.GetMacaddr()) > 0 {
				record.UserMacaddr = hex.EncodeToString(user.GetMacaddr())
			}
			record.UserHwModel = user.GetHwModel().String()
			record.UserIsLicensed = boolToString(user.GetIsLicensed())
		} else {
			record.Error = fmt.Sprintf("Ошибка декодирования User: %v", err)
		}

	case generated.PortNum_TELEMETRY_APP:
		var telemetry generated.Telemetry
		if err := proto.Unmarshal(payload, &telemetry); err == nil {
			if deviceMetrics := telemetry.GetDeviceMetrics(); deviceMetrics != nil {
				record.BatteryLevel = fmt.Sprintf("%d", deviceMetrics.GetBatteryLevel())
				record.Voltage = fmt.Sprintf("%.2f", deviceMetrics.GetVoltage())
				record.ChannelUtilization = fmt.Sprintf("%.2f", deviceMetrics.GetChannelUtilization())
				record.AirUtilTx = fmt.Sprintf("%.2f", deviceMetrics.GetAirUtilTx())
			}
			if envMetrics := telemetry.GetEnvironmentMetrics(); envMetrics != nil {
				record.Temperature = fmt.Sprintf("%.1f", envMetrics.GetTemperature())
				record.RelativeHumidity = fmt.Sprintf("%.1f", envMetrics.GetRelativeHumidity())
				record.BarometricPressure = fmt.Sprintf("%.1f", envMetrics.GetBarometricPressure())
				record.GasResistance = fmt.Sprintf("%.1f", envMetrics.GetGasResistance())
			}
		} else {
			record.Error = fmt.Sprintf("Ошибка декодирования Telemetry: %v", err)
		}

	case generated.PortNum_WAYPOINT_APP:
		var waypoint generated.Waypoint
		if err := proto.Unmarshal(payload, &waypoint); err == nil {
			record.WaypointID = fmt.Sprintf("%d", waypoint.GetId())
			record.WaypointName = waypoint.GetName()
			record.WaypointDescription = waypoint.GetDescription()
			lat := float64(waypoint.GetLatitudeI()) / 1e7
			lon := float64(waypoint.GetLongitudeI()) / 1e7
			if lat != 0 || lon != 0 {
				record.Latitude = fmt.Sprintf("%.7f", lat)
				record.Longitude = fmt.Sprintf("%.7f", lon)
			}
		} else {
			record.Error = fmt.Sprintf("Ошибка декодирования Waypoint: %v", err)
		}

	case generated.PortNum_ROUTING_APP:
		var routing generated.Routing
		if err := proto.Unmarshal(payload, &routing); err == nil {
			record.RoutingVariant = fmt.Sprintf("%v", routing.GetVariant())
			if errRoute := routing.GetErrorReason(); errRoute != generated.Routing_NONE {
				record.RoutingErrorReason = errRoute.String()
			}
		} else {
			record.Error = fmt.Sprintf("Ошибка декодирования Routing: %v", err)
		}

	case generated.PortNum_REMOTE_HARDWARE_APP:
		var hw generated.HardwareMessage
		if err := proto.Unmarshal(payload, &hw); err == nil {
			record.HwType = hw.GetType().String()
			record.HwGpioMask = fmt.Sprintf("%d", hw.GetGpioMask())
			record.HwGpioValue = fmt.Sprintf("%d", hw.GetGpioValue())
		} else {
			record.Error = fmt.Sprintf("Ошибка декодирования HardwareMessage: %v", err)
		}

	case generated.PortNum_MAP_REPORT_APP:
		var mapReport generated.MapReport
		if err := proto.Unmarshal(payload, &mapReport); err == nil {
			record.MapLongName = mapReport.GetLongName()
			record.MapShortName = mapReport.GetShortName()
			record.MapRole = mapReport.GetRole().String()
			record.MapHwModel = mapReport.GetHwModel().String()
			record.MapFirmwareVersion = mapReport.GetFirmwareVersion()
			record.MapRegion = mapReport.GetRegion().String()
			record.MapModemPreset = mapReport.GetModemPreset().String()
			record.MapHasDefaultChannel = boolToString(mapReport.GetHasDefaultChannel())
			lat := float64(mapReport.GetLatitudeI()) / 1e7
			lon := float64(mapReport.GetLongitudeI()) / 1e7
			if lat != 0 || lon != 0 {
				record.Latitude = fmt.Sprintf("%.7f", lat)
				record.Longitude = fmt.Sprintf("%.7f", lon)
				record.Altitude = fmt.Sprintf("%d", mapReport.GetAltitude())
			}
			record.MapPositionPrecision = fmt.Sprintf("%d", mapReport.GetPositionPrecision())
			record.MapOnlineLocalNodes = fmt.Sprintf("%d", mapReport.GetNumOnlineLocalNodes())
			record.MapOptedReportLocation = boolToString(mapReport.GetHasOptedReportLocation())
		} else {
			record.Error = fmt.Sprintf("Ошибка декодирования MapReport: %v", err)
		}
	}
}

func writeRecord(writer *csv.Writer, record CSVRecord) {
	row := []string{
		record.Timestamp, record.Topic, record.MessageType, record.ChannelID, record.GatewayID,
		record.From, record.To, record.PacketID, record.Channel, record.HopLimit, record.WantAck,
		record.Priority, record.ViaMQTT, record.Transport, record.PayloadType, record.Portnum,
		record.PortnumName, record.PayloadSize, record.EncryptedData, record.Latitude, record.Longitude,
		record.Altitude, record.PositionTime, record.LocationSource, record.PrecisionBits,
		record.GroundTrack, record.GroundSpeed, record.TextMessage, record.UserID, record.UserLongName,
		record.UserShortName, record.UserMacaddr, record.UserHwModel, record.UserIsLicensed,
		record.BatteryLevel, record.Voltage, record.ChannelUtilization, record.AirUtilTx,
		record.Temperature, record.RelativeHumidity, record.BarometricPressure, record.GasResistance,
		record.MapLongName, record.MapShortName, record.MapRole, record.MapHwModel,
		record.MapFirmwareVersion, record.MapRegion, record.MapModemPreset, record.MapHasDefaultChannel,
		record.MapPositionPrecision, record.MapOnlineLocalNodes, record.MapOptedReportLocation,
		record.WaypointID, record.WaypointName, record.WaypointDescription, record.RoutingVariant,
		record.RoutingErrorReason, record.HwType, record.HwGpioMask, record.HwGpioValue, record.Error,
	}
	if err := writer.Write(row); err != nil {
		fmt.Printf("Ошибка записи в CSV: %v\n", err)
	}
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// tryDecrypt пытается расшифровать зашифрованные данные используя стандартные ключи Meshtastic
//
// Реализация основана на алгоритме Meshtastic:
// - AES-256-CTR для PSK шифрования каналов
// - Nonce формируется из ID пакета (8 байт) и From узла (4 байта) + padding
// - Поддерживаются стандартные ключи: default и simple1-10
//
// ВАЖНО: PKI шифрование (для прямых сообщений) не поддерживается в этой реализации,
// так как требует приватных ключей устройств.
// tryDecrypt пытается расшифровать зашифрованные данные используя стандартные ключи Meshtastic
// Реализация основана на алгоритме Meshtastic: AES-256-CTR с nonce из ID пакета и From узла
func tryDecrypt(packet *generated.MeshPacket, encrypted []byte) []byte {
	if len(encrypted) == 0 {
		return nil
	}

	// Если используется PKI шифрование, нужен приватный ключ (не реализовано здесь)
	if packet.GetPkiEncrypted() {
		// Для PKI нужен приватный ключ получателя - это требует библиотеки Meshtastic Go
		return nil
	}

	// Стандартный ключ по умолчанию для каналов Meshtastic
	// Из channel.proto: {0xd4, 0xf1, 0xbb, 0x3a, 0x20, 0x29, 0x07, 0x59, 0xf0, 0xbc, 0xff, 0xab, 0xcf, 0x4e, 0x69, 0x01}
	defaultPSK := []byte{0xd4, 0xf1, 0xbb, 0x3a, 0x20, 0x29, 0x07, 0x59, 0xf0, 0xbc, 0xff, 0xab, 0xcf, 0x4e, 0x69, 0x01}

	// Пробуем стандартный ключ и варианты (simple1-10)
	psks := [][]byte{defaultPSK}
	for i := 1; i <= 10; i++ {
		psk := make([]byte, 16)
		copy(psk, defaultPSK)
		psk[15] = defaultPSK[15] + byte(i)
		psks = append(psks, psk)
	}

	// Формируем nonce из ID пакета и From узла
	// Meshtastic использует ID пакета и From узла для формирования nonce
	packetID := packet.GetId()
	fromNode := packet.GetFrom()

	// Пробуем расшифровать с каждым ключом
	for _, psk := range psks {
		// Meshtastic использует AES-256, поэтому расширяем 16-байтный PSK до 32 байт
		// Путем дублирования (стандартный подход Meshtastic)
		key := make([]byte, 32)
		copy(key[:16], psk)
		copy(key[16:], psk)

		// Формируем nonce: для AES-CTR нужен 16-байтный nonce
		// Meshtastic использует ID пакета (8 байт) и From узла (4 байта) + padding
		nonce := make([]byte, 16)
		// Первые 8 байт - ID пакета (little-endian)
		nonce[0] = byte(packetID)
		nonce[1] = byte(packetID >> 8)
		nonce[2] = byte(packetID >> 16)
		nonce[3] = byte(packetID >> 24)
		nonce[4] = 0
		nonce[5] = 0
		nonce[6] = 0
		nonce[7] = 0
		// Следующие 4 байта - From узла (little-endian)
		nonce[8] = byte(fromNode)
		nonce[9] = byte(fromNode >> 8)
		nonce[10] = byte(fromNode >> 16)
		nonce[11] = byte(fromNode >> 24)
		nonce[12] = 0
		nonce[13] = 0
		nonce[14] = 0
		nonce[15] = 0

		// Пробуем AES-256-CTR
		if decrypted := decryptAESCTR(encrypted, key, nonce); len(decrypted) > 0 {
			// Проверяем, что расшифрованные данные валидны (попытка распарсить как Data)
			var testData generated.Data
			if err := proto.Unmarshal(decrypted, &testData); err == nil {
				// Успешно расшифровано!
				return decrypted
			}
		}

		// Также пробуем с оригинальным 16-байтным ключом для AES-128
		if decrypted := decryptAESCTR(encrypted, psk, nonce); len(decrypted) > 0 {
			var testData generated.Data
			if err := proto.Unmarshal(decrypted, &testData); err == nil {
				return decrypted
			}
		}
	}

	return nil
}

// decryptAESCTR расшифровывает данные используя AES-CTR
func decryptAESCTR(ciphertext []byte, key []byte, nonce []byte) []byte {
	if len(ciphertext) == 0 || len(nonce) != 16 {
		return nil
	}

	var block cipher.Block
	var err error

	// Поддерживаем только AES-128 (16 байт) и AES-256 (32 байта)
	if len(key) == 16 {
		block, err = aes.NewCipher(key)
	} else if len(key) == 32 {
		block, err = aes.NewCipher(key)
	} else {
		return nil
	}

	if err != nil {
		return nil
	}

	// Создаем CTR stream
	stream := cipher.NewCTR(block, nonce)
	plaintext := make([]byte, len(ciphertext))
	stream.XORKeyStream(plaintext, ciphertext)

	return plaintext
}
