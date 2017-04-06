// This page implements a 3D point cloud descriptor as described in this paper:
// https://www.researchgate.net/publication/293330421_A_fast_and_robust_local_descriptor_for_3D_point_cloud_registration
package points

import (
	"github.com/gonum/matrix/mat64"
	"image/color"
	"log"
	"math"
)

type plane struct {
	Center     mat64.Vector
	UnitNormal mat64.Vector
}

type neighborhood struct {
	// Embedded
	*mat64.Dense
	// Public
	Center mat64.Vector
	R      float64
	// Private
	normal   *mat64.Vector
	plane    *plane
	universe *PointCloudAnalyzer
}

// https://www.researchgate.net/publication/293330421_A_fast_and_robust_local_descriptor_for_3D_point_cloud_registration
type LFSHDescriptor struct {
	LocalDepthHistogram     map[int]int
	NormalDevianceHistogram map[int]int
	RadialDensityHistogram  map[int]int
}

type PointCloudAnalyzer struct {
	universe *mat64.Dense
	// Mapping of point in universe to precalculated neighborhood. Key is the
	// index of the point in the universe (which column the point is at).
	neighborhoods map[int]neighborhood
}

func (a *PointCloudAnalyzer) MakePointCloudAnalyzer(points *mat64.Dense) {
	a.universe = points
	a.neighborhoods = make(map[int]neighborhood)
}

// Calculates and returns the point's LFSH descriptor.
func (a *PointCloudAnalyzer) Descriptor(col int) LFSHDescriptor {
	n := a.getNeighborhood(col, searchRadius)
	return LFSHDescriptor{
		LocalDepthHistogram:     n.LocalDepthHistogram(),
		NormalDevianceHistogram: n.NormalDevianceHistogram(),
		RadialDensityHistogram:  n.RadialDensityHistogram(),
	}
}

// Visualizes an LFSH descriptor's three maps using color. The
// LocalDepthHistogram's weighted average is used to computed red component, the
// NormalDevianceHistogram is green, and the RadialDensityHistogram is for blue.
// This is meant for visualization and debugging.
func (d *LFSHDescriptor) VisualizeDescriptor() color.RGBA {
	var red_avg float32 = 0.0
	for key, value := range d.LocalDepthHistogram {
		red_avg += float32(key) / float32(numberDepthBuckets) * float32(value)
	}
	var green_avg float32 = 0.0
	for key, value := range d.NormalDevianceHistogram {
		green_avg += float32(key) / float32(numberAngularBuckets) * float32(value)
	}
	var blue_avg float32 = 0.0
	for key, value := range d.RadialDensityHistogram {
		blue_avg += float32(key) / float32(numberAnnuli) * float32(value)
	}
	return color.RGBA{
		R: uint8(red_avg * 255),
		G: uint8(green_avg * 255),
		B: uint8(blue_avg * 255),
		A: 255,
	}
}

const (
	// Number of buckets for the Local Depth Histogram (N1 in the paper linked
	// above).
	numberDepthBuckets = 10

	// Number of buckets for the Deviance Angle Histogram (N2 in the paper linked
	// above).
	numberAngularBuckets = 15

	// Number of buckets for the RadialDensityHistogram (N3 in the paper linked
	// above).
	numberAnnuli  = 5
	frobeniusNorm = 2

	// This value is used to set the size of the neighborhood sphere. In whatever
	// units the coordinate system is in.
	searchRadius = 0.1
)

// n = normal
// x = point in space (finding distance between this and plane p)
// c = point on plane ("center")
// d = distance to plane.
// n*(x - c) = 0
// ((a - d*n) - c) * n = 0
// a*n - d - c*n = 0
// d = a*n - n*c
func (p *plane) distanceToPoint(a mat64.Vector) float64 {
	return mat64.Dot(&a, &p.UnitNormal) - mat64.Dot(&p.UnitNormal, &p.Center)
}

func magnitudeSquared(v *mat64.Vector) float64 {
	sum := float64(0)
	r, _ := v.Dims()
	for i := 0; i < r; i++ {
		elem := v.At(i, 0)
		sum += elem * elem
	}
	return sum
}

func (a *PointCloudAnalyzer) getNeighborhood(col int, radius float64) neighborhood {
	n, ok := a.neighborhoods[col]
	if !ok {
		n = a.implGetNeighborhood(col, searchRadius)
		a.neighborhoods[col] = n
	}
	return n
}

func (a *PointCloudAnalyzer) implGetNeighborhood(col int, radius float64) neighborhood {
	points := a.universe
	_, c := points.Dims()
	point := points.ColView(col)
	neighborhood := &neighborhood{
		Dense:    mat64.NewDense(3, 0, nil),
		Center:   *point,
		R:        radius,
		universe: a,
	}
	diff := mat64.NewVector(3, []float64{0, 0, 0})
	neighborhoodSet := make(map[int]bool)
	for j := 0; j < c; j++ {
		column := points.ColView(j)
		diff.SubVec(column, point)
		distanceSquared := magnitudeSquared(diff)
		if distanceSquared <= radius*radius {
			neighborhoodSet[j] = true
		}
	}
	log.Println(len(neighborhoodSet))
	neighborhood.Dense = neighborhood.Grow(0, len(neighborhoodSet)).(*mat64.Dense)
	index := 0
	for k := range neighborhoodSet {
		*neighborhood.ColView(index) = *points.ColView(k)
		index++
	}
	return *neighborhood
}

func unit(v mat64.Vector) mat64.Vector {
	unitVector := *mat64.NewVector(3, []float64{0, 0, 0})
	unitVector.ScaleVec(1/mat64.Norm(&v, frobeniusNorm), &v)
	return unitVector
}

// A "neighborhood" is defined as the points inside the sphere centered around a
// point. The neighborhood has a plane which is derived by estimating the
// neighborhood's surface normal (see Neighborhood.Normal()).  This normal is
// then the normal of the plane, and the plane is tangent to the neighborhood
// sphere at the point on the sphere reached by traveling in the direction of
// the normal, starting from the center.
//
// The plane's "center" is guaranteed to be the center of the neighborhood
// sphere projected onto the plane.
func (n *neighborhood) Plane() plane {
	if n.plane != nil {
		return *n.plane
	}

	unitNormal := unit(n.Normal())
	center := *mat64.NewVector(3, []float64{0, 0, 0})
	center.AddScaledVec(&n.Center, n.R, &unitNormal)
	n.plane = &plane{
		UnitNormal: unitNormal,
		Center:     center,
	}
	return *n.plane
}

func (n *neighborhood) LocalDepthHistogram() map[int]int {
	histogram := make(map[int]int)
	projectionPlane := n.Plane()
	_, c := n.Dims()
	for j := 0; j < c; j++ {
		point := n.ColView(j)
		dist := projectionPlane.distanceToPoint(*point)
		bucket := int(math.Floor(dist / ((2 * n.R) / numberDepthBuckets)))
		histogram[bucket]++
	}
	return histogram
}

func (n *neighborhood) NormalDevianceHistogram() map[int]int {
	histogram := make(map[int]int)
	_, c := n.Dims()
	unitNormal := unit(n.Normal())
	for j := 0; j < c; j++ {
		otherNeighborhood := n.universe.getNeighborhood(j, n.R)
		otherUnitNormal := unit(otherNeighborhood.Normal())
		deviance := math.Acos(mat64.Dot(&unitNormal, &otherUnitNormal))
		bucket := int(math.Floor(deviance / ((math.Pi) / numberAngularBuckets)))
		histogram[bucket]++
	}
	return histogram
}

func (n *neighborhood) RadialDensityHistogram() map[int]int {
	histogram := make(map[int]int)
	projectionPlane := n.Plane()
	_, c := n.Dims()
	for j := 0; j < c; j++ {
		point := n.ColView(j)
		dist := projectionPlane.distanceToPoint(*point)
		projectedPoint := *mat64.NewVector(3, []float64{0, 0, 0})
		projectedPoint.AddScaledVec(point, dist, &projectionPlane.UnitNormal)
		displacement := *mat64.NewVector(3, []float64{0, 0, 0})
		displacement.SubVec(&projectionPlane.Center, &projectedPoint)
		projectedDistance := mat64.Norm(&displacement, frobeniusNorm)
		annulus := int(math.Floor(projectedDistance / (n.R / numberAnnuli)))
		histogram[annulus]++
	}
	return histogram
}

func sum(v []float64) float64 {
	sum := float64(0)
	for i := 0; i < len(v); i++ {
		sum += v[i]
	}
	return sum
}

func average(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	return sum(v) / float64(len(v))
}

func columnCovariance(a, b int, m mat64.Matrix) float64 {
	r, _ := m.Dims()

	sum := float64(0)

	aAvg := average(mat64.Col(nil, a, m))
	bAvg := average(mat64.Col(nil, b, m))

	for i := 0; i < r; i++ {
		sum += (m.At(i, a) - aAvg) * (m.At(i, b) - bAvg)
	}

	return sum / float64(r)
}

func covariance(m mat64.Matrix) mat64.Matrix {
	_, c := m.Dims()
	covMatrix := mat64.NewDense(c, c, nil)
	for i := 0; i < c; i++ {
		for j := i; j < c; j++ {
			cov := columnCovariance(i, j, m)
			covMatrix.Set(i, j, cov)
			covMatrix.Set(j, i, cov)
		}
	}
	return covMatrix
}

// Approximates the normal of a point cloud by getting the eigenvector of the
// covariance matrix with lowest magnitude.
func (n *neighborhood) Normal() mat64.Vector {
	if n.normal != nil {
		// Return cached value if it has already been calculated.
		return *n.normal
	}

	covMatrix := covariance(n.Dense.T())
	e := mat64.Eigen{}
	e.Factorize(covMatrix, true)
	eigenValues := e.Values(nil)
	mindex := 0
	for i := 0; i < len(eigenValues); i++ {
		if real(eigenValues[i]) < real(eigenValues[mindex]) {
			mindex = i
		}
	}

	meigenVector := e.Vectors().ColView(mindex)
	n.normal = meigenVector
	return *meigenVector
}
