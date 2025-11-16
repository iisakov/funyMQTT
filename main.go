package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func main() {
	opts := mqtt.NewClientOptions()
	opts.AddBroker("tcp://mqtt.skobk.in:1883")
	opts.SetClientID("go_mqtt_client_" + fmt.Sprint(time.Now().Unix()))
	opts.SetDefaultPublishHandler(MessageHandler)
	opts.OnConnect = ConnectHandler
	opts.OnConnectionLost = ConnectionLostHandler
	opts.SetCleanSession(true)
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(10 * time.Second)

	if User != "" && User != "..." {
		opts.SetUsername(User)
		if Password != "" && Password != "..." {
			opts.SetPassword(Password)
		}
	}

	client := mqtt.NewClient(opts)
	fmt.Println("🔗 Подключаемся к брокеру...")

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Printf("❌ Ошибка подключения: %v\n", token.Error())
		return
	}

	// Подписка на топик
	topic := "msh/RU/ARKH/#"
	if token := client.Subscribe(topic, 1, nil); token.Wait() && token.Error() != nil {
		fmt.Printf("❌ Ошибка подписки на топик %s: %v\n", topic, token.Error())
		return
	}

	fmt.Printf("✅ Успешно подписались на топик: %s\n", topic)
	fmt.Println("📡 Ожидаем сообщения... (Ctrl+C для выхода)")

	// Тестовое декодирование (опционально)
	// testDecoding()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n🛑 Завершение работы...")
	client.Unsubscribe(topic)
	client.Disconnect(250)
}
