package fspdriver

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"

	"gocv.io/x/gocv"
)

const (
	CAMERA_SAMPLE_PURGE_SIZE = 3
	CAMERA_SAMPLE_SIZE       = 3
)

const (
	CAMERA_FRAME_WIDTH  = 640
	CAMERA_FRAME_HEIGHT = 480
)

var (
	CAMERA_FRAMERATE = 30
)

func init() {
	if cameraFramerate := os.Getenv("CAMERA_FRAMERATE"); cameraFramerate != "" {
		log.Println("Setting CAMERA_FRAMERATE value provided in CAMERA_FRAMERATE env variable: ", CAMERA_FRAMERATE)
		CAMERA_FRAMERATE, _ = strconv.Atoi(cameraFramerate)
	}
}

func StartCamera(cameraFramerate, cameraShutter int) (*exec.Cmd, io.ReadCloser) {
	cmd := exec.Command(
		"libcamera-raw",
		"--camera", "0",
		"--width", "640",
		"--height", "480",
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
		log.Fatal(err)
	}
	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
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
	log.Println("Histogram: ", max, maxLoc)
	darkValue = byte(maxLoc.Y)

	return darkValue
}
