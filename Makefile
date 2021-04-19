SHELL=/bin/bash

### Config parameters.
ENABLE_CUDA=0

all:
	### ==============================
	###  Preparing for build
	### ==============================
	mkdir -p build/bin
	mkdir -p build/share/kvsp
	cp -a share/* build/share/kvsp/
	### ==============================
	###  Building kvsp
	### ==============================
	mkdir -p build/kvsp
	cd kvsp && \
		go build -o ../build/kvsp/kvsp -ldflags "\
			-X main.kvspVersion=$$(git describe --tags --abbrev=0 || echo "unk") \
			-X main.kvspRevision=$$(git rev-parse --short HEAD || echo "unk") \
			-X main.iyokanRevision=$$(git -C ../Iyokan rev-parse --short HEAD || echo "unk") \
			-X main.iyokanL1Revision=$$(git -C ../Iyokan-L1 rev-parse --short HEAD || echo "unk") \
			-X main.cahpRubyRevision=$$(git -C ../cahp-ruby rev-parse --short HEAD || echo "unk") \
			-X main.cahpPearlRevision=$$(git -C ../cahp-pearl rev-parse --short HEAD || echo "unk") \
			-X main.cahpRtRevision=$$(git -C ../cahp-rt rev-parse --short HEAD || echo "unk") \
			-X main.cahpSimRevision=$$(git -C ../cahp-sim rev-parse --short HEAD || echo "unk") \
			-X main.llvmCahpRevision=$$(git -C ../llvm-cahp rev-parse --short HEAD || echo "unk") \
			-X main.yosysRevision=$$(git -C ../yosys rev-parse --short HEAD || echo "unk")"
	cp -a build/kvsp/kvsp build/bin/
	### ==============================
	###  Building Iyokan
	### ==============================
	mkdir -p build/Iyokan
	cd build/Iyokan && \
		cmake \
			-DCMAKE_BUILD_TYPE="Release" \
			-DIYOKAN_ENABLE_CUDA=$(ENABLE_CUDA) \
			-DCMAKE_C_COMPILER=clang \
			-DCMAKE_CXX_COMPILER=clang++ \
			../../Iyokan && \
		$(MAKE) iyokan iyokan-packet
	cp -a build/Iyokan/bin/iyokan build/bin/
	cp -a build/Iyokan/bin/iyokan-packet build/bin/
	### ==============================
	###  Building cahp-sim
	### ==============================
	mkdir -p build/cahp-sim
	cd build/cahp-sim && \
		cmake \
			-DCMAKE_BUILD_TYPE="Release" \
			../../cahp-sim && \
		$(MAKE) cahp-sim
	cp -a build/cahp-sim/src/cahp-sim build/bin/
	### ==============================
	###  Building Yosys
	### ==============================
	cp -a yosys build/
	cd build/yosys && $(MAKE)
	### ==============================
	###  Building Iyokan-L1
	### ==============================
	cp -a Iyokan-L1 build/
	cd build/Iyokan-L1 && dotnet build
	### ==============================
	###  Building cahp-ruby
	### ==============================
	cp -a cahp-ruby build/
	cd build/cahp-ruby && sbt run
	cd build/cahp-ruby && \
		../yosys/yosys build.ys
	dotnet run -p build/Iyokan-L1/ -c Release build/cahp-ruby/vsp-core-ruby.json build/share/kvsp/ruby-core.json
	### ==============================
	###  Building cahp-pearl
	### ==============================
	cp -a cahp-pearl build/
	cd build/cahp-pearl && sbt run
	cd build/cahp-pearl && \
		../yosys/yosys build.ys
	dotnet run -p build/Iyokan-L1/ -c Release build/cahp-pearl/vsp-core-pearl.json build/share/kvsp/pearl-core.json
	### ==============================
	###  Building llvm-cahp
	### ==============================
	mkdir -p build/llvm-cahp
	cd build/llvm-cahp && \
		cmake \
			-DCMAKE_BUILD_TYPE="Release" \
			-DLLVM_ENABLE_PROJECTS="lld;clang" \
			-DLLVM_TARGETS_TO_BUILD="" \
			-DLLVM_EXPERIMENTAL_TARGETS_TO_BUILD="CAHP" \
			../../llvm-cahp/llvm && \
		$(MAKE)
	cp -a build/llvm-cahp/bin/* build/bin/
	### ==============================
	###  Building cahp-rt
	### ==============================
	cp -a cahp-rt build/
	cd build/cahp-rt && CC=../llvm-cahp/bin/clang $(MAKE)
	mkdir -p build/share/kvsp/cahp-rt
	cd build/cahp-rt && \
		cp -a crt0.o libc.a cahp.lds ../share/kvsp/cahp-rt/
	### ==============================
	###  Build successfully completed!
	### ==============================
