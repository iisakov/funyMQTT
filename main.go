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
	// Настройка подключения
	opts := mqtt.NewClientOptions()
	opts.AddBroker(Broker)
	opts.SetClientID("go_mqtt_client_" + fmt.Sprint(time.Now().Unix()))
	opts.SetDefaultPublishHandler(MessageHandler)
	opts.OnConnect = ConnectHandler
	opts.OnConnectionLost = ConnectionLostHandler

	// Добавляем аутентификацию, если указаны учетные данные
	if User != "" {
		opts.SetUsername(User)
		if Password != "" {
			opts.SetPassword(Password)
		}
	}

	// Подключение
	client := mqtt.NewClient(opts)
	fmt.Println("🔗 Подключаемся к брокеру...")

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Printf("❌ Ошибка подключения: %v\n", token.Error())
		fmt.Println("Возможные причины:")
		fmt.Println("1. Неправильный логин/пароль")
		fmt.Println("2. Брокер требует аутентификацию")
		fmt.Println("3. Проблемы с сетью")
		return
	}

	// Подписка на топик
	if token := client.Subscribe(Topic, 1, nil); token.Wait() && token.Error() != nil {
		fmt.Printf("❌ Ошибка подписки на топик %s: %v\n", Topic, token.Error())
		return
	}

	fmt.Printf("✅ Успешно подписались на топик: %s\n", Topic)
	fmt.Println("📡 Ожидаем сообщения... (Ctrl+C для выхода)")

	// Ожидание завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n🛑 Завершение работы...")
	client.Unsubscribe(Topic)
	client.Disconnect(250)
}
