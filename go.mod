module fyneMMQT

go 1.25.1

require github.com/joho/godotenv v1.5.1

require (
	github.com/gorilla/websocket v1.5.3 // indirect
	golang.org/x/sync v0.17.0 // indirect
	google.golang.org/protobuf v1.36.10
)

require (
	github.com/eclipse/paho.mqtt.golang v1.5.1
	golang.org/x/net v0.44.0 // indirect
)

replace your-project-name/model/meshtastic/go/generated => ./model/meshtastic/go/generated
