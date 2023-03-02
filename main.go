package main

import (
	"time"

	"go-seone-camera-driver/fspdriver"
)

func main() {
	var stateChan chan fspdriver.CameraState = make(chan fspdriver.CameraState, 1)
	var imageTriggerChan chan bool = make(chan bool, 1)

	client, err := fspdriver.NewMQTTClient()
	if err != nil {
		fspdriver.ERRORLogger.Fatal(err)
	}

	fspdriver.SetupMQTTSubscriptionCallbacks(stateChan, imageTriggerChan, client)

	go fspdriver.CameraPipeAndLoop(stateChan, imageTriggerChan, client)

	for {
		time.Sleep(1 * time.Second)
	}
}
