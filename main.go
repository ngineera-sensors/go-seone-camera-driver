package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"go-seone-camera-driver/fspdriver"
)

const (
	USAGE = `
	Usage: %s
	Libcamera-powered Camera Driver for Seone. Broadcasts MMI/MZI values extracted from camera feed to MQTT broker.
	
	Options:
	`
)

func main() {

	serialNumberPathPtr := flag.String("s", "config/serialnumber.txt", "path to serialnumber txt file")
	imagesPath := flag.String("a", "images", "tcp binding addr")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), USAGE, os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if serialNumberPathPtr != nil {
		fspdriver.SEONE_SN_PATH = *serialNumberPathPtr
	}
	fspdriver.InitSerialNumber()

	if imagesPath != nil {
		fspdriver.NODE_DETECTION_IMAGES_PATH = *imagesPath
	}
	fspdriver.InitImagesPath()



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
