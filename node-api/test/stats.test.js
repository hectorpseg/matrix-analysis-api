import test from 'node:test';
import assert from 'node:assert/strict';
import {
  validateMatrix,
  computeGlobalStats,
  isDiagonal,
  computeStats
} from '../src/stats.js';

test('validateMatrix accepts a valid square matrix', () => {
  assert.doesNotThrow(() => validateMatrix([[1, 2], [3, 4]]));
});

test('validateMatrix accepts a valid rectangular matrix', () => {
  assert.doesNotThrow(() => validateMatrix([[1, 2, 3], [4, 5, 6]]));
});

test('validateMatrix rejects non-array', () => {
  assert.throws(() => validateMatrix('not a matrix'), /matrix must be an array/);
});

test('validateMatrix rejects empty matrix', () => {
  assert.throws(() => validateMatrix([]), /matrix must not be empty/);
});

test('validateMatrix rejects empty rows', () => {
  assert.throws(() => validateMatrix([[]]), /matrix rows must be non-empty arrays/);
});

test('validateMatrix rejects ragged rows', () => {
  assert.throws(() => validateMatrix([[1, 2], [3]]), /consistent dimensions/);
});

test('validateMatrix rejects non-numeric values', () => {
  assert.throws(() => validateMatrix([[1, 'x'], [3, 4]]), /finite numbers/);
});

test('validateMatrix rejects NaN', () => {
  assert.throws(() => validateMatrix([[1, NaN], [3, 4]]), /finite numbers/);
});

test('validateMatrix rejects Infinity', () => {
  assert.throws(() => validateMatrix([[1, Infinity], [3, 4]]), /finite numbers/);
});

test('computeGlobalStats returns correct stats for two matrices', () => {
  const q = [[1, 2], [3, 4]];
  const r = [[5, 6], [7, 8]];
  const stats = computeGlobalStats(q, r);
  assert.deepEqual(stats, {
    max: 8,
    min: 1,
    average: 4.5,
    sum: 36
  });
});

test('computeGlobalStats works with rectangular matrices', () => {
  const q = [[1, 2, 3]];
  const r = [[4], [5]];
  const stats = computeGlobalStats(q, r);
  assert.deepEqual(stats, {
    max: 5,
    min: 1,
    average: 3,
    sum: 15
  });
});

test('computeGlobalStats rejects invalid matrices', () => {
  assert.throws(() => computeGlobalStats([], [[1]]), /matrix must not be empty/);
  assert.throws(() => computeGlobalStats([[1]], 'x'), /matrix must be an array/);
});

test('computeGlobalStats rejects missing arguments', () => {
  assert.throws(() => computeGlobalStats([[1]]), /matrix must be an array/);
});

test('isDiagonal returns true for diagonal matrix', () => {
  assert.equal(isDiagonal([[1, 0], [0, 2]]), true);
});

test('isDiagonal returns false for non-square matrix', () => {
  assert.equal(isDiagonal([[1, 0, 0], [0, 2, 0]]), false);
});

test('isDiagonal returns false for non-diagonal matrix', () => {
  assert.equal(isDiagonal([[1, 2], [0, 3]]), false);
});

test('isDiagonal treats small off-diagonal values as zero', () => {
  assert.equal(isDiagonal([[1, 1e-9], [1e-9, 2]]), true);
});

test('isDiagonal rejects values just above tolerance', () => {
  assert.equal(isDiagonal([[1, 1.1e-9], [0, 2]]), false);
});

test('isDiagonal rejects invalid matrix', () => {
  assert.throws(() => isDiagonal('x'), /matrix must be an array/);
});

test('computeStats returns exact expected shape', () => {
  const q = [[1, 0], [0, 2]];
  const r = [[3, 4], [0, 5]];
  const result = computeStats(q, r);
  assert.deepEqual(result, {
    global: {
      max: 5,
      min: 0,
      average: 1.875,
      sum: 15
    },
    q: { isDiagonal: true },
    r: { isDiagonal: false }
  });
});

test('computeStats rejects missing q', () => {
  assert.throws(() => computeStats(undefined, [[1]]), /matrix must be an array/);
});

test('computeStats rejects missing r', () => {
  assert.throws(() => computeStats([[1]]), /matrix must be an array/);
});
