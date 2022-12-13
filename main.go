package main

import (
	"encoding/csv"
	"log"
	"os"
	"path/filepath"

	"go.neose-fsp-camera.gocv-driver/fspdriver"
	"gocv.io/x/gocv"
)

func main() {
	// fspdriver.MainLoop()
	// Read image from disk
	mat := gocv.IMRead(filepath.Join("output", "luma_image_3.jpg"), gocv.IMReadGrayScale)
	if mat.Empty() {
		panic("imread empty")
	}
	defer mat.Close()
	detectedGridNodes := fspdriver.DetectGridNodes(mat)
	log.Printf("Detected %d grid nodes", len(detectedGridNodes))

	grid := fspdriver.ComputeGrid(detectedGridNodes)

	mmis := fspdriver.ExtractMMIs(mat, grid)
	mzis := fspdriver.ExtractMZIs(mmis, grid)

	f, err := os.Create("mzis.csv")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	csvW := csv.NewWriter(f)

	fspdriver.WriteMZIsCSV(csvW, mzis)
	csvW.Flush()
}
