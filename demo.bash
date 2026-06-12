#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
DEMO_DIR=${KVSP_DEMO_DIR:-"$SCRIPT_DIR/demo-work"}
CPU=alexandrite
ARGUMENT=5
FIRST_CYCLES=30
RESUME_MARGIN=5
AUTO=0

while (($#)); do
    case "$1" in
        -y|--yes|--auto)
            AUTO=1
            ;;
        -h|--help)
            cat <<EOF
Usage: $0 [--yes]

Demonstrate the README flow using the current RISC-V/Alexandrite path:
compile fib.c with a system RISC-V compiler, encrypt it, generate keys,
run an encrypted program, resume it from a snapshot, and decrypt the result.

By default each step waits for Enter before executing.  Use --yes to run
without prompts.

The encrypted demo creates large files: the bootstrapping key is about 1.3 GB
with the current parameters, and encrypted packets are about 100 MB each.

Environment:
  KVSP=/path/to/kvsp             Override the kvsp executable.
  RISCV_CC=/path/to/compiler     Override the system RISC-V C compiler.
  KVSP_DEMO_DIR=/path/to/dir     Override the demo work directory.
  KVSP_GPU_COUNT=N               Override CUDA GPU count; use 0 for CPU.
EOF
            exit 0
            ;;
        *)
            printf 'Unknown option: %s\n' "$1" >&2
            exit 2
            ;;
    esac
    shift
done

fail() {
    printf 'demo.bash: %s\n' "$*" >&2
    exit 1
}

find_kvsp() {
    if [[ -n "${KVSP:-}" ]]; then
        [[ -x "$KVSP" ]] || fail "KVSP is not executable: $KVSP"
        printf '%s\n' "$KVSP"
        return
    fi

    local candidate
    for candidate in \
        "$SCRIPT_DIR/build/bin/kvsp" \
        "$SCRIPT_DIR/bin/kvsp"; do
        if [[ -x "$candidate" ]]; then
            printf '%s\n' "$candidate"
            return
        fi
    done

    while IFS= read -r candidate; do
        if [[ -x "$candidate" ]]; then
            printf '%s\n' "$candidate"
            return
        fi
    done < <(find "$SCRIPT_DIR" -maxdepth 2 -path "$SCRIPT_DIR/kvsp_v*/bin/kvsp" -type f 2>/dev/null | sort -Vr)

    fail "could not find kvsp; build it first or set KVSP=/path/to/kvsp"
}

find_riscv_cc() {
    if [[ -n "${RISCV_CC:-}" ]]; then
        command -v "$RISCV_CC" >/dev/null 2>&1 || [[ -x "$RISCV_CC" ]] || fail "RISCV_CC not found: $RISCV_CC"
        command -v "$RISCV_CC" 2>/dev/null || printf '%s\n' "$RISCV_CC"
        return
    fi

    local candidate
    for candidate in riscv32-unknown-elf-gcc riscv64-unknown-elf-gcc riscv64-linux-gnu-gcc clang; do
        if command -v "$candidate" >/dev/null 2>&1; then
            command -v "$candidate"
            return
        fi
    done

    fail "could not find a system RISC-V compiler; install one or set RISCV_CC"
}

find_size_tool() {
    if [[ -x "$KVSP_BIN_DIR/llvm-size" ]]; then
        printf '%s\n' "$KVSP_BIN_DIR/llvm-size"
        return
    fi
    if command -v llvm-size >/dev/null 2>&1; then
        command -v llvm-size
        return
    fi
    if command -v size >/dev/null 2>&1; then
        command -v size
        return
    fi

    fail "could not find llvm-size or size"
}

iyokan_has_cuda() {
    [[ -x "$KVSP_BIN_DIR/iyokan" ]] || return 1
    { "$KVSP_BIN_DIR/iyokan" -h 2>&1 || true; } | grep -q 'GPU support: enabled'
}

cuda_visible_devices_count() {
    [[ -n "${CUDA_VISIBLE_DEVICES+x}" ]] || return 1

    local value=${CUDA_VISIBLE_DEVICES// /}
    value=${value//$'\t'/}
    case "$value" in
        ''|-1|void|none|NoDevFiles)
            printf '0\n'
            return 0
            ;;
    esac

    local -a items
    local count=0
    local item
    IFS=',' read -r -a items <<< "$value"
    for item in "${items[@]}"; do
        if [[ -n "$item" ]]; then
            ((count += 1))
        fi
    done
    printf '%d\n' "$count"
}

detected_nvidia_gpu_count() {
    if command -v nvidia-smi >/dev/null 2>&1; then
        local count
        count=$(nvidia-smi --query-gpu=index --format=csv,noheader 2>/dev/null | sed '/^[[:space:]]*$/d' | wc -l)
        if [[ "$count" =~ ^[0-9]+$ ]] && ((count > 0)); then
            printf '%d\n' "$count"
            return 0
        fi
    fi

    find /dev -maxdepth 1 -type c -name 'nvidia[0-9]*' 2>/dev/null | wc -l
}

detect_cuda_gpus() {
    if [[ -n "${KVSP_GPU_COUNT:-}" ]]; then
        [[ "$KVSP_GPU_COUNT" =~ ^[0-9]+$ ]] || fail "KVSP_GPU_COUNT must be a non-negative integer"
        printf '%d\n' "$KVSP_GPU_COUNT"
        return
    fi

    local detected visible
    detected=$(detected_nvidia_gpu_count)
    [[ "$detected" =~ ^[0-9]+$ ]] || detected=0

    if visible=$(cuda_visible_devices_count); then
        [[ "$visible" =~ ^[0-9]+$ ]] || visible=0
        if ((detected > 0 && visible > detected)); then
            printf '%d\n' "$detected"
        else
            printf '%d\n' "$visible"
        fi
        return
    fi

    printf '%d\n' "$detected"
}

trim() {
    local value=$1
    value=${value#"${value%%[![:space:]]*}"}
    value=${value%"${value##*[![:space:]]}"}
    printf '%s\n' "$value"
}

cpu_product_name() {
    local name=
    if command -v lscpu >/dev/null 2>&1; then
        name=$(LC_ALL=C lscpu | sed -n 's/^Model name:[[:space:]]*//p' | head -n 1)
    fi
    if [[ -z "$name" && -r /proc/cpuinfo ]]; then
        name=$(sed -n 's/^model name[[:space:]]*:[[:space:]]*//p' /proc/cpuinfo | head -n 1)
    fi
    if [[ -n "$name" ]]; then
        printf '%s\n' "$name"
    else
        uname -m
    fi
}

cuda_visible_devices_value() {
    [[ -n "${CUDA_VISIBLE_DEVICES+x}" ]] || return 1

    local value=${CUDA_VISIBLE_DEVICES// /}
    value=${value//$'\t'/}
    case "$value" in
        ''|-1|void|none|NoDevFiles)
            printf '\n'
            return 0
            ;;
    esac
    printf '%s\n' "$value"
}

nvidia_smi_gpu_products() {
    command -v nvidia-smi >/dev/null 2>&1 || return 1

    local visible=
    local has_visible=0
    if visible=$(cuda_visible_devices_value); then
        has_visible=1
    fi

    local line index uuid name item matched
    local -a visible_items
    while IFS= read -r line; do
        IFS=',' read -r index uuid name <<< "$line"
        index=$(trim "$index")
        uuid=$(trim "$uuid")
        name=$(trim "$name")
        [[ -n "$name" ]] || continue

        if ((has_visible == 0)); then
            printf '%s\n' "$name"
            continue
        fi

        [[ -n "$visible" ]] || continue
        matched=0
        IFS=',' read -r -a visible_items <<< "$visible"
        for item in "${visible_items[@]}"; do
            if [[ "$item" == "$index" || "$item" == "$uuid" ]]; then
                matched=1
                break
            fi
        done
        if ((matched)); then
            printf '%s\n' "$name"
        fi
    done < <(nvidia-smi --query-gpu=index,uuid,name --format=csv,noheader,nounits 2>/dev/null)
}

driver_file_gpu_products() {
    local info name
    for info in /proc/driver/nvidia/gpus/*/information; do
        [[ -r "$info" ]] || continue
        name=$(sed -n 's/^Model:[[:space:]]*//p; s/^Product Name:[[:space:]]*//p' "$info" | head -n 1)
        [[ -n "$name" ]] && printf '%s\n' "$name"
    done
}

gpu_product_names() {
    local names=
    names=$({ nvidia_smi_gpu_products || true; } | awk 'NF && !seen[$0]++')
    if [[ -z "$names" ]]; then
        names=$({ driver_file_gpu_products || true; } | awk 'NF && !seen[$0]++')
    fi

    if [[ -n "$names" ]]; then
        awk 'NR > 1 { printf "; " } { printf "%s", $0 } END { printf "\n" }' <<< "$names"
    elif [[ -n "${CUDA_VISIBLE_DEVICES+x}" ]]; then
        printf 'none visible'
    else
        printf 'none detected'
    fi
}

quote_cmd() {
    local arg
    printf '$'
    for arg in "$@"; do
        printf ' %q' "$arg"
    done
    printf '\n'
}

pause() {
    if [[ "$AUTO" -eq 0 ]]; then
        printf '\nPress Enter to continue...'
        read -r _
    fi
}

run_cmd() {
    pause
    quote_cmd "$@"
    "$@"
}

run_in_demo() {
    pause
    printf '$ cd %q &&' "$DEMO_DIR"
    local arg
    for arg in "$@"; do
        printf ' %q' "$arg"
    done
    printf '\n'
    (cd "$DEMO_DIR" && "$@")
}

capture_in_demo() {
    local output=$1
    shift
    pause
    printf '$ cd %q &&' "$DEMO_DIR"
    local arg
    for arg in "$@"; do
        printf ' %q' "$arg"
    done
    printf ' > %q\n' "$output"
    (cd "$DEMO_DIR" && "$@" > "$output")
}

capture_all_in_demo() {
    local output=$1
    shift
    pause
    printf '$ cd %q &&' "$DEMO_DIR"
    local arg
    for arg in "$@"; do
        printf ' %q' "$arg"
    done
    printf ' > %q 2>&1\n' "$output"
    (cd "$DEMO_DIR" && "$@" > "$output" 2>&1)
}

show_command_text() {
    pause
    printf '$ %s\n' "$*"
}

size_one() {
    local path=$1
    if [[ -e "$DEMO_DIR/$path" ]]; then
        local bytes human
        bytes=$(stat -c '%s' "$DEMO_DIR/$path")
        if command -v numfmt >/dev/null 2>&1; then
            human=$(numfmt --to=iec --suffix=B "$bytes")
            printf '  %-22s %12s bytes  (%s)\n' "$path" "$bytes" "$human"
        else
            printf '  %-22s %12s bytes\n' "$path" "$bytes"
        fi
    else
        printf '  %-22s %s\n' "$path" '<missing>'
    fi
}

show_sizes() {
    printf '\nFile sizes:\n'
    local path
    for path in "$@"; do
        size_one "$path"
    done
}

fib_source() {
    cat <<'EOF'
static int fib(int n) {
  int a = 0, b = 1;
  for (int i = 0; i < n; i++) {
    int tmp = a + b;
    a = b;
    b = tmp;
  }
  return a;
}

int main(int argc, char **argv) {
  // Calculate n-th Fibonacci number.
  // n is a 1-digit number and given as command-line argument.
  return fib(argv[1][0] - '0');
}
EOF
}

write_fib() {
    pause
    printf '$ cd %q && cat > fib.c <<'\''EOF'\''\n' "$DEMO_DIR"
    fib_source
    printf 'EOF\n'
    fib_source > "$DEMO_DIR/fib.c"
}

KVSP_BIN=$(find_kvsp)
KVSP_BIN_DIR=$(cd -- "$(dirname -- "$KVSP_BIN")" && pwd)
KVSP_BIN="$KVSP_BIN_DIR/$(basename -- "$KVSP_BIN")"
SHARE_DIR=$(cd -- "$KVSP_BIN_DIR/../share/kvsp" 2>/dev/null && pwd) || fail "could not find share/kvsp next to $KVSP_BIN"
RT_DIR="$SHARE_DIR/alexandrite-rt"
[[ -f "$RT_DIR/crt0.o" && -f "$RT_DIR/libc.a" && -f "$RT_DIR/alexandrite.lds" ]] || fail "missing Alexandrite runtime files in $RT_DIR"

RISCV_CC_BIN=$(find_riscv_cc)
RISCV_CC_NAME=$(basename -- "$RISCV_CC_BIN")
SIZE_TOOL=$(find_size_tool)
TOOLS_DIR="$DEMO_DIR/tools"
case "$DEMO_DIR" in
    /|"$SCRIPT_DIR"|"$SCRIPT_DIR"/)
        fail "refusing to remove unsafe KVSP_DEMO_DIR: $DEMO_DIR"
        ;;
esac
CPU_PRODUCT=$(cpu_product_name)
GPU_PRODUCTS=$(gpu_product_names)
DETECTED_GPU_COUNT=$(detect_cuda_gpus)
GPU_COUNT=0
GPU_NOTE=
if iyokan_has_cuda; then
    GPU_COUNT=$DETECTED_GPU_COUNT
    if ((GPU_COUNT > 0)); then
        GPU_NOTE="CUDA enabled; using $GPU_COUNT GPU(s)"
    else
        GPU_NOTE="CUDA enabled in iyokan, but no visible GPU was detected"
    fi
else
    GPU_NOTE="CUDA not enabled in iyokan; using CPU"
fi

RUN_GPU_ARGS=()
RESUME_GPU_ARGS=()
if ((GPU_COUNT > 0)); then
    RUN_GPU_ARGS=(-iyokan-args --enable-gpu -iyokan-args --num-gpu -iyokan-args "$GPU_COUNT")
    RESUME_GPU_ARGS=(-iyokan-args --num-gpu -iyokan-args "$GPU_COUNT")
fi

printf 'KVSP demo using RISC-V/Alexandrite\n'
printf '  kvsp:      %s\n' "$KVSP_BIN"
printf '  runtime:   %s\n' "$RT_DIR"
printf '  compiler:  %s\n' "$RISCV_CC_BIN"
printf '  size tool: %s\n' "$SIZE_TOOL"
printf '  cpu:       %s\n' "$CPU_PRODUCT"
printf '  gpu:       %s\n' "$GPU_PRODUCTS"
printf '  cuda:      %s\n' "$GPU_NOTE"
printf '  work dir:  %s\n' "$DEMO_DIR"
printf '  note:      this demo writes a ~1.3 GB bootstrapping key\n'

run_cmd rm -rf "$DEMO_DIR"
run_cmd mkdir -p "$DEMO_DIR" "$TOOLS_DIR"

if [[ "$RISCV_CC_NAME" == clang* ]]; then
    if ! command -v ld.lld >/dev/null 2>&1; then
        [[ -x "$KVSP_BIN_DIR/lld" ]] || fail "clang needs ld.lld; install ld.lld or provide $KVSP_BIN_DIR/lld"
        run_cmd ln -sfn "$KVSP_BIN_DIR/lld" "$TOOLS_DIR/ld.lld"
        export PATH="$TOOLS_DIR:$PATH"
        printf 'Added %s to PATH so clang can find ld.lld.\n' "$TOOLS_DIR"
    fi
fi

write_fib

compile_cmd=()
if [[ "$RISCV_CC_NAME" == clang* ]]; then
    compile_cmd=(
        "$RISCV_CC_BIN"
        -target riscv32-unknown-elf
        -march=rv32i
        -mabi=ilp32
        -Oz
        -ffreestanding
        -fno-builtin
        -fno-unwind-tables
        -fno-asynchronous-unwind-tables
        -isystem "$RT_DIR"
        -fuse-ld=lld
        -nostdlib
        "$RT_DIR/crt0.o"
        fib.c
        "-Wl,-T,$RT_DIR/alexandrite.lds"
        -L "$RT_DIR"
        -lc
        -o fib
    )
else
    compile_cmd=(
        "$RISCV_CC_BIN"
        -march=rv32i
        -mabi=ilp32
        -Oz
        -ffreestanding
        -fno-builtin
        -fno-unwind-tables
        -fno-asynchronous-unwind-tables
        -isystem "$RT_DIR"
        -nostdlib
        "$RT_DIR/crt0.o"
        fib.c
        "-Wl,-T,$RT_DIR/alexandrite.lds"
        -L "$RT_DIR"
        -lc
        -o fib
    )
fi

run_in_demo "${compile_cmd[@]}"
run_in_demo file fib
printf 'The ELF file size includes headers, alignment, and symbol/string tables; section sizes show the actual contents.\n'
run_in_demo "$SIZE_TOOL" -A fib
show_sizes fib

capture_all_in_demo emu.txt "$KVSP_BIN" emu --cpu "$CPU" fib "$ARGUMENT"
run_in_demo awk '/^#cycle|^f0|^x10/' emu.txt
EMU_CYCLES=$(awk -F '\t' '$1 == "#cycle" { print $2; exit }' "$DEMO_DIR/emu.txt")
[[ "$EMU_CYCLES" =~ ^[0-9]+$ ]] || fail "could not read cycle count from $DEMO_DIR/emu.txt"
if (( EMU_CYCLES > FIRST_CYCLES )); then
    RESUME_CYCLES=$((EMU_CYCLES - FIRST_CYCLES + RESUME_MARGIN))
else
    RESUME_CYCLES=$RESUME_MARGIN
fi
printf 'Plain RISC-V run finished in %d cycles; encrypted resume will run %d more cycles after the first %d.\n' \
    "$EMU_CYCLES" "$RESUME_CYCLES" "$FIRST_CYCLES"

run_in_demo "$KVSP_BIN" version

run_in_demo "$KVSP_BIN" genkey -o secret.key
show_sizes secret.key

run_in_demo "$KVSP_BIN" genbkey -i secret.key -o bootstrapping.key
show_sizes secret.key bootstrapping.key

run_in_demo "$KVSP_BIN" enc --cpu "$CPU" -k secret.key -i fib -o fib.enc "$ARGUMENT"
show_sizes secret.key bootstrapping.key fib.enc

printf 'The encrypted run below prints cycle progress as #N and per-cycle elapsed time as done. (... us).\n'
run_in_demo "$KVSP_BIN" run --cpu "$CPU" "${RUN_GPU_ARGS[@]}" -bkey bootstrapping.key -i fib.enc -o result-30.enc -c "$FIRST_CYCLES" -snapshot run-30.snapshot
show_sizes fib.enc result-30.enc run-30.snapshot

capture_in_demo decrypt-30.txt "$KVSP_BIN" dec --cpu "$CPU" -k secret.key -i result-30.enc
run_in_demo awk '/^#cycle|^f0|^x10/' decrypt-30.txt

printf 'The resume step also prints cycle progress and per-cycle elapsed time.\n'
run_in_demo "$KVSP_BIN" resume "${RESUME_GPU_ARGS[@]}" -bkey bootstrapping.key -i run-30.snapshot -o result.enc -c "$RESUME_CYCLES" -snapshot run-final.snapshot
show_sizes result.enc run-final.snapshot

capture_in_demo decrypt-final.txt "$KVSP_BIN" dec --cpu "$CPU" -k secret.key -i result.enc
run_in_demo awk '/^#cycle|^f0|^x10/' decrypt-final.txt

printf '\nDone. For RISC-V, x10 is the return-value register; fib(%s) should decrypt to x10 = 5.\n' "$ARGUMENT"
