#!/bin/bash -xeu

failwith() {
    echo -e "\e[1;31m[ERROR]\e[0m $1" >&2
    exit 1
}

[ $# -ge 1 ] || failwith "Usage: $0 PATH-TO-BIN [ARGS-TO-KVSP-RUN]..."

KVSP=$1/kvsp
shift
KVSP_RUN_OPTIONS="$@"

# Check if KVSP is correct
[ -x "$KVSP" ] || failwith "Invalid executable of kvsp"

# Prepare keys
[ -f _test.sk ] || "$KVSP" genkey -o _test.sk
[ -f _test.bk ] || "$KVSP" genbkey -i _test.sk -o _test.bk

letstest() {
    cmdargs="$1"
    expected="$2"
    "$KVSP" cc _test.c -o _test.exe
    ncycles=$("$KVSP" emu _test.exe $cmdargs | grep "#cycle" | cut -f2)
    "$KVSP" enc -k _test.sk -i _test.exe -o _test.enc $cmdargs
    "$KVSP" run -bkey _test.bk -i _test.enc -o _test.res -c 1 -snapshot _test.snapshot $KVSP_RUN_OPTIONS
    "$KVSP" resume -bkey _test.bk -i _test.snapshot -o _test.res -snapshot _test.snapshot -c $ncycles
    result=$("$KVSP" dec -k _test.sk -i _test.res | grep x8 | cut -f2)
    [ $result -eq $expected ] || failwith "test failed"
}

"$KVSP" version

cat <<EOS > _test.c
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
EOS
letstest "5" "5"
