SHELL=/bin/bash

### Get # of jobs (the options -j and --jobs)
### (thanks to: https://stackoverflow.com/questions/5303553/gnu-make-extracting-argument-to-j-within-makefile)
MAKE_PID := $(shell echo $$PPID)
NJOBS := $(shell ps T | sed -n 's/.*$(MAKE_PID).*$(MAKE).* \(-j\|--jobs\) *\([0-9][0-9]*\).*/\2/p')

### Config parameters.
ENABLE_CUDA=0

all: PHASE2

prepare: PHASE0
	mkdir -p build/bin
	mkdir -p build/share/kvsp
	rsync -a --delete share/* build/share/kvsp/

build/kvsp: PHASE0
	mkdir -p build/kvsp
	cd kvsp && go build -o ../build/kvsp/kvsp
	cp build/kvsp/kvsp build/bin/

build/iyokan: PHASE0
	mkdir -p build/iyokan
	cd build/iyokan && \
		cmake \
			-DCMAKE_BUILD_TYPE="Release" \
			-DIYOKAN_ENABLE_CUDA=$(ENABLE_CUDA) \
			../../Iyokan && \
		make iyokan iyokan-packet
	cp build/iyokan/bin/iyokan build/bin/
	cp build/iyokan/bin/iyokan-packet build/bin/

build/cahp-sim: PHASE0
	mkdir -p build/cahp-sim
	cd build/cahp-sim && \
		cmake \
			-DCMAKE_BUILD_TYPE="Release" \
			../../cahp-sim && \
		make cahp-sim
	cp build/cahp-sim/src/cahp-sim build/bin/

build/core: PHASE0
	rsync -a --delete cahp-emerald/ build/core/
	cd build/core && sbt run

build/yosys: PHASE0
	rsync -a --delete yosys build/
	cd build/yosys && $(MAKE) ABCREV=default

build/Iyokan-L1: PHASE0
	cp -r Iyokan-L1 build/
	cd build/Iyokan-L1 && \
		dotnet build

build/core/vsp-core-no-ram-rom.json: build/core build/yosys PHASE0
	cd build/core && \
		../yosys/yosys build-no-ram-rom.ys

build/share/kvsp/emerald-core.json: build/core/vsp-core-no-ram-rom.json build/Iyokan-L1 PHASE0
	dotnet run -p build/Iyokan-L1/ $< $@

build/llvm-cahp: PHASE1
	mkdir -p build/llvm-cahp
	cd build/llvm-cahp && \
		cmake \
			-DCMAKE_BUILD_TYPE="Release" \
			-DLLVM_ENABLE_PROJECTS="lld;clang" \
			-DLLVM_TARGETS_TO_BUILD="" \
			-DLLVM_EXPERIMENTAL_TARGETS_TO_BUILD="CAHP" \
			../../llvm-cahp/llvm && \
		make -j $(NJOBS)
	cp build/llvm-cahp/bin/* build/bin/

build/cahp-rt: build/llvm-cahp PHASE1
	cp -r cahp-rt build/
	cd build/cahp-rt && CC=../llvm-cahp/bin/clang make
	mkdir -p build/share/kvsp/cahp-rt
	cd build/cahp-rt && \
		cp crt0.o libc.a cahp.lds ../share/kvsp/cahp-rt/

PHASE0:

PHASE1: \
	prepare \
	build/cahp-sim \
	build/iyokan \
	build/kvsp \
	build/share/kvsp/emerald-core.json

PHASE2: \
	build/llvm-cahp \
	build/cahp-rt

.PHONY: prepare PHASE0 PHASE1 PHASE2
