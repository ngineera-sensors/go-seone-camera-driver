package fspdriver

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"math"
	"time"
)

// var (
// 	SEONE_SN = ""
// )

// func init() {
// 	sn, err := os.ReadFile(filepath.Join("config", "serialnumber.txt"))
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	if len(sn) != 0 {
// 		log.Printf("Setting SEONE_SN value: %s", string(sn))
// 	}
// 	snStr := string(sn)
// 	snStr = strings.TrimSpace(snStr)
// 	SEONE_SN = snStr
// }

func MainLoop(grid [MMI_N_NODES]GridNode, darkValue byte, out io.ReadCloser) error {

	// mqttClient, err := NewMQTTClient()
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// f, err := os.Open("2023-01-10-18-08-raw-video.bin")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	r := bufio.NewReader(out)
	t0 := time.Now()

	w := CAMERA_FRAME_WIDTH
	h := CAMERA_FRAME_HEIGHT

	var firstMZIs [MZI_N_NODES]float64
	var previousMZIs [MZI_N_NODES]float64
	var unwindedMZIs [MZI_N_NODES]float64
	var Ks [MZI_N_NODES]int

	var firstMZIsAcquired bool

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

		// // Publish MZISfifts Frame
		// mziShiftsFrame := Frame{
		// 	I:         i,
		// 	Timestamp: ts,
		// 	Values:    MZIShifts[:],
		// }
		// publishJsonMsg("fspdriver/frames/mzi", mziShiftsFrame, mqttClient)

		// // Publish MMIs Frame
		// mmiFrame := Frame{
		// 	I:         i,
		// 	Timestamp: ts,
		// 	Values:    MMIs[:],
		// }
		// publishJsonMsg("fspdriver/frames/mmi", mmiFrame, mqttClient)

		// publishImage("fspdriver/images/raw", mat, mqttClient)

		// if i%30 == 0 {

		// 	drawingMat := gocv.NewMatWithSize(mat.Rows(), mat.Cols(), gocv.MatTypeCV8UC1)
		// 	defer drawingMat.Close()
		// 	mat.CopyTo(&drawingMat)
		// 	gocv.CvtColor(drawingMat, &drawingMat, gocv.ColorGrayToBGR)

		// 	DrawSpotsgridDebug(drawingMat, grid)
		// 	publishImage("fspdriver/images/drawing", drawingMat, mqttClient)
		// 	publishImage("fspdriver/images/raw", mat, mqttClient)
		// 	drawingMat.Close()

		// }

		// log.Println(gocv.MatProfile.Count())
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
