SHELL=/bin/bash

### Config parameters.
ENABLE_CUDA=0

all: prepare \
	 build/cahp-sim \
	 build/iyokan \
	 build/kvsp \
	 build/share/kvsp/emerald-core.json \
	 build/llvm-cahp \
	 build/cahp-rt

prepare:
	mkdir -p build/bin
	mkdir -p build/share/kvsp
	rsync -a --delete share/* build/share/kvsp/

build/kvsp:
	mkdir -p build/kvsp
	cd kvsp && go build -o ../build/kvsp/kvsp
	cp build/kvsp/kvsp build/bin/

build/iyokan:
	mkdir -p build/iyokan
	cd build/iyokan && \
		cmake \
			-DCMAKE_BUILD_TYPE="Release" \
			-DIYOKAN_ENABLE_CUDA=$(ENABLE_CUDA) \
			../../Iyokan && \
		$(MAKE) iyokan iyokan-packet
	cp build/iyokan/bin/iyokan build/bin/
	cp build/iyokan/bin/iyokan-packet build/bin/

build/cahp-sim:
	mkdir -p build/cahp-sim
	cd build/cahp-sim && \
		cmake \
			-DCMAKE_BUILD_TYPE="Release" \
			../../cahp-sim && \
		$(MAKE) cahp-sim
	cp build/cahp-sim/src/cahp-sim build/bin/

build/core:
	rsync -a --delete cahp-emerald/ build/core/
	cd build/core && sbt run

build/yosys:
	rsync -a --delete yosys build/
	cd build/yosys && $(MAKE) ABCREV=default

build/Iyokan-L1:
	cp -r Iyokan-L1 build/

build/core/vsp-core-no-ram-rom.json: build/core build/yosys
	cd build/core && \
		../yosys/yosys build-no-ram-rom.ys

build/share/kvsp/emerald-core.json: build/core/vsp-core-no-ram-rom.json build/Iyokan-L1
	dotnet run -p build/Iyokan-L1/ -c Release $< $@

build/llvm-cahp:
	mkdir -p build/llvm-cahp
	cd build/llvm-cahp && \
		cmake \
			-DCMAKE_BUILD_TYPE="Release" \
			-DLLVM_ENABLE_PROJECTS="lld;clang" \
			-DLLVM_TARGETS_TO_BUILD="" \
			-DLLVM_EXPERIMENTAL_TARGETS_TO_BUILD="CAHP" \
			../../llvm-cahp/llvm && \
		$(MAKE)
	cp build/llvm-cahp/bin/* build/bin/

build/cahp-rt: build/llvm-cahp
	cp -r cahp-rt build/
	cd build/cahp-rt && CC=../llvm-cahp/bin/clang $(MAKE)
	mkdir -p build/share/kvsp/cahp-rt
	cd build/cahp-rt && \
		cp crt0.o libc.a cahp.lds ../share/kvsp/cahp-rt/

.PHONY: all prepare
