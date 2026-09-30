package main

import (
	"math"
)

// float64 machine epsilon: largest representable gap between 1 and the next float64.
const eps = 2.220446049250313e-16

// QR returns the complete QR decomposition of A:
//
//	Q is m×m orthogonal, R is m×n upper triangular, A ≈ Q·R.
//
// Invalid inputs (nil slices, empty matrix, ragged rows) return nil Q and nil R.
func QR(A [][]float64) (Q, R [][]float64) {
	m := len(A)
	if m == 0 {
		return nil, nil
	}
	n := len(A[0])
	for i := range A {
		if len(A[i]) != n || A[i] == nil {
			return nil, nil
		}
	}
	if n == 0 {
		return nil, nil
	}

	R = cloneMatrix(A)
	Q = identity(m)
	scale := maxAbsEntry(A)
	tol := eps * max(1.0, scale)

	p := min(m, n)
	for k := 0; k < p; k++ {
		v, tau := householderVector(col(R, k), tol)
		if tau == 0 {
			continue
		}
		applyHouseholderToR(R, v, tau, k)
		applyHouseholderToQ(Q, v, tau, k)
	}
	// Zero out the lower part of R that stores the Householder vectors during
	// the in-place work; the reflection math never produces exact zeros there.
	for k := 0; k < p; k++ {
		for i := k + 1; i < m; i++ {
			R[i][k] = 0
		}
	}
	return Q, R
}

// householderVector builds the reflection H = I - tau*v*v^T that maps x to
// (-sign(x0)*||x||)*e1. Its sign convention avoids cancellation because v[0]
// adds two numbers of the same sign.
func householderVector(x []float64, tol float64) (v []float64, tau float64) {
	n := len(x)
	v = make([]float64, n)
	norm := norm2(x)
	if norm <= tol {
		return v, 0
	}
	s := 1.0
	if x[0] < 0 {
		s = -1.0
	}
	v[0] = x[0] + s*norm
	copy(v[1:], x[1:])
	vnorm2 := dot(v, v)
	if vnorm2 == 0 {
		return v, 0
	}
	tau = 2 / vnorm2
	return v, tau
}

// dot returns the inner product of two vectors of equal length.
func dot(a, b []float64) float64 {
	s := 0.0
	for i := range a {
		s += a[i] * b[i]
	}
	return s
}

// applyHouseholderToR applies H = I - tau*v*v^T to the trailing block of R
// starting at column col, zeroing the subdiagonal entries in that column.
func applyHouseholderToR(R [][]float64, v []float64, tau float64, col int) {
	m := len(R)
	w := make([]float64, len(R[col])-col)
	for j := col; j < len(R[col]); j++ {
		s := 0.0
		for i := col; i < m; i++ {
			s += v[i-col] * R[i][j]
		}
		w[j-col] = s
	}
	for i := col; i < m; i++ {
		for j := col; j < len(R[i]); j++ {
			R[i][j] -= tau * v[i-col] * w[j-col]
		}
	}
}

// applyHouseholderToQ updates Q = Q*H by subtracting tau*(Q·v)*v^T from the
// columns at index col onwards.
func applyHouseholderToQ(Q [][]float64, v []float64, tau float64, col int) {
	m := len(Q)
	w := make([]float64, m)
	for i := 0; i < m; i++ {
		s := 0.0
		for j := col; j < m; j++ {
			s += Q[i][j] * v[j-col]
		}
		w[i] = s
	}
	for i := 0; i < m; i++ {
		for j := col; j < m; j++ {
			Q[i][j] -= tau * w[i] * v[j-col]
		}
	}
}

func cloneMatrix(A [][]float64) [][]float64 {
	B := make([][]float64, len(A))
	for i := range A {
		B[i] = append([]float64(nil), A[i]...)
	}
	return B
}

func zeros(m, n int) [][]float64 {
	R := make([][]float64, m)
	for i := range R {
		R[i] = make([]float64, n)
	}
	return R
}

func identity(m int) [][]float64 {
	Q := zeros(m, m)
	for i := 0; i < m; i++ {
		Q[i][i] = 1
	}
	return Q
}

func transpose(A [][]float64) [][]float64 {
	if len(A) == 0 {
		return nil
	}
	n := len(A[0])
	T := make([][]float64, n)
	for i := 0; i < n; i++ {
		T[i] = make([]float64, len(A))
		for j := 0; j < len(A); j++ {
			T[i][j] = A[j][i]
		}
	}
	return T
}

func multiply(A, B [][]float64) [][]float64 {
	m := len(A)
	n := len(B[0])
	k := len(B)
	C := zeros(m, n)
	for i := 0; i < m; i++ {
		for l := 0; l < k; l++ {
			if A[i][l] == 0 {
				continue
			}
			for j := 0; j < n; j++ {
				C[i][j] += A[i][l] * B[l][j]
			}
		}
	}
	return C
}

func norm2(x []float64) float64 {
	s := 0.0
	for _, xi := range x {
		s += xi * xi
	}
	return math.Sqrt(s)
}

// maxAbsEntry returns the largest absolute value in A.
func maxAbsEntry(A [][]float64) float64 {
	s := 0.0
	for _, row := range A {
		for _, x := range row {
			s = max(s, math.Abs(x))
		}
	}
	return s
}

// col returns the subcolumn of A starting at row start.
func col(A [][]float64, start int) []float64 {
	if start >= len(A) {
		return nil
	}
	x := make([]float64, len(A)-start)
	for i := start; i < len(A); i++ {
		x[i-start] = A[i][start]
	}
	return x
}
