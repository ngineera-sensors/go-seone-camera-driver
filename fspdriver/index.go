package fspdriver

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"gocv.io/x/gocv"
)

const (
	CAMERA_GET_STATE_MQTT_TOPIC_PATH    = "/camera/state/get"
	CAMERA_GET_STATE_CB_MQTT_TOPIC_PATH = "/camera/state/get/cb"

	CAMERA_SET_STATE_MQTT_TOPIC_PATH    = "/camera/state/set"
	CAMERA_SET_STATE_CB_MQTT_TOPIC_PATH = "/camera/state/set/cb"

	CAMERA_GET_FRAMERATE_MQTT_TOPIC_PATH    = "/camera/framerate/get"
	CAMERA_GET_FRAMERATE_CB_MQTT_TOPIC_PATH = "/camera/framerate/get/cb"

	CAMERA_SET_FRAMERATE_MQTT_TOPIC_PATH    = "/camera/framerate/set"
	CAMERA_SET_FRAMERATE_CB_MQTT_TOPIC_PATH = "/camera/framerate/set/cb"

	CAMERA_GET_CALIBRATION_MQTT_TOPIC_PATH    = "/camera/calibration/get"
	CAMERA_GET_CALIBRATION_CB_MQTT_TOPIC_PATH = "/camera/calibration/get/cb"

	// CAMERA_PERFORM_CALIBRATION_MQTT_TOPIC_PATH = "/camera/perform_calibration"
	// CAMERA_PERFORM_CALIBRATION_CB_MQTT_TOPIC_PATH = "/camera/perform_calibration/cb"

	CAMERA_GET_IMAGE_MQTT_TOPIC_PATH    = "/camera/get_image"
	CAMERA_GET_IMAGE_CB_MQTT_TOPIC_PATH = "/camera/get_image/cb"

	CAMERA_GET_DRAWING_MQTT_TOPIC_PATH    = "/camera/get_drawing"
	CAMERA_GET_DRAWING_CB_MQTT_TOPIC_PATH = "/camera/get_drawing/cb"

	CAMERA_MMI_BROADCAST_MQTT_TOPIC_PATH = "/camera/mmi/broadcast"
	CAMERA_MZI_BROADCAST_MQTT_TOPIC_PATH = "/camera/mzi/broadcast"
)

var (
	SEONE_SN = ""
)

func init() {
	sn, err := os.ReadFile(filepath.Join("config", "serialnumber.txt"))
	if err != nil {
		log.Fatal(err)
	}
	if len(sn) != 0 {
		log.Printf("Setting SEONE_SN value: %s", string(sn))
	}
	snStr := string(sn)
	snStr = strings.TrimSpace(snStr)
	SEONE_SN = snStr
	if SEONE_SN == "" {
		log.Fatal("Could not get seone's SN, exiting..")
	}
}

func CameraPipeAndLoop(stateChan chan CameraState, imageTriggerChan chan bool, client mqtt.Client) error {
	initialParameter := AEC_EFFECTIVE_SHUTTER_SPEED
	if initialParameter == 0 {
		initialParameter = (AEC_LOWER_BOUNDARY + AEC_UPPER_BOUNDARY) / 2
	}
	_, err := CalibrateExposure(AEC_LOWER_BOUNDARY, initialParameter, AEC_UPPER_BOUNDARY, 0)
	if err != nil {
		log.Fatal(err)
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM, syscall.SIGINT)

	cmd, stdoutReader := StartCamera(CAMERA_FRAMERATE_MUT, AEC_EFFECTIVE_SHUTTER_SPEED)
	mat, err := SampleCamera(stdoutReader)
	if err != nil {
		log.Fatal(err)
	}
	_, err = CalibrateSpotsGrid(mat)
	if err != nil {
		log.Fatal(err)
	}

	CalibrateDarkValue(mat)

	mat.Close()

	go MainLoop(client, stdoutReader, imageTriggerChan)

	select {
	case sig := <-signalChan: // Block until signal is received
		log.Println("Received signal:", sig.String())
		StopCamera(cmd)
		os.Exit(0)

	case state := <-stateChan:
		if state == 0 {
			log.Println("Received OFF state")
			StopCamera(cmd)
			signal.Stop(signalChan)
			break
		}
	}
	log.Println("Exiting CameraPipeAndLoop..")
	return err
}

func GetCameraStateHandler(client mqtt.Client, msg mqtt.Message) {
	var err error
	respTopic := getFullTopicString(CAMERA_GET_STATE_CB_MQTT_TOPIC_PATH)

	respObj := MQTTResponse{
		Message: CameraStateMessage{
			State: CAMERA_STATE_MUT,
		},
	}
	err = PublishJsonMsg(respTopic, respObj, client)
	if err != nil {
		log.Printf("Error occurred in GetCameraStateHandler MQTT CB: %s", err.Error())
	}
}

func SetCameraStateHandler(stateChan chan CameraState, imageTriggerChan chan bool) mqtt.MessageHandler {

	var f = func(client mqtt.Client, msg mqtt.Message) {
		var err error
		respTopic := getFullTopicString(CAMERA_SET_STATE_CB_MQTT_TOPIC_PATH)

		payload := msg.Payload()
		var state CameraStateMessage
		err = json.Unmarshal(payload, &state)
		if err != nil {
			log.Printf("Error occurred in SetCameraFramerateHandler MQTT CB while unmarshalling the JSON message: %s", err.Error())
		}
		log.Printf("Setting CAMERA_STATE to %d", state.State)

		switch state.State {

		case 0:
			stateChan <- 0
			break
		case 1:
			if CAMERA_STATE_MUT != 0 {
				log.Println("Received SET_STATE=1 but camera is already in running state. ignoring..")
				break
			}
			go CameraPipeAndLoop(stateChan, imageTriggerChan, client)
			break
		default:
			log.Printf("Received invalid State value for CAMERA_STATE: %d. Must be either 0 either 1.", state.State)
			return
		}

		respObj := MQTTResponse{
			Message: CameraStateMessage{
				State: CAMERA_STATE_MUT,
			},
		}
		err = PublishJsonMsg(respTopic, respObj, client)
		if err != nil {
			log.Printf("Error occurred in SetCameraFramerateHandler MQTT CB: %s", err.Error())
		}
	}
	return f
}

func GetCameraFramerateHandler(client mqtt.Client, msg mqtt.Message) {
	var err error
	respTopic := getFullTopicString(CAMERA_GET_FRAMERATE_CB_MQTT_TOPIC_PATH)

	respObj := MQTTResponse{
		Message: CameraFramerateMessage{
			Framerate: CAMERA_FRAMERATE_MUT,
		},
	}
	err = PublishJsonMsg(respTopic, respObj, client)
	if err != nil {
		log.Printf("Error occurred in GetCameraFramerateHandler MQTT CB: %s", err.Error())
	}
}

func SetCameraFramerateHandler(stateChan chan CameraState, imageTriggerChan chan bool) mqtt.MessageHandler {

	var f = func(client mqtt.Client, msg mqtt.Message) {
		var err error
		respTopic := getFullTopicString(CAMERA_SET_FRAMERATE_CB_MQTT_TOPIC_PATH)

		payload := msg.Payload()
		var framerate CameraFramerateMessage
		err = json.Unmarshal(payload, &framerate)
		if err != nil {
			log.Printf("Error occurred in SetCameraFramerateHandler MQTT CB while unmarshalling the JSON message: %s", err.Error())
		}
		log.Printf("Setting CAMERA_FRAMERATE to %d", framerate.Framerate)

		CAMERA_FRAMERATE_MUT = framerate.Framerate

		stateChan <- 0
		go CameraPipeAndLoop(stateChan, imageTriggerChan, client)

		respObj := MQTTResponse{
			Message: CameraFramerateMessage{
				Framerate: CAMERA_FRAMERATE_MUT,
			},
		}
		err = PublishJsonMsg(respTopic, respObj, client)
		if err != nil {
			log.Printf("Error occurred in SetCameraFramerateHandler MQTT CB: %s", err.Error())
		}
	}
	return f
}

func GetCalibrationHandler(client mqtt.Client, msg mqtt.Message) {
	var err error
	respTopic := getFullTopicString(CAMERA_GET_CALIBRATION_CB_MQTT_TOPIC_PATH)

	respObj := MQTTResponse{
		Message: CameraCalibrationMessage{
			TargetMaxValue:        AEC_MAX_VALUE_TARGET,
			EffectiveMaxValue:     AEC_EFFECTIVE_MAX_VALUE,
			EffectiveShutterSpeed: AEC_EFFECTIVE_SHUTTER_SPEED,
			EffectiveDarkValue:    AEC_EFFECTIVE_DARK_VALUE,
			EffectiveGrid:         NODE_DETECTION_EFFECTIVE_GRID,
		},
	}
	err = PublishJsonMsg(respTopic, respObj, client)
	if err != nil {
		log.Printf("Error occurred in GetCalibrationHandler MQTT CB: %s", err.Error())
	}
}

func GetImageHandler(stateChan chan CameraState, imageTriggerChan chan bool) mqtt.MessageHandler {

	var f = func(client mqtt.Client, msg mqtt.Message) {
		imageTriggerChan <- true
	}
	return f
}

func SetupMQTTSubscriptionCallbacks(stateChan chan CameraState, imageTriggerChan chan bool, client mqtt.Client) {
	var topic string

	// State
	topic = getFullTopicString(CAMERA_GET_STATE_MQTT_TOPIC_PATH)
	log.Printf("Subscribing to Camera GET_STATE: %s", topic)
	client.Subscribe(topic, DEFAULT_QOS, GetCameraStateHandler)

	topic = getFullTopicString(CAMERA_SET_STATE_MQTT_TOPIC_PATH)
	log.Printf("Subscribing to Camera SET_STATE: %s", topic)
	client.Subscribe(topic, DEFAULT_QOS, SetCameraStateHandler(stateChan, imageTriggerChan))

	// Framerate
	topic = getFullTopicString(CAMERA_GET_FRAMERATE_MQTT_TOPIC_PATH)
	log.Printf("Subscribing to Camera GET_FRAMERATE: %s", topic)
	client.Subscribe(topic, DEFAULT_QOS, GetCameraFramerateHandler)

	topic = getFullTopicString(CAMERA_SET_FRAMERATE_MQTT_TOPIC_PATH)
	log.Printf("Subscribing to Camera SET_FRAMERATE: %s", topic)
	client.Subscribe(topic, DEFAULT_QOS, SetCameraFramerateHandler(stateChan, imageTriggerChan))

	// Calibration
	topic = getFullTopicString(CAMERA_GET_CALIBRATION_MQTT_TOPIC_PATH)
	log.Printf("Subscribing to Camera GET_CALIBRATION: %s", topic)
	client.Subscribe(topic, DEFAULT_QOS, GetCalibrationHandler)
	// Calibration is performed on each SET_CAMERA=1, no need to implement a separate command

	// Image
	topic = getFullTopicString(CAMERA_GET_IMAGE_MQTT_TOPIC_PATH)
	log.Printf("Subscribing to Camera GET_IMAGE: %s", topic)
	client.Subscribe(topic, DEFAULT_QOS, GetImageHandler(stateChan, imageTriggerChan))

}

func MainLoop(client mqtt.Client, reader io.ReadCloser, imageTriggerChan chan bool) error {

	r := bufio.NewReader(reader)
	t0 := time.Now()

	w := CAMERA_FRAME_WIDTH
	h := CAMERA_FRAME_HEIGHT

	var firstMZIs [MZI_N_NODES]float64
	var previousMZIs [MZI_N_NODES]float64
	var unwindedMZIs [MZI_N_NODES]float64
	var Ks [MZI_N_NODES]int

	var firstMZIsAcquired bool

	grid := NODE_DETECTION_EFFECTIVE_GRID
	darkValue := AEC_EFFECTIVE_DARK_VALUE

	// mzif, err := os.Create("mzis.csv")
	// if err != nil {
	// 	panic(err)
	// }
	// defer mzif.Close()

	// csvWMZI := csv.NewWriter(mzif)

	// mmif, err := os.Create("mmis.csv")
	// if err != nil {
	// 	panic(err)
	// }
	// defer mmif.Close()
	// csvWMMI := csv.NewWriter(mmif)

	// NV12 (YUV4:2:0) camera bayer grid format is composed of 1 luma plane and
	// 1/2 chroma plane
	// http://www.chiark.greenend.org.uk/doc/linux-doc-3.16/html/media_api/re29.html

	fullBuf := make([]byte, w*h+w*h/2)

	previousPublishTs := time.Now()
	for i := 0; ; i++ {
		// ts := int((time.Since(t0)).Milliseconds())
		_, err := io.ReadFull(r, fullBuf)
		if err != nil {
			return err
		}
		if i == 0 {
			log.Printf("Time until first frame arrived: %.3f", float64(time.Since(t0).Microseconds())/1e3)
			t0 = time.Now()
		}

		buf := fullBuf[:w*h]

		MMIs := ExtractMMIsBuffer(buf, grid, darkValue)
		MZIs := ExtractMZIsIndexed(MMIs, grid)

		if !firstMZIsAcquired {
			firstMZIs = MZIs
			previousMZIs = MZIs
			firstMZIsAcquired = true
			continue
		}

		for i := range MZIs {
			dv := MZIs[i] - previousMZIs[i]
			if math.Abs(dv) > math.Pi {
				if dv < 0 {
					Ks[i]++
				} else {
					Ks[i]--
				}
			}
		}

		for i := range MZIs {
			unwindedMZIs[i] = MZIs[i] + 2*math.Pi*float64(Ks[i])
		}

		previousMZIs = MZIs

		var MZIShifts [MZI_N_NODES]float64
		for i, mzi := range unwindedMZIs {
			MZIShifts[i] = mzi - firstMZIs[i]
		}

		var meanMZIAcc float64
		for _, mzi := range MZIShifts {
			meanMZIAcc += mzi
		}

		if time.Since(previousPublishTs).Milliseconds() < 333 {
			continue
		}
		previousPublishTs = time.Now()

		log.Println(i, meanMZIAcc/float64(len(MZIs)))

		// WriteCSV(csvWMMI, MMIs[:])
		// WriteCSV(csvWMZI, MZIShifts[:])

		ts := int(time.Now().UnixMilli())
		// Publish MZISfifts Frame
		mziShiftsFrame := Frame{
			I:         i,
			Timestamp: ts,
			Values:    MZIShifts[:],
		}
		topicMZI := getFullTopicString(CAMERA_MZI_BROADCAST_MQTT_TOPIC_PATH)
		err = PublishJsonMsg(topicMZI, mziShiftsFrame, client)
		if err != nil {
			log.Println(err)
		}

		// Publish MMIs Frame
		mmiFrame := Frame{
			I:         i,
			Timestamp: ts,
			Values:    MMIs[:],
		}
		topicMMI := getFullTopicString(CAMERA_MZI_BROADCAST_MQTT_TOPIC_PATH)
		err = PublishJsonMsg(topicMMI, mmiFrame, client)
		if err != nil {
			log.Println(err)
		}
		select {
		case <-imageTriggerChan:
			// Raw image
			topicImage := getFullTopicString(CAMERA_GET_IMAGE_CB_MQTT_TOPIC_PATH)
			mat, err := gocv.NewMatFromBytes(h, w, gocv.MatTypeCV8UC1, buf)
			if err != nil {
				log.Println(err)
				mat.Close()
				break
			}
			err = PublishImage(topicImage, mat, client)
			mat.Close()
			if err != nil {
				log.Println(err)
			}

			// Image with debug symbols ("drawing")

			drawingMat := gocv.NewMatWithSize(h, w, gocv.MatTypeCV8UC1)
			mat.CopyTo(&drawingMat)
			gocv.CvtColor(drawingMat, &drawingMat, gocv.ColorGrayToBGR)
			DrawSpotsgridDebug(drawingMat, grid)

			topicDrawing := getFullTopicString(CAMERA_GET_DRAWING_CB_MQTT_TOPIC_PATH)
			err = PublishImage(topicDrawing, drawingMat, client)
			drawingMat.Close()
			if err != nil {
				log.Println(err)
			}
		default:
			break
		}
	}
}

func WriteCSV(csvW *csv.Writer, values []float64) {
	var valuesStrings []string = make([]string, len(values))
	for i, mzi := range values {
		valuesStrings[i] = fmt.Sprint(mzi)
	}
	csvW.Write(valuesStrings)
	csvW.Flush()
}
