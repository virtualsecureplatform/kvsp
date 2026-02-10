SHELL=/bin/bash

### Config parameters.
ENABLE_CUDA=0
BUILDDIR := build

.PHONY: all prepare kvsp iyokan cahp-sim yosys cahp-ruby cahp-pearl llvm-cahp cahp-rt clean

all: kvsp iyokan cahp-sim cahp-ruby cahp-pearl cahp-rt
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
			-X main.iyokanRevision=$$(git -C ../Iyokan rev-parse --short HEAD || echo "unk") \
			-X main.cahpRubyRevision=$$(git -C ../cahp-ruby rev-parse --short HEAD || echo "unk") \
			-X main.cahpPearlRevision=$$(git -C ../cahp-pearl rev-parse --short HEAD || echo "unk") \
			-X main.cahpRtRevision=$$(git -C ../cahp-rt rev-parse --short HEAD || echo "unk") \
			-X main.cahpSimRevision=$$(git -C ../cahp-sim rev-parse --short HEAD || echo "unk") \
			-X main.llvmCahpRevision=$$(git -C ../llvm-cahp rev-parse --short HEAD || echo "unk") \
			-X main.yosysRevision=$$(git -C ../yosys rev-parse --short HEAD || echo "unk")"
	cp -a $(BUILDDIR)/kvsp/kvsp $(BUILDDIR)/bin/

iyokan: prepare
	@echo "Building Iyokan..."
	mkdir -p $(BUILDDIR)/Iyokan
	cd $(BUILDDIR)/Iyokan && \
		if [ ! -f CMakeCache.txt ]; then cmake \
			-DCMAKE_BUILD_TYPE="Release" \
			-DIYOKAN_ENABLE_CUDA=$(ENABLE_CUDA) \
			../../Iyokan; fi && \
		$(MAKE) iyokan iyokan-packet
	cp -a $(BUILDDIR)/Iyokan/bin/iyokan $(BUILDDIR)/bin/
	cp -a $(BUILDDIR)/Iyokan/bin/iyokan-packet $(BUILDDIR)/bin/

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

llvm-cahp: prepare
	@echo "Building llvm-cahp..."
	mkdir -p $(BUILDDIR)/llvm-cahp
	cd $(BUILDDIR)/llvm-cahp && \
		if [ ! -f CMakeCache.txt ]; then cmake \
			-DCMAKE_BUILD_TYPE="Release" \
			-DCMAKE_POLICY_VERSION_MINIMUM=3.5 \
			-DLLVM_ENABLE_PROJECTS="lld;clang" \
			-DLLVM_TARGETS_TO_BUILD="" \
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

clean:
	rm -rf $(BUILDDIR)
