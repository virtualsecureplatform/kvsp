CC = clang
AR = ar
CFLAGS ?= -target riscv32-unknown-elf -march=rv32i -mabi=ilp32 -Oz -ffreestanding -fno-builtin -fno-unwind-tables -fno-asynchronous-unwind-tables -nostdlib

all: crt0.o libc.a

crt0.o: crt0.s
	$(CC) $(CFLAGS) -c $< -o $@

libc.a: runtime.o
	$(AR) rcs $@ $^

%.o: %.c
	$(CC) $(CFLAGS) -c $< -o $@

clean:
	rm -f *.o *.a

.PHONY: clean all
