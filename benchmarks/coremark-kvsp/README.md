# CoreMark matrix-kernel comparison

This port reproducibly compares all four KVSP CPUs using the
unmodified `matrix_mul_matrix` implementation from the pinned EEMBC CoreMark
submodule. The small driver initializes two 8-by-8 matrices from a run-time
seed and returns one output element. The runner rejects results that do not
match, so an incorrect execution cannot be mistaken for a fast one.

Build and run the plain-mode comparison with:

```sh
git submodule update --init benchmarks/coremark
make coremark-matrix
make coremark-matrix-run
```

The run takes tens of thousands of emulated CPU cycles. Override `BACKEND`,
`KVSP`, or `OUT_DIR` when needed. The reference result for seed 5 is signed
`-15816` (unsigned `49720` on 16-bit CAHP and `4294951480` on 32-bit RISC-V).

Reference results for the pinned toolchains are:

| CPU | Cycles | Text bytes |
| --- | ---: | ---: |
| Ruby | 177,409 | 572 |
| Pearl | 148,851 | 572 |
| Alexandrite | 35,442 | 1,020 |
| Chrysoberyl | 30,690 | 852 |

The cycle count is architectural and does not depend on host speed. The
plain-mode wall-clock time does depend on the evaluator build and machine.

This is an isolated CoreMark kernel comparison, **not** a reportable CoreMark
score. A standard CoreMark validation requires a 2,000-byte data set, which
does not fit the current 1 KiB KVSP RAM once runtime state is included.

The port also serves as a regression for CAHP widened subtraction, 32-bit
multiplication support, and calls between separately compiled translation
units.
