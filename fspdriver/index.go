package fspdriver

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"gocv.io/x/gocv"
)

func CameraPipeAndLoop(stateChan chan CameraState, imageTriggerChan chan bool, client mqtt.Client) error {

	err := CalibrateExposure()
	if err != nil {
		ERRORLogger.Fatal(err)
	}
	if LOG_LEVEL <= INFO_LEVEL {
		INFOLogger.Printf("AEC completed. ShutterSpeed: %d, MaxValue: %d", AEC_EFFECTIVE_SHUTTER_SPEED, AEC_EFFECTIVE_MAX_VALUE)
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM, syscall.SIGINT)

	cmd, stdoutReader := StartCamera(CAMERA_FRAMERATE_MUT, AEC_EFFECTIVE_SHUTTER_SPEED)
	mat, err := SampleCamera(stdoutReader)
	if err != nil {
		ERRORLogger.Fatal(err)
	}
	_, err = CalibrateSpotsGrid(mat)
	if err != nil {
		ERRORLogger.Fatal(err)
	}

	CalibrateDarkValue(mat)

	mat.Close()

	go MainLoop(client, stdoutReader, imageTriggerChan)

	select {
	case sig := <-signalChan: // Block until signal is received
		if LOG_LEVEL <= WARNING_LEVEL {
			WARNINGLogger.Println("Received SIGNAL:", sig.String())
		}
		StopCamera(cmd)
		os.Exit(0)

	case state := <-stateChan:
		if state == 0 {
			if LOG_LEVEL <= INFO_LEVEL {
				INFOLogger.Println("Received OFF state")
			}
			StopCamera(cmd)
			signal.Stop(signalChan)
			break
		}
	}
	if LOG_LEVEL <= INFO_LEVEL {
		INFOLogger.Println("Exiting CameraPipeAndLoop..")
	}
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
		if LOG_LEVEL <= ERROR_LEVEL {
			ERRORLogger.Printf("Error occurred in GetCameraStateHandler MQTT CB: %s", err.Error())
		}
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
			if LOG_LEVEL <= ERROR_LEVEL {
				ERRORLogger.Printf("Error occurred in SetCameraFramerateHandler MQTT CB while unmarshalling the JSON message: %s", err.Error())
			}
		}
		if LOG_LEVEL <= INFO_LEVEL {
			INFOLogger.Printf("Setting CAMERA_STATE to %d", state.State)
		}

		switch state.State {

		case 0:
			stateChan <- 0
		case 1:
			if CAMERA_STATE_MUT != 0 {
				if LOG_LEVEL <= WARNING_LEVEL {
					WARNINGLogger.Println("Received SET_STATE=1 but camera is already in running state. Ignoring..")
				}
			}
			go CameraPipeAndLoop(stateChan, imageTriggerChan, client)
		default:
			if LOG_LEVEL <= WARNING_LEVEL {
				WARNINGLogger.Printf("Received invalid State value for CAMERA_STATE: %d. Must be either 0 either 1.", state.State)
			}
			return
		}

		respObj := MQTTResponse{
			Message: CameraStateMessage{
				State: CAMERA_STATE_MUT,
			},
		}
		err = PublishJsonMsg(respTopic, respObj, client)
		if err != nil {
			if LOG_LEVEL <= ERROR_LEVEL {
				ERRORLogger.Printf("Error occurred in SetCameraFramerateHandler MQTT CB: %s", err.Error())
			}
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
		if LOG_LEVEL <= ERROR_LEVEL {
			ERRORLogger.Printf("Error occurred in GetCameraFramerateHandler MQTT CB: %s", err.Error())
		}
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
			if LOG_LEVEL <= ERROR_LEVEL {
				ERRORLogger.Printf("Error occurred in SetCameraFramerateHandler MQTT CB while unmarshalling the JSON message: %s", err.Error())
			}
		}
		
		if LOG_LEVEL <= INFO_LEVEL {
			INFOLogger.Printf("Setting CAMERA_FRAMERATE to %d", framerate.Framerate)
		}

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
			if LOG_LEVEL <= ERROR_LEVEL {
				ERRORLogger.Printf("Error occurred in SetCameraFramerateHandler MQTT CB: %s", err.Error())
			}
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
		if LOG_LEVEL <= ERROR_LEVEL {
			ERRORLogger.Printf("Error occurred in GetCalibrationHandler MQTT CB: %s", err.Error())
		}
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
	if LOG_LEVEL <= INFO_LEVEL {
		INFOLogger.Printf("Subscribing to Camera GET_STATE: %s", topic)
	}
	client.Subscribe(topic, DEFAULT_QOS, GetCameraStateHandler)

	topic = getFullTopicString(CAMERA_SET_STATE_MQTT_TOPIC_PATH)
	if LOG_LEVEL <= INFO_LEVEL {
		INFOLogger.Printf("Subscribing to Camera SET_STATE: %s", topic)
	}
	client.Subscribe(topic, DEFAULT_QOS, SetCameraStateHandler(stateChan, imageTriggerChan))

	// Framerate
	topic = getFullTopicString(CAMERA_GET_FRAMERATE_MQTT_TOPIC_PATH)
	if LOG_LEVEL <= INFO_LEVEL {
		INFOLogger.Printf("Subscribing to Camera GET_FRAMERATE: %s", topic)
	}
	client.Subscribe(topic, DEFAULT_QOS, GetCameraFramerateHandler)

	topic = getFullTopicString(CAMERA_SET_FRAMERATE_MQTT_TOPIC_PATH)
	if LOG_LEVEL <= INFO_LEVEL {
		INFOLogger.Printf("Subscribing to Camera SET_FRAMERATE: %s", topic)
	}
	client.Subscribe(topic, DEFAULT_QOS, SetCameraFramerateHandler(stateChan, imageTriggerChan))

	// Calibration
	topic = getFullTopicString(CAMERA_GET_CALIBRATION_MQTT_TOPIC_PATH)
	if LOG_LEVEL <= INFO_LEVEL {
		INFOLogger.Printf("Subscribing to Camera GET_CALIBRATION: %s", topic)
	}
	client.Subscribe(topic, DEFAULT_QOS, GetCalibrationHandler)
	// Calibration is performed on each SET_CAMERA=1, no need to implement a separate command

	// Image
	topic = getFullTopicString(CAMERA_GET_IMAGE_MQTT_TOPIC_PATH)
	if LOG_LEVEL <= INFO_LEVEL {
		INFOLogger.Printf("Subscribing to Camera GET_IMAGE: %s", topic)
	}
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

	MZIShiftsAccumulatorTs := time.Now()

	var MZIShiftsAccumulator [MZI_N_NODES]float64
	var MZIShiftsAccumulatorCount int

	for i := 0; ; i++ {
		_, err := io.ReadFull(r, fullBuf)
		if err != nil {
			return err
		}
		if i == 0 {
			if LOG_LEVEL <= INFO_LEVEL {
				INFOLogger.Printf("Time until first frame arrived: %s", time.Since(t0).String())
			}
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

		durationSinceLastMZIShiftsBuffer := time.Since(MZIShiftsAccumulatorTs)
		// Accumulate MZI values during bufferred period
		if LOG_LEVEL <= DEBUG_LEVEL {
			DEBUGLogger.Println("Accumulating master", durationSinceLastMZIShiftsBuffer.String())
		}
		for i, mziValue := range MZIShifts {
			MZIShiftsAccumulator[i] += mziValue
		}
		MZIShiftsAccumulatorCount++
		if durationSinceLastMZIShiftsBuffer.Milliseconds() < int64(1000/MZI_EXTRACTION_FRAMERATE_MUT)-int64(1000/CAMERA_FRAMERATE_MUT) {
			continue
		}
		// Calculate the master (mean) mzi shifts
		// accumulated during the bufferred period
		var MZIShiftsMaster [MMI_N_NODES]float64
		for i, mziValue := range MZIShiftsAccumulator {
			MZIShiftsMaster[i] = mziValue / float64(MZIShiftsAccumulatorCount)
		}

		var globalMeanMZIAcc float64
		for _, mzi := range MZIShiftsMaster {
			globalMeanMZIAcc += mzi
		}
		globalMeanMaster := globalMeanMZIAcc / float64(len(MZIs))
		if LOG_LEVEL <= DEBUG_LEVEL {
			DEBUGLogger.Printf("I: %d; Global mean: %.2f; Accumulated %d in %s; Effective MMI FPS: %.2f",
				i, globalMeanMaster, MZIShiftsAccumulatorCount, durationSinceLastMZIShiftsBuffer.String(),
				float64(MZIShiftsAccumulatorCount)/(float64(durationSinceLastMZIShiftsBuffer.Milliseconds())/1e3),
			)
		}

		// Reset the accumulator and the counter
		MZIShiftsAccumulatorTs = time.Now()
		MZIShiftsAccumulatorCount = 0
		for i := range MZIShiftsAccumulator {
			MZIShiftsAccumulator[i] = 0
		}

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
			if LOG_LEVEL <= ERROR_LEVEL {
				ERRORLogger.Println(err)
			}
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
			if LOG_LEVEL <= ERROR_LEVEL {
				ERRORLogger.Println(err)
			}
		}
		select {
		case <-imageTriggerChan:
			// Raw image
			topicImage := getFullTopicString(CAMERA_GET_IMAGE_CB_MQTT_TOPIC_PATH)
			mat, err := gocv.NewMatFromBytes(h, w, gocv.MatTypeCV8UC1, buf)
			if err != nil {
				if LOG_LEVEL <= ERROR_LEVEL {
					ERRORLogger.Println(err)
				}
				mat.Close()
				break
			}
			err = PublishImage(topicImage, mat, client)
			mat.Close()
			if err != nil {
				if LOG_LEVEL <= ERROR_LEVEL {
					ERRORLogger.Println(err)
				}
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
				if LOG_LEVEL <= ERROR_LEVEL {
					ERRORLogger.Println(err)
				}
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
