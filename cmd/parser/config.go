package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func init() {
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	User = os.Getenv("MQTT_USER")
	Password = os.Getenv("MQTT_PASSWORD")
	Broker = os.Getenv("MQTT_BROKER")
	Topic = os.Getenv("MQTT_TOPIC")
}

var User string
var Password string
var Broker string
var Topic string
