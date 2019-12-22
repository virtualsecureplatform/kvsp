# KVSP; Kyoto Virtual Secure Platform

KVSP is the first virtual secure platform in the world,
which makes your life better.

On VSP you can run your encrypted code as is.
No need to decrypt while running. See [here](https://anqou.net/poc/2019/10/18/post-3106/)
for more details (in Japanese).

KVSP consists of many other sub-projects.
`kvsp` command, which this repository serves, is
a simple interface to use them easily.

## Quick Start

```
## Download a KVSP release and unzip it.
## (It has been compiled on Ubuntu 18.04 LTS. If it doesn't work in the following steps,
## please read sections below and try to build KVSP on your own.
## It may be time-consuming, but not so hard.)
$ wget 'https://github.com/virtualsecureplatform/kvsp/releases/download/v2/kvsp_v2.tar.gz'
$ tar xf kvsp_v2.tar.gz
$ cd kvsp_v2/bin

## Write some C code...
$ vim fib.c

## ...like so. This program computes the 5th term of the Fibonacci sequence, that is, 5.
$ cat fib.c
static int fib(int n) {
  int a = 0, b = 1;
  for (int i = 0; i < n; i++) {
    int tmp = a + b;
    a = b;
    b = tmp;
  }
  return a;
}

int main() {
  // The result will be set in the register x8.
  return fib(5);
}

## Compile the C code (`fib.c`) to an executable file (`fib`).
$ ./kvsp cc fib.c -o fib

## Let's check if the program is correct by emulator, which runs
## without encryption.
$ ./kvsp emu fib
LogicFile:/path/to/kvsp/share/kvsp/vsp-core.json
ResultFile:/tmp/389221298
Exec count:13
---Debug Output---

...

Reg 8 : 5

...

## We can see `Reg 8 : 5` here, so the program above seems correct.
## Also we now know it takes 13 clocks by `Exec count:13`.

## Now we will run the same program with encryption.

## Generate a secret key (`secret.key`).
$ ./kvsp genkey -o secret.key

## Encrypt `fib` with `secret.key` to get an encrypted executable file (`fib.enc`).
$ ./kvsp enc -k secret.key -i fib -o fib.enc

## Run `fib.enc` for 13 clocks to get an encrypted result (`result.enc`).
## Notice that we DON'T need the secret key (`secret.key`) here,
## which means the encrypted program (`fib.enc`) runs without decryption!
$ ./kvsp run -i fib.enc -o result.enc -c 13 ## Use -g option if you have GPUs.
LogicFile:/path/to/kvsp/share/kvsp/vsp-core.json
ResultFile:result.enc
ExecCycle:13
ThreadNum:17
CipherFile:fib.enc
Execution time 661857.385000[ms]
---Debug Output---
---Execution Stats---

...

## Decrypt `result.enc` with `secret.key` to print the result.
$ ./kvsp dec -k secret.key -i result.enc
...

Reg 8 : 5

...

## We could get the correct answer using secure computation!
```

## Build

```
## Clone this repository.
$ git clone https://github.com/virtualsecureplatform/kvsp.git

## Clone submodules recursively.
$ git submodule update --init --recursive

## Build KVSP. (It may take a while.)
$ make
```

## Build KVSP Using Docker

Based on Ubuntu 18.04 LTS image.

```
# docker build -t kvsp-build .
# docker run -it -v $PWD:/build -w /build kvsp-build:latest
```
