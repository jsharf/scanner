package points

import "github.com/gonum/matrix/mat64"
import "math"

type Plane struct {
	Center     mat64.Vector
	UnitNormal mat64.Vector
}

type Neighborhood struct {
	Center mat64.Vector
	R      float64
	mat64.Dense
}

const numberAnnuli = 5
const frobeniusNorm = 2

// n = normal
// x = point in space (finding distance between this and plane p)
// c = point on plane ("center")
// d = distance to plane.
// n*(x - c) = 0
// ((a - d*n) - c) * n = 0
// a*n - d - c*n = 0
// d = a*n - n*c
func (p *Plane) DistanceToPoint(a mat64.Vector) float64 {
	return mat64.Dot(&a, &p.UnitNormal) - mat64.Dot(&p.UnitNormal, &p.Center)
}

func GetNeighborhood(point mat64.Vector, points mat64.Dense, r float64) Neighborhood {
	_, c := points.Dims()
	neighborhood := &Neighborhood{
		Center: point,
		R:      r,
	}
	for j := 0; j < c; j++ {
		column := points.ColView(j)
		diff := &mat64.Vector{}
		diff.SubVec(column, &point)
		distance := mat64.Norm(diff, frobeniusNorm)
		if distance <= r {
			neighborhood.Augment(neighborhood, column)
		}
	}
	return *neighborhood
}

func (n *Neighborhood) Plane() Plane {
	unitNormal := &mat64.Vector{}
	normal := n.Normal()
	unitNormal.ScaleVec(1/mat64.Norm(&normal, frobeniusNorm), unitNormal)
	center := mat64.Vector{}
	center.AddScaledVec(&n.Center, n.R, unitNormal)
	return Plane{
		UnitNormal: *unitNormal,
		Center:     center,
	}
}

func (n *Neighborhood) RadialHistogram() map[int]int {
	histogram := make(map[int]int)
	projectionPlane := n.Plane()
	r, c := n.Dims()
	for j := 0; j < c; j++ {
		point := n.ColView(j)
		dist := projectionPlane.DistanceToPoint(*point)
		projectedPoint := mat64.Vector{}
		projectedPoint.AddScaledVec(point, dist, &projectionPlane.UnitNormal)
		displacement := mat64.Vector{}
		displacement.SubVec(&n.Center, &projectedPoint)
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
	if v.Len() == 0 {
		return 0
	}
	return sum(v) / v.Len()
}

func columnCovariance(a, b int, m mat64.Matrix) float64 {
	r, c := m.Dims()

	sum := float64(0)

	aAvg := average(mat64.Col(nil, a, m))
	bAvg := average(mat64.Col(nil, b, m))

	for i := 0; i < r; i++ {
		sum += (m.At(i, a) - aAvg) * (m.At(i, b) - bBvg)
	}

	return sum / r
}

func Covariance(m mat64.Dense) mat64.Matrix {
	r, c := m.Dims()
	covMatrix := mat64.NewDense(r, c, nil)
	for i := 0; i < r; i++ {
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
func (n *Neighborhood) Normal() mat64.Vector {
	covMatrix := Covariance(n)
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
	return *meigenVector
}
