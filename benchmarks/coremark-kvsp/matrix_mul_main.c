#include "coremark.h"

#define MATRIX_ORDER 8

static MATDAT matrix_a[MATRIX_ORDER * MATRIX_ORDER];
static MATDAT matrix_b[MATRIX_ORDER * MATRIX_ORDER];
static MATRES matrix_c[MATRIX_ORDER * MATRIX_ORDER];

void matrix_mul_matrix(ee_u32 N, MATRES *C, MATDAT *A, MATDAT *B);

int main(int argc, char **argv) {
  MATDAT seed = (MATDAT)(argv[1][0] - '0');
  ee_u32 i;

  for (i = 0; i < MATRIX_ORDER * MATRIX_ORDER; ++i) {
    matrix_a[i] = (MATDAT)(seed + i);
    matrix_b[i] = (MATDAT)(seed - i);
  }
  matrix_mul_matrix(MATRIX_ORDER, matrix_c, matrix_a, matrix_b);
  return (int)matrix_c[MATRIX_ORDER * MATRIX_ORDER - 1];
}
