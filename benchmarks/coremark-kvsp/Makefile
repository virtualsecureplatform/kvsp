KVSP ?= ../../build/bin/kvsp
COREMARK_DIR ?= ../coremark
OUT_DIR ?= ../../build/coremark-kvsp
BACKEND ?= iyokan
CPUS ?= ruby pearl alexandrite chrysoberyl

CFLAGS := -I. -I$(COREMARK_DIR) -ffunction-sections -fdata-sections \
	-Wl,--gc-sections
PROGRAMS := $(addprefix $(OUT_DIR)/matrix-mul-,$(CPUS))

.PHONY: all run clean

all: $(PROGRAMS)

$(OUT_DIR)/matrix-mul-%: matrix_mul_main.c core_portme.h \
		$(COREMARK_DIR)/core_matrix.c $(COREMARK_DIR)/coremark.h
	mkdir -p $(OUT_DIR)
	$(KVSP) cc --cpu $* $(CFLAGS) matrix_mul_main.c \
		$(COREMARK_DIR)/core_matrix.c -o $@

run: all
	KVSP=$(KVSP) OUT_DIR=$(OUT_DIR) BACKEND=$(BACKEND) ./run.sh $(CPUS)

clean:
	rm -f $(OUT_DIR)/matrix-mul-*
