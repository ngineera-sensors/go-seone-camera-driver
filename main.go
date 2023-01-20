package main

import (
	"log"

	"go.neose-fsp-camera.gocv-driver/fspdriver"
)

func main() {
	log.Fatal(fspdriver.MainLoop())

	// // Read image from disk
	// mat := gocv.IMRead(filepath.Join("compute", "original.bmp"), gocv.IMReadGrayScale)
	// if mat.Empty() {
	// 	panic("imread empty")
	// }
	// defer mat.Close()
	// detectedGridNodes := fspdriver.DetectGridNodes(mat)
	// log.Printf("Detected %d grid nodes", len(detectedGridNodes))

	// grid := fspdriver.ComputeGrid(detectedGridNodes)

	// mmis := fspdriver.ExtractMMIs(mat, grid)
	// mzis := fspdriver.ExtractMZIs(mmis, grid)

	// f, err := os.Create("mzis.csv")
	// if err != nil {
	// 	panic(err)
	// }
	// defer f.Close()
	// csvW := csv.NewWriter(f)

	// fspdriver.WriteMZIsCSV(csvW, mzis)
	// csvW.Flush()
}
