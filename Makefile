SHELL=/bin/bash

### Config parameters.
ENABLE_CUDA=0

all: step10_cahp-rt
	### ==============================
	###  Build successfully completed!
	### ==============================

step1_prepare:
	### ==============================
	###  Preparing for build
	### ==============================
	mkdir -p build/bin
	mkdir -p build/share/kvsp
	cp -a share/* build/share/kvsp/

step2_kvsp: step1_prepare
	### ==============================
	###  Building kvsp
	### ==============================
	mkdir -p build/kvsp
	cd kvsp && \
		if [ ! -f go.mod ]; then \
			go mod init github.com/kvsp/kvsp && \
			go get github.com/BurntSushi/toml@latest && \
			go mod tidy; \
		fi && \
		go build -o ../build/kvsp/kvsp -ldflags "\
			-X main.kvspVersion=$$(git describe --tags --abbrev=0 || echo "unk") \
			-X main.kvspRevision=$$(git rev-parse --short HEAD || echo "unk") \
			-X main.iyokanRevision=$$(git -C ../Iyokan rev-parse --short HEAD || echo "unk") \
			-X main.cahpRubyRevision=$$(git -C ../cahp-ruby rev-parse --short HEAD || echo "unk") \
			-X main.cahpPearlRevision=$$(git -C ../cahp-pearl rev-parse --short HEAD || echo "unk") \
			-X main.cahpRtRevision=$$(git -C ../cahp-rt rev-parse --short HEAD || echo "unk") \
			-X main.cahpSimRevision=$$(git -C ../cahp-sim rev-parse --short HEAD || echo "unk") \
			-X main.llvmCahpRevision=$$(git -C ../llvm-cahp rev-parse --short HEAD || echo "unk") \
			-X main.yosysRevision=$$(git -C ../yosys rev-parse --short HEAD || echo "unk")"
	cp -a build/kvsp/kvsp build/bin/

step3_iyokan: step2_kvsp
	### ==============================
	###  Building Iyokan
	### ==============================
	mkdir -p build/Iyokan
	cd build/Iyokan && \
		cmake \
			-DCMAKE_BUILD_TYPE="Release" \
			-DIYOKAN_ENABLE_CUDA=$(ENABLE_CUDA) \
			../../Iyokan && \
		$(MAKE) iyokan iyokan-packet
	cp -a build/Iyokan/bin/iyokan build/bin/
	cp -a build/Iyokan/bin/iyokan-packet build/bin/

step4_cahp-sim: step3_iyokan
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

step5_yosys: step4_cahp-sim
	### ==============================
	###  Building Yosys
	### ==============================
	if [ ! -e "./build/yosys" ]; then ln -s ${PWD}/yosys build/yosys; fi
	cd build/yosys && $(MAKE)

step7_cahp-ruby: step5_yosys
	### ==============================
	###  Building cahp-ruby
	### ==============================
	cp -a cahp-ruby build/
	cd build/cahp-ruby && sbt run
	cd build/cahp-ruby && \
		../yosys/yosys build.ys
	cp build/cahp-ruby/vsp-core-ruby.json build/share/kvsp/ruby-core.json

step8_cahp-pearl: step7_cahp-ruby
	### ==============================
	###  Building cahp-pearl
	### ==============================
	cp -a cahp-pearl build/
	cd build/cahp-pearl && sbt run
	cd build/cahp-pearl && \
		../yosys/yosys build.ys
	cp build/cahp-pearl/vsp-core-pearl.json build/share/kvsp/pearl-core.json

step9_llvm-cahp: step8_cahp-pearl
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

step10_cahp-rt: step9_llvm-cahp
	### ==============================
	###  Building cahp-rt
	### ==============================
	cp -a cahp-rt build/
	cd build/cahp-rt && CC=../llvm-cahp/bin/clang $(MAKE)
	mkdir -p build/share/kvsp/cahp-rt
	cd build/cahp-rt && \
		cp -a crt0.o libc.a cahp.lds ../share/kvsp/cahp-rt/

