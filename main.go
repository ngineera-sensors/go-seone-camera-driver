package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.neose-fsp-camera.gocv-driver/fspdriver"
)

func main() {

	optimalCameraShutter, err := fspdriver.CalibrateExposure(fspdriver.AEC_LOWER_BOUNDARY, fspdriver.AEC_UPPER_BOUNDARY)
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
	cmd.Process.Wait()
}
