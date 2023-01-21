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

	// Discard contours which area is
	// less than minimum
	// and more than maximum
	NODE_DETECTION_MIN_CONTOUR_AREA = 5
	NODE_DETECTION_MAX_CONTOUR_AREA = 200

	// Initial image dilation kernel size
	NODE_DETECTION_DILATION_KERNEL_SIZE = 3

	// Grid nodes detection relies on primary nodes being sorted
	// in row-major, left-to-right manner
	NODE_DETECTION_NODE_INTERLACE_GAP = 10

	// Minimum number of contours to be detected before grid calculation
	// 100 is a little bit more than half (192 MMIs in total)
	NODE_DETECTION_MINIMUM_PRIMARY_CONTOURS = 100

	NODE_DETECTION_COMMON_ANGLE_SEARCH_ARC_DEG  = 5
	NODE_DETECTION_COMMON_ANGLE_SEARCH_STEP_DEG = 0.1
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

// findCommonAngleRad will linearly search
// for the angle
// at which the grid is pivoted on the image
// TODO: implement bilateral minimum search instead of linear sweep
func findCommonAngleRad(
	aroundAngleRad float64,
	angleToSweepRad float64,
	angleStepRad float64,
	gridNodes []GridNode,
) float64 {
	var commonAngle float64
	var angleStepsInt int = int(math.Round(angleToSweepRad / angleStepRad))
	// These are what we search for
	var commonAngleEnergy float64 = 0
	for angleIdx := -angleStepsInt; angleIdx < angleStepsInt; angleIdx++ {
		var min float64 = math.MaxFloat64  // init min variable as a very high one
		var max float64 = -math.MaxFloat64 // init max variable as a very low one
		// Define the theta
		theta := aroundAngleRad + float64(angleIdx)*angleStepRad
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
		binWidth := step / 3     // Bins overlap 1/3 one upon another (semi-sliding bin)
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

func DrawSpotsgridDebug(mat gocv.Mat, grid [MMI_N_NODES]GridNode) {

	for nodeI, node := range grid {
		gocv.Ellipse(
			&mat,
			image.Pt(node.X, node.Y),
			image.Pt(MMI_EXTRACTION_ELLIPSE_RADIUS, MMI_EXTRACTION_ELLIPSE_RADIUS),
			0, 0, 360,
			color.RGBA{R: 255, G: 0, B: 255, A: 255},
			1,
		)
		var mziIdx int
		var mmiL int
	LoopMZI:
		for i, mzi := range MZI_MMI_GRID_MAP {
			for l, mmi := range mzi {
				if node.Row == mmi[0] && node.Col == mmi[1] {
					mziIdx = i
					mmiL = l
					break LoopMZI
				}
			}
		}
		mziRow := mziIdx / 4
		mziCol := mziIdx % 4
		mziRowStr := string("PONMLKJIHGFEDCBA"[mziRow])

		gocv.PutText(
			&mat,
			fmt.Sprintf("%d[%d:%d]", nodeI, node.Row, node.Col),
			image.Pt(node.X+2, node.Y-4),
			gocv.FontHersheyPlain,
			0.7,
			color.RGBA{R: 255, G: 255, B: 0, A: 255},
			1,
		)
		gocv.PutText(
			&mat,
			fmt.Sprintf("[%s%d]%d%s", mziRowStr, mziCol, mziIdx, [3]string{"a", "b", "c"}[mmiL]),
			image.Pt(node.X+2, node.Y+4),
			gocv.FontHersheyPlain,
			0.7,
			color.RGBA{R: 255, G: 255, B: 0, A: 255},
			1,
		)
	}
}

func computeFullGrid(detectedGridNodes []GridNode) ([MMI_N_NODES]GridNode, error) {
	var err error
	var grid [MMI_N_NODES]GridNode

	HorizontalAngleRad := findCommonAngleRad(
		0, // 0 for horizontal axis
		deg2Rad(NODE_DETECTION_COMMON_ANGLE_SEARCH_ARC_DEG),
		deg2Rad(NODE_DETECTION_COMMON_ANGLE_SEARCH_STEP_DEG),
		detectedGridNodes,
	)
	VerticalAngleRad := findCommonAngleRad(
		math.Pi/2, // 90 for vertical axis
		deg2Rad(NODE_DETECTION_COMMON_ANGLE_SEARCH_ARC_DEG),
		deg2Rad(NODE_DETECTION_COMMON_ANGLE_SEARCH_STEP_DEG),
		detectedGridNodes,
	)

	forwardEffectiveAngleRad := (HorizontalAngleRad + VerticalAngleRad - math.Pi/2) / 2
	backwardEffectiveAngleRad := -forwardEffectiveAngleRad
	log.Printf("Grid's horizontal angle: %.2f; Grid's vertical angle: %.2f. Effective Angle: %.2f", rad2Deg(HorizontalAngleRad), rad2Deg(VerticalAngleRad), rad2Deg(forwardEffectiveAngleRad))

	var minX float64 = math.MaxFloat64
	var maxX float64 = -math.MaxFloat64
	var minY float64 = math.MaxFloat64
	var maxY float64 = -math.MaxFloat64

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
	return grid, err
}

func detectPrimaryGridNodes(mat gocv.Mat) ([]GridNode, error) {

	var err error
	gridNodes := make([]GridNode, 0)

	if ok := gocv.IMWrite(filepath.Join("compute", "original.bmp"), mat); !ok {
		log.Println("DetectPrimaryGridNodes: original.bmp imwrite nok")
	}

	_min, _max, _, _ := gocv.MinMaxLoc(mat)
	log.Println("DetectPrimaryGridNodes: mat min/max: ", _min, _max)

	dilatedMat := gocv.NewMatWithSize(mat.Rows(), mat.Cols(), gocv.MatTypeCV8UC1)
	defer dilatedMat.Close()

	gocv.Dilate(
		mat,
		&dilatedMat,
		gocv.GetStructuringElement(gocv.MorphRect, image.Pt(NODE_DETECTION_DILATION_KERNEL_SIZE, NODE_DETECTION_DILATION_KERNEL_SIZE)),
	)

	if ok := gocv.IMWrite(filepath.Join("compute", "dilated_mat.bmp"), dilatedMat); !ok {
		log.Println("DetectPrimaryGridNodes: dilated_mat.bmp imwrite nok")
	}

	compareMat := gocv.NewMatWithSize(mat.Rows(), mat.Cols(), gocv.MatTypeCV8UC1)
	defer compareMat.Close()

	gocv.Compare(mat, dilatedMat, &compareMat, gocv.CompareGE)
	gocv.BitwiseNot(compareMat, &compareMat)

	if ok := gocv.IMWrite(filepath.Join("compute", "compare_mat.bmp"), compareMat); !ok {
		log.Println("DetectPrimaryGridNodes: compare_mat.bmp imwrite nok")
	}

	// Detect Contours
	contours := gocv.FindContours(compareMat, gocv.RetrievalTree, gocv.ChainApproxSimple)
	log.Printf("Found %d contours", contours.Size())

	if contours.Size() < NODE_DETECTION_MINIMUM_PRIMARY_CONTOURS {
		err = fmt.Errorf("not enough contours detected: %d", contours.Size())
		return gridNodes, err
	}

	thresholdedMatchResultWithEllipses := gocv.NewMatWithSize(mat.Rows(), mat.Cols(), gocv.MatTypeCV8UC1)
	defer thresholdedMatchResultWithEllipses.Close()
	compareMat.CopyTo(&thresholdedMatchResultWithEllipses)
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

		if node2.X < node1.X-NODE_DETECTION_NODE_INTERLACE_GAP || node2.X > node1.X+NODE_DETECTION_NODE_INTERLACE_GAP {
			return node1.X < node2.X
		} else {
			return node1.Y < node2.Y
		}
	})

	for i, gridNode := range gridNodes {
		// log.Printf("Ellipse %d; Position: %v", i, ellipse.Center)
		gocv.Ellipse(&thresholdedMatchResultWithEllipses, image.Pt(gridNode.X, gridNode.Y), image.Pt(MMI_EXTRACTION_ELLIPSE_RADIUS, MMI_EXTRACTION_ELLIPSE_RADIUS), 0, 0, 360, color.RGBA{R: 0, G: 0, B: 255, A: 127}, 2)
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
		log.Println("DetectPrimaryGridNodes: thresholded_matching_result_with_detected_ellipses.bmp imwrite nok")
	}
	return gridNodes, err
}

func CalibrateSpotsGrid(mat gocv.Mat) ([MMI_N_NODES]GridNode, error) {
	var err error
	var gridNodes [MMI_N_NODES]GridNode

	primaryGridNodes, err := detectPrimaryGridNodes(mat)
	if err != nil {
		return gridNodes, err
	}
	return computeFullGrid(primaryGridNodes)
}

func SaveSpotsgrid(grid [MMI_N_NODES]GridNode) {
	// f, err := os.Create(fmt.Sprintf("%d.spotsgrid.csv", time.Now().UnixMilli()))
	f, err := os.Create("spotsgrid.csv")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	csvW := csv.NewWriter(f)
	csvW.Write([]string{"row", "col", "x", "y"})
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
