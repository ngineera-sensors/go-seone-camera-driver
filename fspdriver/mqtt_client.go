package fspdriver

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"gocv.io/x/gocv"
)

var (
	MQTT_SCHEME   = "tcp"
	MQTT_HOST     = ""
	MQTT_PORT     = "1883"
	MQTT_USERNAME = ""
	MQTT_PASSWORD = ""
)

var f mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("TOPIC: %s\n", msg.Topic())
	fmt.Printf("MSG: %s\n", msg.Payload())
}

func init() {
	mqttScheme := os.Getenv("MQTT_SCHEME")
	if mqttScheme != "" {
		log.Printf("Setting MQTT_SCHEME value provided in MQTT_SCHEME env variable : %s", mqttScheme)
		MQTT_SCHEME = mqttScheme
	}
	mqttHost := os.Getenv("MQTT_HOST")
	if mqttHost != "" {
		log.Printf("Setting MQTT_HOST value provided in MQTT_HOST env variable : %s", mqttHost)
		MQTT_HOST = mqttHost
	}
	if MQTT_HOST == "" {
		log.Fatalf("MQTT_HOST is not defined neither by default nor manually. Exiting.")
	}
	mqttPort := os.Getenv("MQTT_PORT")
	if mqttPort != "" {
		log.Printf("Setting MQTT_PORT value provided in MQTT_PORT env variable : %s", mqttPort)
		MQTT_PORT = mqttPort
	}
	mqttUsername := os.Getenv("MQTT_USERNAME")
	if mqttUsername != "" {
		log.Println("Setting MQTT_USERNAME value provided in MQTT_USERNAME env variable : ***")
		MQTT_USERNAME = mqttUsername
	}
	mqttPassword := os.Getenv("MQTT_PASSWORD")
	if mqttPassword != "" {
		log.Println("Setting MQTT_PASSWORD value provided in MQTT_PASSWORD env variable : ***")
		MQTT_PASSWORD = mqttPassword
	}
}

func NewMQTTClient() (mqtt.Client, error) {
	var err error

	mqttBrokerUri := fmt.Sprintf("%s://%s:%s", MQTT_SCHEME, MQTT_HOST, MQTT_PORT)
	// mqttClientID := fmt.Sprintf("seone_%s", SEONE_SN)
	mqttClientID := "gotrivial"

	log.Printf("Connecting to MQTT Broker: %s. ClientID: %s", mqttBrokerUri, mqttClientID)

	opts := mqtt.
		NewClientOptions().
		AddBroker(mqttBrokerUri).
		SetClientID(mqttClientID).
		SetUsername(MQTT_USERNAME).
		SetPassword(MQTT_PASSWORD)
	opts.SetPingTimeout(3 * time.Second)

	c := mqtt.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		return c, token.Error()
	}

	return c, err
}

func publishImage(topic string, mat gocv.Mat, mqttClient mqtt.Client) error {
	// Publish image (jpg/base64)
	imgBuf, err := gocv.IMEncode(gocv.JPEGFileExt, mat)
	if err != nil {
		return err
	}
	imgBytes := imgBuf.GetBytes()
	var b64bytes []byte = make([]byte, base64.StdEncoding.EncodedLen(len(imgBytes)))
	base64.StdEncoding.Encode(b64bytes, imgBytes)
	mqttClient.Publish(topic, 2, false, b64bytes)
	return err
}

func publishJsonMsg(topic string, obj interface{}, mqttClient mqtt.Client) error {
	msg, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	mqttClient.Publish(topic, 2, false, msg)
	return err
}
