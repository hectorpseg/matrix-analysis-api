const TOLERANCE = 1e-9;

export function validateMatrix(matrix) {
  if (!Array.isArray(matrix)) {
    throw new Error('matrix must be an array');
  }
  if (matrix.length === 0) {
    throw new Error('matrix must not be empty');
  }
  if (!Array.isArray(matrix[0]) || matrix[0].length === 0) {
    throw new Error('matrix rows must be non-empty arrays');
  }

  const expectedCols = matrix[0].length;

  for (let i = 0; i < matrix.length; i++) {
    const row = matrix[i];
    if (!Array.isArray(row)) {
      throw new Error('matrix rows must be arrays');
    }
    if (row.length !== expectedCols) {
      throw new Error('matrix rows must have consistent dimensions');
    }
    for (let j = 0; j < row.length; j++) {
      if (!Number.isFinite(row[j])) {
        throw new Error('matrix values must be finite numbers');
      }
    }
  }
}

export function computeGlobalStats(q, r) {
  validateMatrix(q);
  validateMatrix(r);

  let sum = 0;
  let max = q[0][0];
  let min = q[0][0];
  let count = 0;

  for (const matrix of [q, r]) {
    for (const row of matrix) {
      for (const value of row) {
        sum += value;
        count += 1;
        if (value > max) max = value;
        if (value < min) min = value;
      }
    }
  }

  return {
    max,
    min,
    average: sum / count,
    sum
  };
}

export function isDiagonal(matrix) {
  validateMatrix(matrix);

  const rows = matrix.length;
  const cols = matrix[0].length;
  if (rows !== cols) {
    return false;
  }

  for (let i = 0; i < rows; i++) {
    for (let j = 0; j < cols; j++) {
      if (i !== j && Math.abs(matrix[i][j]) > TOLERANCE) {
        return false;
      }
    }
  }
  return true;
}

export function computeStats(q, r) {
  validateMatrix(q);
  validateMatrix(r);

  return {
    global: computeGlobalStats(q, r),
    q: { isDiagonal: isDiagonal(q) },
    r: { isDiagonal: isDiagonal(r) }
  };
}
