package fspdriver

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"strconv"

	"gocv.io/x/gocv"
)

const ()

const (
	CAMERA_FRAME_WIDTH  = 640
	CAMERA_FRAME_HEIGHT = 480
)

const (
	CAMERA_SAMPLE_PURGE_SIZE = 3
	CAMERA_SAMPLE_SIZE       = 3

	AEC_UPPER_BOUNDARY      = 3000
	AEC_LOWER_BOUNDARY      = 100
	AEC_MAX_VALUE_TARGET    = 150
	AEC_MAX_VALUE_TOLERANCE = 5
	AEC_MAX_NB_TRIALS       = 5
)

var (
	AEC_EFFECTIVE_MAX_VALUE          = 0
	AEC_EFFECTIVE_SHUTTER_SPEED      = 0
	AEC_EFFECTIVE_DARK_VALUE    byte = 0
)

var (
	CAMERA_STATE_MUT             CameraState = 0
	CAMERA_FRAMERATE_MUT                     = 10
	MZI_EXTRACTION_FRAMERATE_MUT             = 3
)

func init() {
	if cameraFramerate := os.Getenv("CAMERA_FRAMERATE"); cameraFramerate != "" {
		if LOG_LEVEL <= INFO_LEVEL {
			INFOLogger.Println("Setting CAMERA_FRAMERATE value provided in CAMERA_FRAMERATE env variable: ", CAMERA_FRAMERATE_MUT)
		}
		CAMERA_FRAMERATE_MUT, _ = strconv.Atoi(cameraFramerate)
	}
}

func startCameraAndSampleMaxValue(cameraShutter int) (int, error) {
	var err error
	var max int

	cmd, out := StartCamera(30, cameraShutter)
	defer func() {
		err = StopCamera(cmd)
		if err != nil {
			if LOG_LEVEL <= ERROR_LEVEL {
				ERRORLogger.Println(err)
			}
		}
	}()

	mat, err := SampleCamera(out)
	if err != nil {
		return max, err
	}
	_, maxF, _, _ := gocv.MinMaxIdx(mat)
	max = int(maxF)
	mat.Close()
	return max, err
}

// CalibrateExposure performs a binary search on camera
// image maxValue target CAMERA_IMAGE_MAX_VALUE_TARGET
// with tolerance of CAMERA_IMAGE_MAX_VALUE_TOLERANCE
func CalibrateExposure() error {
	initialParameter := AEC_EFFECTIVE_SHUTTER_SPEED
	if initialParameter == 0 {
		initialParameter = (AEC_LOWER_BOUNDARY + AEC_UPPER_BOUNDARY) / 2
	}
	_, err := exposureBinarySearch(AEC_LOWER_BOUNDARY, initialParameter, AEC_UPPER_BOUNDARY, 0)
	return err
}

func exposureBinarySearch(lowerBoundary, parameter, upperBoundary, i int) (int, error) {
	var err error

	if i > AEC_MAX_NB_TRIALS {
		if LOG_LEVEL <= WARNING_LEVEL {
			WARNINGLogger.Printf("ExposureCalibration: reached AEC_MAX_TRIES. ShutterSpeed: %d. MaxValue: %d", AEC_EFFECTIVE_SHUTTER_SPEED, AEC_EFFECTIVE_MAX_VALUE)
		}
		return parameter, err
	}

	value, err := startCameraAndSampleMaxValue(parameter)
	if err != nil {
		return parameter, err
	}

	AEC_EFFECTIVE_MAX_VALUE = value
	AEC_EFFECTIVE_SHUTTER_SPEED = parameter

	diff := math.Abs(float64(AEC_MAX_VALUE_TARGET - value))

	if LOG_LEVEL <= DEBUG_LEVEL {
		DEBUGLogger.Printf("ExposureCalibration: parameter: %d, value: %d; diff: %.0f", parameter, value, diff)
	}

	if diff < AEC_MAX_VALUE_TOLERANCE {
		return parameter, err
	}
	newParameter := (lowerBoundary + upperBoundary) / 2
	if value < AEC_MAX_VALUE_TARGET {
		return exposureBinarySearch(parameter, newParameter, upperBoundary, i+1)
	} else {
		return exposureBinarySearch(lowerBoundary, newParameter, parameter, i+1)
	}
}

func StartCamera(cameraFramerate, cameraShutter int) (*exec.Cmd, io.ReadCloser) {
	cmd := exec.Command(
		"libcamera-raw",
		"--camera", "0",
		"--width", fmt.Sprint(CAMERA_FRAME_WIDTH),
		"--height", fmt.Sprint(CAMERA_FRAME_HEIGHT),
		"--framerate", fmt.Sprint(cameraFramerate),
		"--flush", "1",
		"-t", "0",
		"--shutter", fmt.Sprint(cameraShutter),
		"--gain", "1",
		"--ev", "0",
		"--denoise", "off",
		"--contrast", "1",
		"-o", "-",
	)
	out, err := cmd.StdoutPipe()
	if err != nil {
		ERRORLogger.Fatal(err)
	}
	err = cmd.Start()
	if err != nil {
		ERRORLogger.Fatal(err)
	}
	CAMERA_STATE_MUT = 1
	return cmd, out
}

func SampleCamera(out io.ReadCloser) (gocv.Mat, error) {
	var err error

	w := CAMERA_FRAME_WIDTH
	h := CAMERA_FRAME_HEIGHT

	r := bufio.NewReader(out)
	buf := make([]byte, w*h+w*h/2)

	masterMat := gocv.Zeros(h, w, gocv.MatTypeCV16UC1)

	// Purge buffer for CAMERA_SAMPLE_PURGE_SIZE frames
	for i := 0; i < CAMERA_SAMPLE_PURGE_SIZE; i++ {
		_, err := io.ReadFull(r, buf)
		if err != nil {
			return masterMat, err
		}
	}
	// Accumulate CAMERA_SAMPLE_SIZE frames
	for i := 0; i < CAMERA_SAMPLE_SIZE; i++ {
		_, err := io.ReadFull(r, buf)
		if err != nil {
			return masterMat, err
		}
		mat, err := gocv.NewMatFromBytes(h, w, gocv.MatTypeCV8UC1, buf[:w*h])
		if err != nil {
			return masterMat, err
		}
		mat.ConvertTo(&mat, gocv.MatTypeCV16UC1)
		gocv.Add(mat, masterMat, &masterMat)
		mat.Close()
	}
	// Divide by CAMERA_SAMPLE_SIZE and convert back to 8U
	masterMat.DivideUChar(CAMERA_SAMPLE_SIZE)
	return masterMat, err
}

func CalibrateDarkValue(mat gocv.Mat) byte {
	var darkValue byte

	hist := gocv.NewMatWithSize(1, 256, gocv.MatTypeCV8UC1)
	mask := gocv.Ones(mat.Rows(), mat.Cols(), gocv.MatTypeCV8UC1)
	gocv.CalcHist([]gocv.Mat{mat}, []int{0}, mask, &hist, []int{256}, []float64{0, 256}, false)

	_, max, _, maxLoc := gocv.MinMaxLoc(hist)
	if LOG_LEVEL <= DEBUG_LEVEL {
		DEBUGLogger.Printf("Histogram: max=%.1f, maxLoc=%#v", max, maxLoc)
	}
	darkValue = byte(maxLoc.Y)

	AEC_EFFECTIVE_DARK_VALUE = darkValue

	return darkValue
}

func StopCamera(cmd *exec.Cmd) error {
	var err error
	if LOG_LEVEL <= DEBUG_LEVEL {
		DEBUGLogger.Println("Killing camera..")
	}
	err = cmd.Process.Kill()
	if err != nil {
		if LOG_LEVEL <= ERROR_LEVEL {
			ERRORLogger.Println(err)
		}
	}
	if LOG_LEVEL <= DEBUG_LEVEL {
		DEBUGLogger.Println("Waiting camera..")
	}
	state, err := cmd.Process.Wait()
	if err != nil {
		if LOG_LEVEL <= DEBUG_LEVEL {
			DEBUGLogger.Println(err)
		}
	}
	if LOG_LEVEL <= DEBUG_LEVEL {
		DEBUGLogger.Println("Camera state after killing and waiting: ", state.String())
	}
	CAMERA_STATE_MUT = 0
	return err
}
