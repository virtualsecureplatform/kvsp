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

Download a KVSP release and unzip it.
(It has been compiled on Ubuntu 18.04 LTS. If it doesn't work in the following steps,
please read __Build__ section and try to build KVSP on your own.
It may be time-consuming, but not so hard.)

```
$ wget 'https://github.com/virtualsecureplatform/kvsp/releases/download/v5/kvsp_v5.tar.gz'
$ tar xf kvsp_v5.tar.gz
$ cd kvsp_v5/bin
```

Write some C code...

```
$ vim fib.c

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
```

...like so. This program (`fib.c`) computes the 5th term of the Fibonacci sequence, that is, 5.

Compile the C code (`fib.c`) to an executable file (`fib`).

```
$ ./kvsp cc fib.c -o fib
```

Let's check if the program is correct by emulator, which runs
without encryption.

```
$ ./kvsp emu fib
#1	done. (1001 us)
#2	done. (903 us)

...

#12	done. (827 us)
#13	done. (832 us)
break.
#cycle  13

...

x8  5

...
```

We can see `x8  5` here, so the program above seems correct.
Also we now know it takes 13 clocks by `#cycle  13`.

Now we will run the same program with encryption.

Generate a secret key (`secret.key`).

```
$ ./kvsp genkey -o secret.key
```

Encrypt `fib` with `secret.key` to get an encrypted executable file (`fib.enc`).

```
$ ./kvsp enc -k secret.key -i fib -o fib.enc
```

Run `fib.enc` for 13 clocks to get an encrypted result (`result.enc`).
Notice that we DON'T need the secret key (`secret.key`) here,
which means the encrypted program (`fib.enc`) runs without decryption!

```
$ ./kvsp run -i fib.enc -o result.enc -c 13 ## Use -g option if you have GPUs.
#1	done. (15726678 us)
#2	done. (15948790 us)

...

#12	done. (15699488 us)
#13	done. (16104918 us)
```

Decrypt `result.enc` with `secret.key` to print the result.

```
$ ./kvsp dec -k secret.key -i result.enc
...

x8  5

...
```

We could get the correct answer using secure computation!

## Build

Clone this repository:

```
$ git clone https://github.com/virtualsecureplatform/kvsp.git
```

Clone submodules recursively:

```
$ git submodule update --init --recursive
```

Build KVSP:

```
$ make  # It may take a while.
```

Use option `ENABLE_CUDA` if you build KVSP with GPU support:

```
$ make ENABLE_CUDA=1 CUDACXX="/usr/local/cuda/bin/nvcc" CUDAHOSTCXX="/usr/bin/clang-8"
```

## Build KVSP Using Docker

Based on Ubuntu 18.04 LTS image.

```
# docker build -t kvsp-build .
# docker run -it -v $PWD:/build -w /build kvsp-build:latest
```
