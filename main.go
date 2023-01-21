package main

import (
	"log"
	"math"
	"os"
	"os/signal"
	"syscall"

	"go.neose-fsp-camera.gocv-driver/fspdriver"
	"gocv.io/x/gocv"
)

const (
	AEC_UPPER_BOUNDARY      = 3000
	AEC_LOWER_BOUNDARY      = 100
	AEC_MAX_VALUE_TARGET    = 150
	AEC_MAX_VALUE_TOLERANCE = 5
)

func startCameraAndSampleMaxValue(cameraShutter int) (int, error) {
	var err error
	var max int

	cmd, out := fspdriver.StartCamera(30, cameraShutter)
	defer func() {
		log.Println("Killing camera..")
		err = cmd.Process.Kill()
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Waiting camera..")
		state, err := cmd.Process.Wait()
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Camera state after killing and waiting: ", state.String())
	}()

	mat, err := fspdriver.SampleCamera(out)
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
func CalibrateExposure(lowerBoundary, upperBoundary int) (int, error) {
	var err error

	parameter := (lowerBoundary + upperBoundary) / 2

	value, err := startCameraAndSampleMaxValue(parameter)
	if err != nil {
		return parameter, err
	}
	diff := math.Abs(float64(AEC_MAX_VALUE_TARGET - value))
	log.Printf("ExposureCalibration. Parameter: %d, Value: %d; Diff: %.0f", parameter, value, diff)
	if diff < AEC_MAX_VALUE_TOLERANCE {
		return parameter, err
	}
	if value < AEC_MAX_VALUE_TARGET {
		return CalibrateExposure(parameter, upperBoundary)
	} else {
		return CalibrateExposure(lowerBoundary, parameter)
	}
}

func main() {

	optimalCameraShutter, err := CalibrateExposure(AEC_LOWER_BOUNDARY, AEC_UPPER_BOUNDARY)
	if err != nil {
		log.Fatal(err)
	}

	cancelChan := make(chan os.Signal, 1)
	signal.Notify(cancelChan, syscall.SIGTERM, syscall.SIGINT)

	cmd, out := fspdriver.StartCamera(fspdriver.CAMERA_FRAMERATE, optimalCameraShutter)

	mat, err := fspdriver.SampleCamera(out)
	if err != nil {
		log.Fatal(err)
	}
	grid, err := fspdriver.CalibrateSpotsGrid(mat)
	if err != nil {
		log.Fatal(err)
	}

	darkValue := fspdriver.CalibrateDarkValue(mat)

	mat.Close()
	go fspdriver.MainLoop(grid, darkValue, out)
	sig := <-cancelChan // Block until signal is received
	log.Println("Received signal:", sig.String())
	cmd.Process.Kill()
}
