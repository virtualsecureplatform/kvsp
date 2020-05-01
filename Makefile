SHELL=/bin/bash

### Config parameters.
ENABLE_CUDA=0

all: prepare \
	 build/cahp-sim \
	 build/iyokan \
	 build/kvsp \
	 build/share/kvsp/diamond-core.json \
	 build/share/kvsp/emerald-core.json \
	 build/llvm-cahp \
	 build/cahp-rt

prepare:
	mkdir -p build/bin
	mkdir -p build/share/kvsp
	rsync -a --delete share/* build/share/kvsp/

build/kvsp:
	mkdir -p build/kvsp
	cd kvsp && \
		go build -o ../build/kvsp/kvsp -ldflags "\
			-X main.kvspVersion=$$(git describe --tags --abbrev=0) \
			-X main.kvspRevision=$$(git rev-parse --short HEAD) \
			-X main.iyokanRevision=$$(git -C ../Iyokan rev-parse --short HEAD) \
			-X main.iyokanL1Revision=$$(git -C ../Iyokan-L1 rev-parse --short HEAD) \
			-X main.cahpDiamondRevision=$$(git -C ../cahp-diamond rev-parse --short HEAD) \
			-X main.cahpEmeraldRevision=$$(git -C ../cahp-emerald rev-parse --short HEAD) \
			-X main.cahpRtRevision=$$(git -C ../cahp-rt rev-parse --short HEAD) \
			-X main.cahpSimRevision=$$(git -C ../cahp-sim rev-parse --short HEAD) \
			-X main.llvmCahpRevision=$$(git -C ../llvm-cahp rev-parse --short HEAD) \
			-X main.yosysRevision=$$(git -C ../yosys rev-parse --short HEAD)"
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

build/cahp-diamond:
	rsync -a --delete cahp-diamond/ build/cahp-diamond/
	cd build/cahp-diamond && sbt run

build/cahp-emerald:
	rsync -a --delete cahp-emerald/ build/cahp-emerald/
	cd build/cahp-emerald && sbt run

build/yosys:
	rsync -a --delete yosys build/
	cd build/yosys && $(MAKE)

build/Iyokan-L1:
	cp -r Iyokan-L1 build/

build/cahp-diamond/vsp-core-no-ram-rom.json: build/cahp-diamond build/yosys
	cd build/cahp-diamond && \
		../yosys/yosys build-no-ram-rom.ys

build/cahp-emerald/vsp-core-no-ram-rom.json: build/cahp-emerald build/yosys
	cd build/cahp-emerald && \
		../yosys/yosys build-no-ram-rom.ys

build/share/kvsp/diamond-core.json: build/cahp-diamond/vsp-core-no-ram-rom.json build/Iyokan-L1
	dotnet run -p build/Iyokan-L1/ -c Release $< $@

build/share/kvsp/emerald-core.json: build/cahp-emerald/vsp-core-no-ram-rom.json build/Iyokan-L1
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
