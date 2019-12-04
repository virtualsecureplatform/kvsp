all: build/kvsp build/app build/cahp-sim build/tfheutil build/llvm-cahp

build/kvsp: FORCE
	mkdir -p build/
	cp kvsp build/

build/app: FORCE
	mkdir -p build/app
	cd app && go build
	cp app/app build/app/app

build/cahp-sim: FORCE
	mkdir -p build/cahp-sim
	make -C cahp-sim
	cp cahp-sim/cahp-sim build/cahp-sim/cahp-sim

build/tfheutil: build/lib/tfhe FORCE
	mkdir -p build/tfheutil
	cd tfheutil
	make -C tfheutil
	cp tfheutil/tfheutil build/tfheutil/tfheutil

build/llvm-cahp: FORCE
	mkdir -p build/llvm-cahp
	cd build/llvm-cahp && \
		cmake -G Ninja \
			-DLLVM_ENABLE_PROJECTS="lld;clang" \
			-DCMAKE_BUILD_TYPE="Release" \
			-DLLVM_TARGETS_TO_BUILD="" \
			-DLLVM_EXPERIMENTAL_TARGETS_TO_BUILD="CAHP" \
			../../llvm-cahp/llvm && \
		cmake --build .

build/lib/tfhe: FORCE
	mkdir -p tfhe/build
	cd tfhe/build && cmake ../src -DCMAKE_INSTALL_PREFIX=$(PWD)/tfhe/build && make && make install
	mkdir -p build/lib
	cp tfhe/build/libtfhe/*.so build/lib/

FORCE:

.PHONY: FORCE
