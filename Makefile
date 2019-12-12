SHELL=/bin/bash
NCORES=$(shell grep cpu.cores /proc/cpuinfo | sort -u | sed 's/[^0-9]//g')

all: build/kvsp build/app build/tools build/llvm-cahp build/cahp-rt

build/kvsp: FORCE
	mkdir -p build/bin
	cp kvsp build/bin/

build/app: FORCE
	mkdir -p build/app
	cd app && go build -o ../build/app/app
	cp build/app/app build/bin/

build/tools: FORCE
	mkdir -p build/tools
	cd build/tools && \
		cmake \
			-DCMAKE_BUILD_TYPE="Release" \
			-DENABLE_FFTW=off \
			-DENABLE_NAYUKI_PORTABLE=off -DENABLE_NAYUKI_AVX=off \
			-DENABLE_SPQLIOS_AVX=on -DENABLE_SPQLIOS_FMA=off \
			../.. && \
		make -j $$(( $(NCORES) + 1 ))
	cp build/tools/bin/* build/bin/

build/llvm-cahp: FORCE
	mkdir -p build/llvm-cahp
	cd build/llvm-cahp && \
		cmake \
			-DCMAKE_BUILD_TYPE="Release" \
			-DLLVM_ENABLE_PROJECTS="lld;clang" \
			-DLLVM_TARGETS_TO_BUILD="" \
			-DLLVM_EXPERIMENTAL_TARGETS_TO_BUILD="CAHP" \
			../../llvm-cahp/llvm && \
		make -j $$(( $(NCORES) + 1 ))
	cp build/llvm-cahp/bin/* build/bin/

build/cahp-rt: build/llvm-cahp FORCE
	cp -r cahp-rt build/cahp-rt
	cd build/cahp-rt && CC=../llvm-cahp/bin/clang make
	mkdir -p build/cahp-sysroot
	cd build/cahp-rt && \
		cp crt0.o libc.a cahp.lds ../cahp-sysroot/

FORCE:

.PHONY: FORCE
