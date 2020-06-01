SHELL=/bin/bash

### Config parameters.
ENABLE_CUDA=0

all: prepare \
	 build/cahp-sim \
	 build/Iyokan \
	 build/kvsp \
	 build/share/kvsp/diamond-core.json \
	 build/share/kvsp/emerald-core.json \
	 build/share/kvsp/ruby-core.json \
	 build/share/kvsp/pearl-core.json \
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
			-X main.cahpRubyRevision=$$(git -C ../cahp-ruby rev-parse --short HEAD) \
			-X main.cahpPearlRevision=$$(git -C ../cahp-pearl rev-parse --short HEAD) \
			-X main.cahpRtRevision=$$(git -C ../cahp-rt rev-parse --short HEAD) \
			-X main.cahpSimRevision=$$(git -C ../cahp-sim rev-parse --short HEAD) \
			-X main.llvmCahpRevision=$$(git -C ../llvm-cahp rev-parse --short HEAD) \
			-X main.yosysRevision=$$(git -C ../yosys rev-parse --short HEAD)"
	cp build/kvsp/kvsp build/bin/

build/Iyokan:
	mkdir -p build/Iyokan
	cd build/Iyokan && \
		cmake \
			-DCMAKE_BUILD_TYPE="Release" \
			-DIYOKAN_ENABLE_CUDA=$(ENABLE_CUDA) \
			../../Iyokan && \
		$(MAKE) iyokan iyokan-packet
	cp build/Iyokan/bin/iyokan build/bin/
	cp build/Iyokan/bin/iyokan-packet build/bin/

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

# NOTE: build/cahp-diamond is "fake" dependency;
# parallel `sbt run` may cause some problems about file lock.
# build/cahp-ruby and build/cahp-pearl are also similar.
build/cahp-emerald: build/cahp-diamond
	rsync -a --delete cahp-emerald/ build/cahp-emerald/
	cd build/cahp-emerald && sbt run

build/cahp-ruby: build/cahp-emerald
	rsync -a --delete cahp-ruby/ build/cahp-ruby/
	cd build/cahp-ruby && sbt run

build/cahp-pearl: build/cahp-ruby
	rsync -a --delete cahp-pearl/ build/cahp-pearl/
	cd build/cahp-pearl && sbt run

build/yosys:
	rsync -a --delete yosys build/
	cd build/yosys && $(MAKE)

build/Iyokan-L1:
	cp -r Iyokan-L1 build/

build/cahp-diamond/vsp-core-no-ram-rom.json: build/cahp-diamond build/yosys
	cd build/cahp-diamond && \
		../yosys/yosys build-no-ram-rom.ys

# NOTE: build/cahp-diamond/vsp-core-no-ram-rom.json is "fake" dependency;
# Without this the builds for processors will run in parallel
# to consume too much memory.
build/cahp-emerald/vsp-core-no-ram-rom.json: build/cahp-emerald build/yosys build/cahp-diamond/vsp-core-no-ram-rom.json
	cd build/cahp-emerald && \
		../yosys/yosys build-no-ram-rom.ys

# NOTE: build/cahp-emerald/vsp-core-no-ram-rom.json is "fake" dependency;
# Without this the builds for processors will run in parallel
# to consume too much memory.
build/cahp-ruby/vsp-core-ruby.json: build/cahp-ruby build/yosys build/cahp-emerald/vsp-core-no-ram-rom.json
	cd build/cahp-ruby && \
		../yosys/yosys build.ys

# NOTE: build/cahp-ruby/vsp-core-ruby.json is "fake" dependency;
# Without this the builds for processors will run in parallel
# to consume too much memory.
build/cahp-pearl/vsp-core-pearl.json: build/cahp-pearl build/yosys build/cahp-ruby/vsp-core-ruby.json
	cd build/cahp-pearl && \
		../yosys/yosys build.ys

build/share/kvsp/diamond-core.json: build/cahp-diamond/vsp-core-no-ram-rom.json build/Iyokan-L1
	dotnet run -p build/Iyokan-L1/ -c Release $< $@

build/share/kvsp/emerald-core.json: build/cahp-emerald/vsp-core-no-ram-rom.json build/Iyokan-L1
	dotnet run -p build/Iyokan-L1/ -c Release $< $@

build/share/kvsp/ruby-core.json: build/cahp-ruby/vsp-core-ruby.json build/Iyokan-L1
	dotnet run -p build/Iyokan-L1/ -c Release $< $@

build/share/kvsp/pearl-core.json: build/cahp-pearl/vsp-core-pearl.json build/Iyokan-L1
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
	rsync -a build/llvm-cahp/bin/ build/bin/

build/cahp-rt: build/llvm-cahp
	cp -r cahp-rt build/
	cd build/cahp-rt && CC=../llvm-cahp/bin/clang $(MAKE)
	mkdir -p build/share/kvsp/cahp-rt
	cd build/cahp-rt && \
		cp crt0.o libc.a cahp.lds ../share/kvsp/cahp-rt/

.PHONY: all prepare
