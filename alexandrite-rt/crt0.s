.section .text.start
.global _start
_start:
	li sp, 0x00010008
	lw sp, 0(sp)
	lw a0, 0(sp)
	addi a1, sp, 4

	call main

	li t0, 0x00010004
	li t1, 1
	sw t1, 0(t0)
1:
	j 1b
