package main

import (
	"encoding/hex"
	"log"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var MessageHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	// Сохраняем сырые данные в файл
	saveRawData(msg.Topic(), msg.Payload())

	log.Printf("Saved message from topic: %s", msg.Topic())
}

var ConnectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	log.Println("Connected to MQTT broker")
}

var ConnectionLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	log.Printf("Connection lost: %v", err)
}

func saveRawData(topic string, payload []byte) {
	timestamp := time.Now().Format("01.02.2006 15:04:05")
	filename := "raw_messages.txt"

	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Error opening file: %v", err)
		return
	}
	defer file.Close()

	// Сохраняем в формате: timestamp | topic | hex_data
	hexData := hex.EncodeToString(payload)
	line := timestamp + " | " + topic + " | " + hexData + "\n"

	if _, err := file.WriteString(line); err != nil {
		log.Printf("Error writing to file: %v", err)
	}
}
