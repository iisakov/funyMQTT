package main

import (
	"fmt"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var MessageHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	dm, _ := DecodeMessage(msg.Payload())
	fmt.Printf("📨 Топик: %s\nСообщение: %s\n\n", msg.Topic(), dm)
}

var ConnectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("✅ Подключение к MQTT брокеру установлено")
}

var ConnectionLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("❌ Соединение потеряно: %v\n", err)
}
