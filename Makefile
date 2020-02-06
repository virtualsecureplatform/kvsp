SHELL=/bin/bash
NCORES=$(shell grep cpu.cores /proc/cpuinfo | sort -u | sed 's/[^0-9]//g')

### Config parameters.
ENABLE_CUDA=0

all: prepare \
	build/cahp-sim \
	build/iyokan \
	build/kvsp \
	build/share/kvsp/vsp-core.json \
	build/share/kvsp/vsp-core-wo-rom.json \
	build/share/kvsp/vsp-core-wo-ram-rom.json \
	build/llvm-cahp \
	build/cahp-rt

prepare: FORCE
	mkdir -p build/bin
	mkdir -p build/share/kvsp

build/kvsp: FORCE
	mkdir -p build/kvsp
	cd kvsp && go build -o ../build/kvsp/kvsp
	cp build/kvsp/kvsp build/bin/

build/iyokan: FORCE
	mkdir -p build/iyokan
	cd build/iyokan && \
		cmake \
			-DCMAKE_BUILD_TYPE="Release" \
			-DIYOKAN_ENABLE_CUDA=$(ENABLE_CUDA) \
			../../Iyokan && \
		make -j $$(( $(NCORES) + 1 )) iyokan kvsp-packet
	cp build/iyokan/bin/iyokan build/bin/
	cp build/iyokan/bin/kvsp-packet build/bin/

build/cahp-sim: FORCE
	mkdir -p build/cahp-sim
	cd build/cahp-sim && \
		cmake \
			-DCMAKE_BUILD_TYPE="Release" \
			../../cahp-sim && \
		make -j $$(( $(NCORES) + 1 )) cahp-sim
	cp build/cahp-sim/src/cahp-sim build/bin/

build/core: FORCE
	rsync -a --delete cahp-diamond/ build/core/
	cd build/core && sbt run

build/Iyokan-L1: FORCE
	cp -r Iyokan-L1 build/
	cd build/Iyokan-L1 && \
		dotnet build

build/core/vsp-core-no-ram-rom.json: build/core FORCE
	cd build/core && \
		yosys build-no-ram-rom.ys

build/core/vsp-core-no-rom.json: build/core FORCE
	cd build/core && \
		yosys build-no-rom.ys

build/share/kvsp/vsp-core.json: build/core/vsp-core-no-rom.json build/Iyokan-L1 FORCE
	cd build/Iyokan-L1 && \
		dotnet run $@ $< --with-rom

build/share/kvsp/vsp-core-wo-rom.json: build/core/vsp-core-no-rom.json build/Iyokan-L1 FORCE
	cd build/Iyokan-L1 && \
		dotnet run $@ $<

build/share/kvsp/vsp-core-wo-ram-rom.json: build/core/vsp-core-no-ram-rom.json build/Iyokan-L1 FORCE
	cd build/Iyokan-L1 && \
		dotnet run $@ $<

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
	cp -r cahp-rt build/
	cd build/cahp-rt && CC=../llvm-cahp/bin/clang make
	mkdir -p build/share/kvsp/cahp-rt
	cd build/cahp-rt && \
		cp crt0.o libc.a cahp.lds ../share/kvsp/cahp-rt/

FORCE:

.PHONY: FORCE prepare
