package main

import (
	"log"
	"time"

	"go.neose-fsp-camera.gocv-driver/fspdriver"
)

func main() {
	var stateChan chan fspdriver.CameraState = make(chan fspdriver.CameraState, 1)
	var imageTriggerChan chan bool = make(chan bool, 1)

	client, err := fspdriver.NewMQTTClient()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("MQTT Client: ", client.IsConnected())

	fspdriver.SetupMQTTSubscriptionCallbacks(stateChan, imageTriggerChan, client)

	go fspdriver.CameraPipeAndLoop(stateChan, imageTriggerChan, client)

	for {
		time.Sleep(1 * time.Second)
	}
}
