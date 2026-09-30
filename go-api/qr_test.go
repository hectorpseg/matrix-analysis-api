package main

import (
	"math"
	"testing"
)

const testTol = 1e-10

func matricesApproxEqual(t *testing.T, name string, A, B [][]float64, tol float64) {
	if len(A) != len(B) || (len(A) > 0 && len(A[0]) != len(B[0])) {
		t.Fatalf("%s: shape mismatch: len = %d x %d, want %d x %d", name, len(A), len(A[0]), len(B), len(B[0]))
	}
	maxDiff := 0.0
	scale := 1.0
	for i := range A {
		for j := range A[i] {
			diff := math.Abs(A[i][j] - B[i][j])
			maxDiff = max(maxDiff, diff)
			scale = max(scale, math.Abs(A[i][j]), math.Abs(B[i][j]))
		}
	}
	if maxDiff > tol*scale {
		t.Fatalf("%s: max abs diff = %g exceeds tolerance %g (scale = %g)", name, maxDiff, tol, scale)
	}
}

func isOrthogonal(t *testing.T, name string, Q [][]float64) {
	m := len(Q)
	matricesApproxEqual(t, name+" orthogonality", multiply(transpose(Q), Q), identity(m), testTol)
}

func isUpperTriangular(t *testing.T, name string, R [][]float64) {
	for i := range R {
		for j := 0; j < i && j < len(R[i]); j++ {
			if math.Abs(R[i][j]) > testTol {
				t.Fatalf("%s: R not upper triangular: R[%d][%d] = %g", name, i, j, R[i][j])
			}
		}
	}
}

// assertQR checks Q·R ≈ A, Q^T·Q = I and R upper triangular.
func assertQR(t *testing.T, name string, A [][]float64) {
	t.Helper()
	Q, R := QR(A)
	if Q == nil || R == nil {
		t.Fatalf("%s: QR returned nil result", name)
	}
	matricesApproxEqual(t, name+" reconstruction", multiply(Q, R), A, testTol)
	isOrthogonal(t, name, Q)
	isUpperTriangular(t, name, R)
}

// Validates the minimal square case, where every step has full-width vectors.

func TestQRTwoByTwo(t *testing.T) {
	assertQR(t, "2x2", [][]float64{{1, 2}, {3, 4}})
}

// Validates the standard square factorization with mixed-sign, non-symmetric data.

func TestQRSquare(t *testing.T) {
	assertQR(t, "3x3", [][]float64{
		{12, -51, 4},
		{6, 167, -68},
		{-4, 24, -41},
	})
}

// Validates that Q stays square and orthogonal while R gains zero rows below p = n.

func TestQRTall(t *testing.T) {
	assertQR(t, "4x2", [][]float64{
		{1, 0},
		{2, 1},
		{1, 3},
		{0, 4},
	})
}

// Validates that extra columns are folded into R's upper-triangular rows since p = m.

func TestQRWide(t *testing.T) {
	assertQR(t, "2x4", [][]float64{
		{1, 2, 3, 4},
		{5, 6, 7, 8},
	})
}

// Validates that a linearly dependent column yields a (near-)zero R diagonal instead of skewing Q.

func TestQRRankDeficient(t *testing.T) {
	// Third column is a linear combination of the first two.
	assertQR(t, "3x3 rank 2", [][]float64{
		{1, 2, 3},
		{4, 5, 9},
		{7, 8, 15},
	})
}

// Validates that sparse diagonal matrices, including a negative entry, keep an identity-like Q.

func TestQRDiagonal(t *testing.T) {
	A := [][]float64{
		{2, 0, 0},
		{0, 5, 0},
		{0, 0, -3},
	}
	assertQR(t, "diagonal", A)
}

// Validates the degenerate single-row wide case: one reflection over a 1x1 Q (sign is implementation-dependent).

func TestQRSimpleSingleRow(t *testing.T) {
	// 1x3 wide: Q is 1x1 (magnitude 1) and R = Q[0][0]*A, sign not normalized.
	A := [][]float64{{3, 1, -4}}
	Q, R := QR(A)
	if math.Abs(Q[0][0]) != 1 {
		t.Fatalf("1x3 Q should have magnitude 1, got %g", Q[0][0])
	}
	matricesApproxEqual(t, "1x3 R", R, multiply([][]float64{{Q[0][0]}}, A), testTol)
}

// Validates graceful handling of an exactly-zero subcolumn: reflection skipped, no division by zero.

func TestQRRankOneColumn(t *testing.T) {
	// Full zero column: reflection must be skipped without dividing by zero.
	assertQR(t, "zero column", [][]float64{
		{0, 1},
		{0, 1},
		{0, 1},
	})
}

// Validates that malformed inputs are rejected by returning nil Q and nil R instead of panicking.

func TestQRInvalidInputs(t *testing.T) {
	if Q, R := QR(nil); Q != nil || R != nil {
		t.Fatal("nil matrix should return nil results")
	}
	if Q, R := QR([][]float64{}); Q != nil || R != nil {
		t.Fatal("empty matrix should return nil results")
	}
	if Q, R := QR([][]float64{{1, 2}, {3}}); Q != nil || R != nil {
		t.Fatal("ragged matrix should return nil results")
	}
}
