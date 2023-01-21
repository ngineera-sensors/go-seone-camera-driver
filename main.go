package main

import (
	"log"
	"math"

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
	defer cmd.Process.Kill()

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

	_, out := fspdriver.StartCamera(fspdriver.CAMERA_FRAMERATE, optimalCameraShutter)

	mat, err := fspdriver.SampleCamera(out)
	if err != nil {
		log.Fatal(err)
	}
	grid, err := fspdriver.CalibrateSpotsGrid(mat)
	if err != nil {
		log.Fatal(err)
	}
	fspdriver.SaveSpotsgrid(grid)

	hist := gocv.NewMatWithSize(1, 256, gocv.MatTypeCV8UC1)
	mask := gocv.Ones(mat.Rows(), mat.Cols(), gocv.MatTypeCV8UC1)
	gocv.CalcHist([]gocv.Mat{mat}, []int{0}, mask, &hist, []int{256}, []float64{0, 256}, false)

	_, max, _, maxLoc := gocv.MinMaxLoc(hist)
	log.Println("Histogram: ", max, maxLoc)

	mat.Close()
	log.Fatal(fspdriver.MainLoop(grid, out))
}
