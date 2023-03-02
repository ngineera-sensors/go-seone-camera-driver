# SeOne camera driver and MMI/MZI extraction module
## Requirements
Requires the libcamera stack to be present on host device. Installation how-to's
* https://docs.arducam.com/Raspberry-Pi-Camera/Native-camera/Libcamera-User-Guide/
* https://libcamera.org/getting-started.html


Also requires OpenCV bindings for Go: GoCV: https://github.com/hybridgroup/gocv. Pay attention to version, currently built on **OpenCV version 4.6 (GoCV v. 0.31)**

## Configuration
* Libcamera executable config is hardcoded in fspdriver.StartCamera()
* Framerate: CAMERA_FRAMERATE = 10
* Automatic exposure configuration (AEC) parameters (hardcoded):
    * AEC_UPPER_BOUNDARY      = 3000
	* AEC_LOWER_BOUNDARY      = 100
	* AEC_MAX_VALUE_TARGET    = 150
	* AEC_MAX_VALUE_TOLERANCE = 5
	* AEC_MAX_NB_TRIALS       = 5
* MQTT client parameters:
    * MQTT_SCHEME   = "tcp"
	* MQTT_HOST     = "localhost"
	* MQTT_PORT     = "1883"
	* MQTT_USERNAME = ""
	* MQTT_PASSWORD = ""
	* DEFAULT_QOS byte = 2
* Node detection parameters (hardcoded) are in the fspdriver/spotsgrid.go

## MQTT callbacks and broadcasting topics
```
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

	// Calibration is performed each time CAMERA_STATE is turned from 0 to 1
	// No need to implement these callbacks
	// CAMERA_PERFORM_CALIBRATION_MQTT_TOPIC_PATH = "/camera/perform_calibration"
	// CAMERA_PERFORM_CALIBRATION_CB_MQTT_TOPIC_PATH = "/camera/perform_calibration/cb"

	CAMERA_GET_IMAGE_MQTT_TOPIC_PATH    = "/camera/get_image"
	CAMERA_GET_IMAGE_CB_MQTT_TOPIC_PATH = "/camera/get_image/cb"

	CAMERA_GET_DRAWING_MQTT_TOPIC_PATH    = "/camera/get_drawing"
	CAMERA_GET_DRAWING_CB_MQTT_TOPIC_PATH = "/camera/get_drawing/cb"

	CAMERA_MMI_BROADCAST_MQTT_TOPIC_PATH = "/camera/mmi/broadcast"
	CAMERA_MZI_BROADCAST_MQTT_TOPIC_PATH = "/camera/mzi/broadcast"
)
```
