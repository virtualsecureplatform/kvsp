SHELL=/bin/bash
NCORES=$(shell grep cpu.cores /proc/cpuinfo | sort -u | sed 's/[^0-9]//g')

cpu: common build/tools

gpu: common build/tools_gpu

common: prepare build/kvsp build/share/kvsp/vsp-core.json build/llvm-cahp build/cahp-rt

prepare: FORCE
	mkdir -p build/bin
	mkdir -p build/share/kvsp

build/kvsp: FORCE
	mkdir -p build/kvsp
	cd kvsp && go build -o ../build/kvsp/kvsp
	cp build/kvsp/kvsp build/bin/

build/tools: FORCE
	mkdir -p build/tools
	cd build/tools && \
		cmake \
			-DCMAKE_BUILD_TYPE="Release" \
			-DENABLE_FFTW=off \
			-DENABLE_NAYUKI_PORTABLE=off -DENABLE_NAYUKI_AVX=off \
			-DENABLE_SPQLIOS_AVX=on -DENABLE_SPQLIOS_FMA=off \
			-DKVSP_ENABLE_CUDA=off \
			../.. && \
		make -j $$(( $(NCORES) + 1 ))
	cp build/tools/bin/* build/bin/

build/tools_gpu: FORCE
	mkdir -p build/tools_gpu
	cd build/tools_gpu && \
		cmake \
			-DCMAKE_BUILD_TYPE="Release" \
			-DENABLE_FFTW=off \
			-DENABLE_NAYUKI_PORTABLE=off -DENABLE_NAYUKI_AVX=off \
			-DENABLE_SPQLIOS_AVX=on -DENABLE_SPQLIOS_FMA=off \
			-DKVSP_ENABLE_CUDA=on \
			../.. && \
		make -j $$(( $(NCORES) + 1 ))
	cp build/tools_gpu/bin/* build/bin/

build/core: FORCE
	rsync -a --delete cahp-diamond/ build/core/
	cd build/core && sbt run

build/share/kvsp/vsp-core.json: build/core FORCE
	cp -r Iyokan-L1 build/
	cp build/core/VSPCore.v build/Iyokan-L1/
	cd build/Iyokan-L1 && \
		yosys build.ys && \
		dotnet run vsp-core.json vsp-core-converted.json
	cp build/Iyokan-L1/vsp-core-converted.json build/share/kvsp/vsp-core.json

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
