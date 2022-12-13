package fspdriver

import (
	"encoding/csv"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"

	"gocv.io/x/gocv"
)

const (
	NODE_DETECTION_TEMPLATE_SIZE       int     = 15
	NODE_DETECTION_DOT_SIZE            int     = 5
	NODE_DETECTION_MIN_CONTOUR_AREA    float64 = 5
	NODE_DETECTION_MAX_CONTOUR_AREA    float64 = 200
	NODE_DETECTION_THRESHOLD_OFFSET_FR float64 = 0.15
)

func computeBorders(a []float64) []float64 {
	// log.Println("Computing borders. Initial elements:", len(a))

	var diffA []float64
	sort.Float64s(a)

	for i := 1; i < len(a); i++ {
		var d = math.Abs(a[i] - a[i-1])
		diffA = append(diffA, d)
	}
	var bordersA []float64

	var deltaMaxAcc float64
	for _, d := range diffA {
		deltaMaxAcc = math.Max(deltaMaxAcc, d)
	}
	thresh := deltaMaxAcc / 2
	// log.Println("Effective threshold: ", thresh)

	var cAcc float64
	var cCnt int

	for i, d := range diffA {
		if cCnt < 1 {
			cAcc += a[i]
			cCnt++
			continue
		}
		// log.Printf("A: %.2f, D: %.2f", a[i], d)
		if d > thresh {
			// log.Println("\n")
			bordersA = append(bordersA, cAcc/float64(cCnt)) // Mean
			cAcc = 0
			cCnt = 0
		} else {
			cAcc += a[i]
			cCnt++
		}
		i++
	}
	bordersA = append(bordersA, a[len(a)-1]) // Add last one because it cannot be detected by the dx
	sort.Float64s(bordersA)
	return bordersA
}

func computeCommonAngleRad(
	aroundAngleRad float64,
	angleToSweepRad float64,
	angleStepRad float64,
	gridNodes []GridNode,
) float64 {
	var commonAngle float64
	var angleStepsInt int = int(math.Round(angleToSweepRad / angleStepRad))
	// These are what we search for
	var commonAngleEnergy float64 = 0
	for angleNb := -angleStepsInt; angleNb < angleStepsInt; angleNb++ {
		var min float64 = 1e6  // init min variable as a very high one
		var max float64 = -1e6 // init max variable as a very low one
		// Define the theta
		theta := aroundAngleRad + float64(angleNb)*angleStepRad
		var rs []float64
		for _, gridNode := range gridNodes {
			// Calculate r as a function of just defined theta
			r := float64(gridNode.X)*math.Cos(theta) + float64(gridNode.Y)*math.Sin(theta)
			// Use the loop to get the range of r values
			min = math.Min(min, r)
			max = math.Max(max, r)
			rs = append(rs, r)
		}
		step := (max - min) / 90 // divide into 90 bins // TODO: Optimize this hard-coded params
		binWidth := step / 3     // Bins overlap 3 times one upon another (semi-sliding bin)
		var binEnergySum float64 = 0
		// Calculte the enery of each bin
		for bin := min; bin < max; bin += binWidth {
			var binPopulation float64 = 0
			for _, r := range rs {
				if bin <= r && r < bin+step {
					binPopulation++
				}
			}
			binEnergySum += math.Pow(binPopulation, 4)
		}
		// log.Printf("Bin energy sum for angleNb %d: %.2f", angleNb, binEnergySum)
		// Save the value of commonAngle if its energy is higher than previously captured one
		if binEnergySum > commonAngleEnergy {
			commonAngle = theta
			commonAngleEnergy = binEnergySum
		}
	}

	return commonAngle
}

func rad2Deg(rad float64) float64 {
	return (180 / math.Pi) * rad
}

func deg2Rad(deg float64) float64 {
	return (math.Pi / 180) * deg
}

func pivot(x int, y int, pivotPointX int, pivotPointY int, angleRad float64) (int, int) {
	// Subtract the pivot point
	centeredX := pivotPointX - x
	centeredY := pivotPointY - y

	// Rotate
	pivotedCenteredX := int(float64(centeredX)*math.Cos(angleRad) - float64(centeredY)*math.Sin(angleRad))
	pivotedCenteredY := int(float64(centeredX)*math.Sin(angleRad) + float64(centeredY)*math.Cos(angleRad))

	// Add up the pivot point back
	pivotedX := pivotPointX - pivotedCenteredX
	pivotedY := pivotPointY - pivotedCenteredY

	return pivotedX, pivotedY
}

func DrawSpotsgridMZIMap(mat *gocv.Mat, grid []GridNode) {
	for _, node := range grid {

		gocv.Ellipse(mat, image.Pt(node.X, node.Y), image.Pt(MMI_EXTRACTION_ELLIPSE_RADIUS, MMI_EXTRACTION_ELLIPSE_RADIUS), 0, 0, 360, color.RGBA{R: 255, G: 0, B: 255, A: 255}, 1)

		var mziIdx int
		var mmiL int
	LoopMZI:
		for i, mzi := range MZI_MMI_MAP {
			for l, mmi := range mzi {
				if node.Row == mmi[0] && node.Col == mmi[1] {
					mziIdx = i
					mmiL = l
					break LoopMZI
				}
			}
		}

		gocv.PutText(
			mat,
			fmt.Sprintf("%d%s", mziIdx, [3]string{"a", "b", "c"}[mmiL]),
			image.Pt(node.X+5, node.Y-5),
			gocv.FontHersheyPlain,
			1,
			color.RGBA{R: 255, G: 127, B: 255, A: 255},
			1,
		)
	}
	// if ok := gocv.IMWrite(filepath.Join("compute", "mat_grid_mzi.bmp"), secondMat); !ok {
	// 	panic("imwrite nok")
	// }
}

// func DrawSpotsgridDebug(mat *gocv.Mat, gri []GridNode) {

// 	actualMat := gocv.NewMatWithSize(mat.Rows(), mat.Cols(), gocv.MatTypeCV8UC1)
// 	defer actualMat.Close()
// 	gocv.CvtColor(mat, &actualMat, gocv.ColorBGRToGray)
// 	actualMat.SubtractUChar(15)

// 	secondMat := gocv.NewMatWithSize(mat.Rows(), mat.Cols(), gocv.MatTypeCV8UC3)
// 	defer secondMat.Close()
// 	mat.CopyTo(&secondMat)

// 	for _, detectedGridNode := range grid {
// 		gocv.Ellipse(&mat, image.Pt(detectedGridNode.X, detectedGridNode.Y), image.Pt(10, 10), 0, 0, 360, color.RGBA{R: 255, G: 255, B: 255, A: 255}, 1)
// 	}

// 	for _, pivotedGridNode := range pivotedGridNodes {
// 		gocv.Ellipse(&mat, image.Pt(pivotedGridNode.X, pivotedGridNode.Y), image.Pt(5, 5), 0, 0, 360, color.RGBA{R: 0, G: 255, B: 0, A: 255}, 1)
// 	}

// 	if ok := gocv.IMWrite(filepath.Join("compute", "mat_withellipsesAnd_pivoted_ellipses.bmp"), mat); !ok {
// 		panic("imwrite nok")
// 	}
// }

func ComputeGrid(detectedGridNodes []GridNode) [MMI_N_NODES]GridNode {

	var grid [MMI_N_NODES]GridNode

	HorizontalAngleRad := computeCommonAngleRad(
		0,
		deg2Rad(5),
		deg2Rad(0.1),
		detectedGridNodes,
	)
	VerticalAngleRad := computeCommonAngleRad(
		math.Pi/2,
		deg2Rad(5),
		deg2Rad(0.1),
		detectedGridNodes,
	)

	forwardEffectiveAngleRad := (HorizontalAngleRad + VerticalAngleRad - math.Pi/2) / 2
	backwardEffectiveAngleRad := -forwardEffectiveAngleRad
	log.Printf("Grid's horizontal angle: %.2f; Grid's vertical angle: %.2f. Effective Angle: %.2f", rad2Deg(HorizontalAngleRad), rad2Deg(VerticalAngleRad), rad2Deg(forwardEffectiveAngleRad))

	var minX float64 = 1e6
	var maxX float64 = -1e6
	var minY float64 = 1e6
	var maxY float64 = -1e6

	for _, gridNode := range detectedGridNodes {
		minX = math.Min(minX, float64(gridNode.X))
		maxX = math.Max(maxX, float64(gridNode.X))

		minY = math.Min(minY, float64(gridNode.Y))
		maxY = math.Max(maxY, float64(gridNode.Y))
	}

	// log.Println(minX, maxX, minY, maxY)

	gridCenterX := int(minX + (maxX-minX)/2)
	gridCenterY := int(minY + (maxY-minY)/2)

	// log.Println("Grid center", gridCenterX, gridCenterY)

	var pivotedGridNodes []GridNode
	for _, gridNode := range detectedGridNodes {
		pivotedX, pivotedY := pivot(gridNode.X, gridNode.Y, gridCenterX, gridCenterY, backwardEffectiveAngleRad)
		pivotedGridNodes = append(pivotedGridNodes, GridNode{
			X: pivotedX,
			Y: pivotedY,
		})
		// log.Printf("Pivoted the ellipse. Before: %v; After: %v", ellipse.Center, pivotedEllipse.Center)
	}

	// Populate known grid projections on X and Y
	// Use pivoted ones to robustify grid search
	// Will unpivot back later
	var pivotedX []float64
	var pivotedY []float64
	for _, pivotedGridNode := range pivotedGridNodes {
		pivotedX = append(pivotedX, float64(pivotedGridNode.X))
		pivotedY = append(pivotedY, float64(pivotedGridNode.Y))
	}

	// Calculate borders
	ProjectionsX := computeBorders(pivotedX)
	ProjectionsY := computeBorders(pivotedY)

	log.Printf("X projected borders: %d; Y projected borders: %d", len(ProjectionsX), len(ProjectionsY))

	var n int
	for i := 0; i < len(ProjectionsX); i++ {
		var j int
		// Alternating rows
		if i%2 == 0 {
			j = 1
		}
		for ; j < len(ProjectionsY); j += 2 {
			var x = ProjectionsX[i]
			var y = ProjectionsY[j]

			unPivotedX, unPivotedY := pivot(
				int(math.Round(x)),
				int(math.Round(y)),
				gridCenterX,
				gridCenterY,
				forwardEffectiveAngleRad,
			)

			node := GridNode{
				X:   unPivotedX,
				Y:   unPivotedY,
				Row: j,
				Col: i,
			}
			grid[n] = node
			n++
		}
	}
	return grid
}

func DetectGridNodes(mat gocv.Mat) []GridNode {

	thresholdedMatchResult := gocv.NewMatWithSize(mat.Rows(), mat.Cols(), gocv.MatTypeCV8UC1)
	defer thresholdedMatchResult.Close()

	// gocv.AdaptiveThreshold(mat, &matchResult, 256, gocv.AdaptiveThresholdMean, gocv.ThresholdBinaryInv, 3, 0)
	gocv.Threshold(mat, &thresholdedMatchResult, 127, 255, gocv.ThresholdBinary)

	if ok := gocv.IMWrite(filepath.Join("compute", "thresholded_matching_result.bmp"), thresholdedMatchResult); !ok {
		panic("imwrite nok")
	}

	gocv.MorphologyEx(
		thresholdedMatchResult,
		&thresholdedMatchResult,
		gocv.MorphOpen,
		gocv.GetStructuringElement(gocv.MorphRect, image.Pt(3, 3)),
	)

	if ok := gocv.IMWrite(filepath.Join("compute", "thresholded_matching_result_opened.bmp"), thresholdedMatchResult); !ok {
		panic("imwrite nok")
	}

	// Detect Contours
	contours := gocv.FindContours(thresholdedMatchResult, gocv.RetrievalTree, gocv.ChainApproxSimple)
	log.Printf("Found %d contours", contours.Size())

	gridNodes := make([]GridNode, 0)

	thresholdedMatchResultWithEllipses := gocv.NewMatWithSize(mat.Rows(), mat.Cols(), gocv.MatTypeCV8UC1)
	defer thresholdedMatchResultWithEllipses.Close()
	thresholdedMatchResult.CopyTo(&thresholdedMatchResultWithEllipses)
	gocv.CvtColor(thresholdedMatchResultWithEllipses, &thresholdedMatchResultWithEllipses, gocv.ColorGrayToBGRA)

	var j int
	for i := 0; i < contours.Size(); i++ {
		contour := contours.At(i)
		area := gocv.ContourArea(contour)
		if area > float64(NODE_DETECTION_MAX_CONTOUR_AREA) || area < NODE_DETECTION_MIN_CONTOUR_AREA {
			continue
		}
		hull := gocv.NewMat()
		gocv.ConvexHull(contour, &hull, true, true)
		m := gocv.Moments(hull, true)
		cX := int(m["m10"] / m["m00"])
		cY := int(m["m01"] / m["m00"])
		gridNodes = append(gridNodes, GridNode{
			X: cX,
			Y: cY,
		})
		j++
	}

	sort.SliceStable(gridNodes, func(i, j int) bool {
		node1 := gridNodes[i]
		node2 := gridNodes[j]

		//TODO: magic number "10"  should reflect matcher template size

		if node2.X < node1.X-10 || node2.X > node1.X+10 {
			return node1.X < node2.X
		} else {
			return node1.Y < node2.Y
		}
	})

	for i, gridNode := range gridNodes {
		// log.Printf("Ellipse %d; Position: %v", i, ellipse.Center)
		gocv.Ellipse(&thresholdedMatchResultWithEllipses, image.Pt(gridNode.X, gridNode.Y), image.Pt(10, 10), 0, 0, 360, color.RGBA{R: 0, G: 0, B: 255, A: 127}, 2)
		gocv.PutText(
			&thresholdedMatchResultWithEllipses,
			fmt.Sprint(i),
			image.Pt(gridNode.X+5, gridNode.Y-5),
			gocv.FontHersheyPlain,
			1,
			color.RGBA{R: 255, G: 0, B: 0, A: 255},
			2,
		)
	}

	if ok := gocv.IMWrite(filepath.Join("compute", "thresholded_matching_result_with_detected_ellipses.bmp"), thresholdedMatchResultWithEllipses); !ok {
		panic("imwrite nok")
	}

	return gridNodes
}

func SaveSpotsgrid(grid []GridNode) {
	// f, err := os.Create(fmt.Sprintf("%d.spotsgrid.csv", time.Now().UnixMilli()))
	f, err := os.Create("spotsgrid.csv")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	csvW := csv.NewWriter(f)
	for _, node := range grid {
		csvW.Write([]string{
			fmt.Sprint(node.Row),
			fmt.Sprint(node.Col),
			fmt.Sprint(node.X),
			fmt.Sprint(node.Y),
		})
	}
	csvW.Flush()
}
