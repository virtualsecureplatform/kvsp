SHELL=/bin/bash

### Config parameters.
ENABLE_CUDA=0
BUILDDIR := build
# Source tree that provides Iyokan's standard `iyokan` and `iyokan-packet`
# targets.
IYOKAN_SOURCE ?= Iyokan
IYOKAN_SOURCE_ABS := $(abspath $(IYOKAN_SOURCE))
# Tangor is built independently, so both evaluator backends can be installed
# in one KVSP tree and selected with `kvsp --backend`.
TANGOR_SOURCE ?= Tangor
TANGOR_SOURCE_ABS := $(abspath $(TANGOR_SOURCE))
# Extra arguments passed to Iyokan's CMake configure step.
IYOKAN_CMAKE_ARGS ?=
# Extra arguments passed to Tangor's CMake configure step. With ENABLE_CUDA=1,
# cuFHEpp CUDA workers join TFHEpp CPU workers in Tangor's StarPU gate graph.
TANGOR_CMAKE_ARGS ?=

.PHONY: all prepare kvsp iyokan iyokan-avx2 iyokan-avx512 tangor tangor-avx2 tangor-avx512 cahp-sim yosys cahp-ruby cahp-pearl alexandrite llvm-cahp cahp-rt alexandrite-rt clean

all: kvsp iyokan-avx2 iyokan-avx512 tangor-avx2 tangor-avx512 cahp-sim cahp-ruby cahp-pearl alexandrite cahp-rt alexandrite-rt
	@echo "Build successfully completed!"

prepare:
	@echo "Preparing for build..."
	mkdir -p $(BUILDDIR)/bin
	mkdir -p $(BUILDDIR)/share/kvsp
	cp -a share/* $(BUILDDIR)/share/kvsp/

kvsp: prepare
	@echo "Building kvsp..."
	mkdir -p $(BUILDDIR)/kvsp
	cd kvsp && \
		if [ ! -f go.mod ]; then \
			go mod init github.com/kvsp/kvsp && \
			go get github.com/BurntSushi/toml@latest && \
			go mod tidy; \
		fi && \
		go build -o ../$(BUILDDIR)/kvsp/kvsp -ldflags "\
			-X main.kvspVersion=$$(git describe --tags --abbrev=0 || echo "unk") \
			-X main.kvspRevision=$$(git rev-parse --short HEAD || echo "unk") \
			-X main.iyokanRevision=$$(git -C "$(IYOKAN_SOURCE_ABS)" rev-parse --short HEAD || echo "unk") \
			-X main.tangorRevision=$$(git -C "$(TANGOR_SOURCE_ABS)" rev-parse --short HEAD || echo "unk") \
			-X main.alexandriteRevision=$$(git -C ../Alexandrite rev-parse --short HEAD || echo "unk") \
			-X main.alexandriteRtRevision=$$(git rev-parse --short HEAD || echo "unk") \
			-X main.cahpRubyRevision=$$(git -C ../cahp-ruby rev-parse --short HEAD || echo "unk") \
			-X main.cahpPearlRevision=$$(git -C ../cahp-pearl rev-parse --short HEAD || echo "unk") \
			-X main.cahpRtRevision=$$(git -C ../cahp-rt rev-parse --short HEAD || echo "unk") \
			-X main.cahpSimRevision=$$(git -C ../cahp-sim rev-parse --short HEAD || echo "unk") \
			-X main.llvmCahpRevision=$$(git -C ../llvm-cahp rev-parse --short HEAD || echo "unk") \
			-X main.yosysRevision=$$(git -C ../yosys rev-parse --short HEAD || echo "unk")"
	cp -a $(BUILDDIR)/kvsp/kvsp $(BUILDDIR)/bin/

iyokan: iyokan-avx512

iyokan-avx2: prepare
	@echo "Building Iyokan (AVX2, -march=x86-64-v3, USE_AVX512=OFF)..."
	mkdir -p $(BUILDDIR)/Iyokan-avx2
	cd $(BUILDDIR)/Iyokan-avx2 && \
		if [ ! -f CMakeCache.txt ]; then cmake \
			-DCMAKE_BUILD_TYPE="Release" \
			-DIYOKAN_ENABLE_CUDA=$(ENABLE_CUDA) \
			-DIYOKAN_MARCH=x86-64-v3 \
			-DUSE_AVX512=OFF \
			$(IYOKAN_CMAKE_ARGS) \
			"$(IYOKAN_SOURCE_ABS)"; fi && \
		$(MAKE) iyokan iyokan-packet
	cp -a $(BUILDDIR)/Iyokan-avx2/bin/iyokan $(BUILDDIR)/bin/iyokan-avx2
	cp -a $(BUILDDIR)/Iyokan-avx2/bin/iyokan-packet $(BUILDDIR)/bin/iyokan-packet-avx2

iyokan-avx512: prepare
	@echo "Building Iyokan (AVX512, -march=native, USE_AVX512=ON)..."
	mkdir -p $(BUILDDIR)/Iyokan-avx512
	cd $(BUILDDIR)/Iyokan-avx512 && \
		if [ ! -f CMakeCache.txt ]; then cmake \
			-DCMAKE_BUILD_TYPE="Release" \
			-DIYOKAN_ENABLE_CUDA=$(ENABLE_CUDA) \
			-DUSE_AVX512=ON \
			$(IYOKAN_CMAKE_ARGS) \
			"$(IYOKAN_SOURCE_ABS)"; fi && \
		$(MAKE) iyokan iyokan-packet
	cp -a $(BUILDDIR)/Iyokan-avx512/bin/iyokan $(BUILDDIR)/bin/iyokan
	cp -a $(BUILDDIR)/Iyokan-avx512/bin/iyokan-packet $(BUILDDIR)/bin/iyokan-packet

tangor: tangor-avx512

tangor-avx2: prepare
	@echo "Building Tangor (AVX2, -march=x86-64-v3, USE_AVX512=OFF)..."
	mkdir -p $(BUILDDIR)/Tangor-avx2
	cd $(BUILDDIR)/Tangor-avx2 && \
		cmake \
			-DCMAKE_BUILD_TYPE="Release" \
			-DIYOKAN_ENABLE_CUDA=$(ENABLE_CUDA) \
			-DIYOKAN_MARCH=x86-64-v3 \
			-DUSE_AVX512=OFF \
			-DTANGOR_KVSP_STARPU_GATE_OFFLOAD=ON \
			-DTANGOR_USE_BUNDLED_STARPU=ON \
			$(TANGOR_CMAKE_ARGS) \
			"$(TANGOR_SOURCE_ABS)" && \
		$(MAKE) iyokan iyokan-packet
	cp -a $(BUILDDIR)/Tangor-avx2/bin/iyokan $(BUILDDIR)/bin/tangor-iyokan-avx2
	cp -a $(BUILDDIR)/Tangor-avx2/bin/iyokan-packet $(BUILDDIR)/bin/tangor-iyokan-packet-avx2

tangor-avx512: prepare
	@echo "Building Tangor (AVX512, -march=native, USE_AVX512=ON)..."
	mkdir -p $(BUILDDIR)/Tangor-avx512
	cd $(BUILDDIR)/Tangor-avx512 && \
		cmake \
			-DCMAKE_BUILD_TYPE="Release" \
			-DIYOKAN_ENABLE_CUDA=$(ENABLE_CUDA) \
			-DUSE_AVX512=ON \
			-DTANGOR_KVSP_STARPU_GATE_OFFLOAD=ON \
			-DTANGOR_USE_BUNDLED_STARPU=ON \
			$(TANGOR_CMAKE_ARGS) \
			"$(TANGOR_SOURCE_ABS)" && \
		$(MAKE) iyokan iyokan-packet
	cp -a $(BUILDDIR)/Tangor-avx512/bin/iyokan $(BUILDDIR)/bin/tangor-iyokan
	cp -a $(BUILDDIR)/Tangor-avx512/bin/iyokan-packet $(BUILDDIR)/bin/tangor-iyokan-packet

cahp-sim: prepare
	@echo "Building cahp-sim..."
	mkdir -p $(BUILDDIR)/cahp-sim
	cd $(BUILDDIR)/cahp-sim && \
		if [ ! -f CMakeCache.txt ]; then cmake \
			-DCMAKE_BUILD_TYPE="Release" \
			../../cahp-sim; fi && \
		$(MAKE) cahp-sim
	cp -a $(BUILDDIR)/cahp-sim/src/cahp-sim $(BUILDDIR)/bin/

yosys: prepare
	@echo "Building Yosys..."
	if [ ! -e "./$(BUILDDIR)/yosys" ]; then ln -s ${PWD}/yosys $(BUILDDIR)/yosys; fi
	cd $(BUILDDIR)/yosys && $(MAKE)

cahp-ruby: yosys prepare
	@echo "Building cahp-ruby..."
	cp -a cahp-ruby $(BUILDDIR)/
	cd $(BUILDDIR)/cahp-ruby && sbt run
	cd $(BUILDDIR)/cahp-ruby && \
		../yosys/yosys build.ys
	cp $(BUILDDIR)/cahp-ruby/vsp-core-ruby.json $(BUILDDIR)/share/kvsp/ruby-core.json

cahp-pearl: yosys prepare
	@echo "Building cahp-pearl..."
	cp -a cahp-pearl $(BUILDDIR)/
	cd $(BUILDDIR)/cahp-pearl && sbt run
	cd $(BUILDDIR)/cahp-pearl && \
		../yosys/yosys build.ys
	cp $(BUILDDIR)/cahp-pearl/vsp-core-pearl.json $(BUILDDIR)/share/kvsp/pearl-core.json

alexandrite: yosys prepare
	@echo "Building Alexandrite..."
	cp -a Alexandrite $(BUILDDIR)/
	cd $(BUILDDIR)/Alexandrite && sbt "runMain AlexandriteKVSPTop"
	cd $(BUILDDIR)/Alexandrite && \
		../yosys/yosys build.ys
	cp $(BUILDDIR)/Alexandrite/alexandrite-core.json $(BUILDDIR)/share/kvsp/alexandrite-core.json

llvm-cahp: prepare
	@echo "Building llvm-cahp..."
	mkdir -p $(BUILDDIR)/llvm-cahp
	cd $(BUILDDIR)/llvm-cahp && \
		if [ ! -f CMakeCache.txt ]; then cmake \
			-DCMAKE_BUILD_TYPE="Release" \
			-DCMAKE_POLICY_VERSION_MINIMUM=3.5 \
			-DLLVM_ENABLE_PROJECTS="lld;clang" \
			-DLLVM_TARGETS_TO_BUILD="RISCV" \
			-DLLVM_EXPERIMENTAL_TARGETS_TO_BUILD="CAHP" \
			../../llvm-cahp/llvm; fi && \
		$(MAKE)
	cp -a $(BUILDDIR)/llvm-cahp/bin/* $(BUILDDIR)/bin/

cahp-rt: llvm-cahp prepare
	@echo "Building cahp-rt..."
	cp -a cahp-rt $(BUILDDIR)/
	cd $(BUILDDIR)/cahp-rt && CC=../llvm-cahp/bin/clang $(MAKE)
	mkdir -p $(BUILDDIR)/share/kvsp/cahp-rt
	cd $(BUILDDIR)/cahp-rt && \
		cp -a crt0.o libc.a cahp.lds ../share/kvsp/cahp-rt/

alexandrite-rt: llvm-cahp prepare
	@echo "Building alexandrite-rt..."
	cp -a alexandrite-rt $(BUILDDIR)/
	cd $(BUILDDIR)/alexandrite-rt && CC=../llvm-cahp/bin/clang AR=../llvm-cahp/bin/llvm-ar $(MAKE)
	mkdir -p $(BUILDDIR)/share/kvsp/alexandrite-rt
	cd $(BUILDDIR)/alexandrite-rt && \
		cp -a crt0.o libc.a alexandrite.lds ../share/kvsp/alexandrite-rt/

clean:
	rm -rf $(BUILDDIR)
