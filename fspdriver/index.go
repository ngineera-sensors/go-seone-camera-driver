package fspdriver

import (
	"bufio"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"gocv.io/x/gocv"
)

const (
	MMI_EXTRACTION_ELLIPSE_RADIUS int = 8
	MZI_N_NODES                   int = 64
	MMI_N_NODES                   int = MZI_N_NODES * 3
)

const (
	CAMERA_FRAME_WIDTH  = 640
	CAMERA_FRAME_HEIGHT = 480
)

var (
	MZI_MMI_MAP [64][3][]int = [64][3][]int{
		{{13, 14}, {15, 14}, {17, 14}}, // P0 (1) cba p
		{{19, 14}, {21, 14}, {23, 14}}, // P1 (2) cba p
		{{12, 15}, {14, 15}, {16, 15}}, // P2 (3) cba i
		{{18, 15}, {20, 15}, {22, 15}}, // P3 (4) cba i

		{{17, 12}, {15, 12}, {13, 12}}, // O0 (5) abc p
		{{23, 12}, {21, 12}, {19, 12}}, // 01 (6) abc p
		{{16, 13}, {14, 13}, {12, 13}}, // 02 (7) abc i
		{{22, 13}, {20, 13}, {18, 13}}, // O3 (8) abc i

		{{13, 10}, {15, 10}, {17, 10}}, // N0 (9) cba p
		{{19, 10}, {21, 10}, {23, 10}}, // N1 (10) cba p
		{{12, 11}, {14, 11}, {16, 11}}, // N2 (11) cba i
		{{18, 11}, {20, 11}, {22, 11}}, // N3 (12) cba i

		{{17, 8}, {15, 8}, {13, 8}}, // M0 (13) abc p
		{{23, 8}, {21, 8}, {19, 8}}, // M1 (14) abc p
		{{16, 9}, {14, 9}, {12, 9}}, // M2 (15) abc i
		{{22, 9}, {20, 9}, {18, 9}}, // M3 (16) abc i

		{{13, 6}, {15, 6}, {17, 6}}, // L0 (17) cba p
		{{19, 6}, {21, 6}, {23, 6}}, // L1 (18) cba p
		{{12, 7}, {14, 7}, {16, 7}}, // L2 (19) cba i
		{{18, 7}, {20, 7}, {22, 7}}, // L3 (20) cba i

		{{17, 4}, {15, 4}, {13, 4}}, // K0 (21) abc p
		{{23, 4}, {21, 4}, {19, 4}}, // K1 (22) abc p
		{{16, 5}, {14, 5}, {12, 5}}, // K2 (23) abc i
		{{22, 5}, {20, 5}, {18, 5}}, // K3 (24) abc i

		{{13, 2}, {15, 2}, {17, 2}}, // J0 (25) cba p
		{{19, 2}, {21, 2}, {23, 2}}, // J1 (26) cba p
		{{12, 3}, {14, 3}, {16, 3}}, // J2 (27) cba i
		{{18, 3}, {20, 3}, {22, 3}}, // J3 (28) cba i

		{{17, 0}, {15, 0}, {13, 0}}, // I0 (29) abc p
		{{23, 0}, {21, 0}, {19, 0}}, // I1 (30) abc p
		{{16, 1}, {14, 1}, {12, 1}}, // I2 (31) abc i
		{{22, 1}, {20, 1}, {18, 1}}, // I3 (32) abc i

		//

		{{0, 1}, {2, 1}, {4, 1}},  // H0 (33) cba p
		{{6, 1}, {8, 1}, {10, 1}}, // H1 (34) cba p
		{{1, 0}, {3, 0}, {5, 0}},  // H2 (35) cba i
		{{7, 0}, {9, 0}, {11, 0}}, // H3 (36) cba i

		{{4, 3}, {2, 3}, {0, 3}},  // G0 (37) abc p
		{{10, 3}, {8, 3}, {6, 3}}, // G1 (38) abc p
		{{5, 2}, {3, 2}, {1, 2}},  // G2 (39) abc i
		{{11, 2}, {9, 2}, {7, 2}}, // G3 (40) abc i

		{{0, 5}, {2, 5}, {4, 5}},  // F0 (41) cba p
		{{6, 5}, {8, 5}, {10, 5}}, // F1 (42) cba p
		{{1, 4}, {3, 4}, {5, 4}},  // F2 (43) cba i
		{{7, 4}, {9, 4}, {11, 4}}, // F3 (44) cba i

		{{4, 7}, {2, 7}, {0, 7}},  // E0 (45) abc p
		{{10, 7}, {8, 7}, {6, 7}}, // E1 (46) abc p
		{{5, 6}, {3, 6}, {1, 6}},  // E2 (47) abc i
		{{11, 6}, {9, 6}, {7, 6}}, // E3 (48) abc i

		{{0, 9}, {2, 9}, {4, 9}},  // D0 (49) cba p
		{{6, 9}, {8, 9}, {10, 9}}, // D1 (50) cba p
		{{1, 8}, {3, 8}, {5, 8}},  // D2 (51) cba i
		{{7, 8}, {9, 8}, {11, 8}}, // D3 (52) cba i

		{{4, 11}, {2, 11}, {0, 11}},  // C0 (53) abc p
		{{10, 11}, {8, 11}, {6, 11}}, // C1 (54) abc p
		{{5, 10}, {3, 10}, {1, 10}},  // C2 (55) abc i
		{{11, 10}, {9, 10}, {7, 10}}, // C3 (56) abc p

		{{0, 13}, {2, 13}, {4, 13}},  // B0 (57) cba p
		{{6, 13}, {8, 13}, {10, 13}}, // B1 (58) cba p
		{{1, 12}, {3, 12}, {5, 12}},  // B2 (59) cba i
		{{7, 12}, {9, 12}, {11, 12}}, // B3 (60) cba i

		{{4, 15}, {2, 15}, {0, 15}},  // A0 (61) abc p
		{{10, 15}, {8, 15}, {6, 15}}, // A1 (62) abc p
		{{5, 14}, {3, 14}, {1, 14}},  // A2 (63) abc i
		{{11, 14}, {9, 14}, {7, 14}}, // A3 (64) abc i
	}
)

func publishImage(topic string, mat gocv.Mat, mqttClient mqtt.Client) error {
	// Publish image (jpg/base64)
	imgBuf, err := gocv.IMEncode(gocv.JPEGFileExt, mat)
	if err != nil {
		return err
	}
	imgBytes := imgBuf.GetBytes()
	var b64bytes []byte = make([]byte, base64.StdEncoding.EncodedLen(len(imgBytes)))
	base64.StdEncoding.Encode(b64bytes, imgBytes)
	mqttClient.Publish(topic, 2, false, b64bytes)
	return err
}

func publishJsonMsg(topic string, obj interface{}, mqttClient mqtt.Client) error {
	msg, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	mqttClient.Publish(topic, 2, false, msg)
	return err
}

func MainLoop() error {

	mqttClient := NewMQTTClient()

	r := bufio.NewReader(os.Stdin)
	// capture, err := gocv.OpenVideoCapture("tcp://127.0.0.1:8888")
	// if err != nil {
	// 	panic(err)
	// }
	t := time.Now()

	w := CAMERA_FRAME_WIDTH
	h := CAMERA_FRAME_HEIGHT

	var grid [MMI_N_NODES]GridNode
	var firstMZIs [MZI_N_NODES]float64
	var previousMZIs [MZI_N_NODES]float64
	var unwindedMZIs [MZI_N_NODES]float64
	var Ks [MZI_N_NODES]int

	var gridAcquired bool
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

	buf := make([]byte, w*h+w*h/2)

	for i := 0; ; i++ {
		ts := time.Now()
		_, err := io.ReadFull(r, buf)
		if err != nil {
			return err
		}
		if i == 0 {
			log.Printf("Time until first frame arrived: %.3f", float64(time.Since(t).Microseconds())/1e3)
			t = time.Now()
		}
		if i < 3 {
			continue
		}
		mat, err := gocv.NewMatFromBytes(h, w, gocv.MatTypeCV8UC1, buf[:w*h])
		if err != nil {
			return err
		}
		defer mat.Close()

		gocv.Threshold(mat, &mat, 15, 256, gocv.ThresholdToZero)

		if !gridAcquired {
			detectedGridNodes := DetectGridNodes(mat)
			grid = ComputeGrid(detectedGridNodes)
			for i, node := range grid {
				if node.X == 0 && node.Y == 0 {
					log.Fatalf("missing node: %d", i)
				}
			}
			SaveSpotsgrid(grid)
			gridAcquired = true
		}

		MMIs := ExtractMMIs(mat, grid)
		MZIs := ExtractMZIs(MMIs, grid)

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

		// Publish MZISfifts Frame
		mziShiftsFrame := Frame{
			I:         i,
			Timestamp: int(ts.UnixMilli()),
			Values:    MZIShifts[:],
		}
		publishJsonMsg("fspdriver/frames/mzi", mziShiftsFrame, mqttClient)

		// Publish MMIs Frame
		mmiFrame := Frame{
			I:         i,
			Timestamp: int(ts.UnixMilli()),
			Values:    MMIs[:],
		}
		publishJsonMsg("fspdriver/frames/mmi", mmiFrame, mqttClient)

		// publishImage("fspdriver/images/raw", mat, mqttClient)

		if i%30 == 0 {
			log.Println(i, meanMZIAcc/float64(len(MZIs)))
			drawingMat := gocv.NewMatWithSize(mat.Rows(), mat.Cols(), gocv.MatTypeCV8UC1)
			defer drawingMat.Close()
			mat.CopyTo(&drawingMat)
			gocv.CvtColor(drawingMat, &drawingMat, gocv.ColorGrayToBGR)

			DrawSpotsgridDebug(drawingMat, grid)
			publishImage("fspdriver/images/drawing", drawingMat, mqttClient)
			publishImage("fspdriver/images/raw", mat, mqttClient)
			drawingMat.Close()

		}
		mat.Close()
		log.Println(gocv.MatProfile.Count())
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
