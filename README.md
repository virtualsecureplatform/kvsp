# KVSP; Kyoto Virtual Secure Platform

KVSP is a small interface to use a virtual secure platform simply,
which makes your life better.

Caveat: this project is under construction.

## Usage

```
## Compile C code (`foo.c`) to an executable file (`foo`).
$ kvsp cc foo.c -o foo

## Generate a secret key (`secret.key`).
$ kvsp genkey -out secret.key

## Encrypt `foo` with `secret.key` to get an encrypted executable file (`foo.enc`).
$ kvsp enc -inkey secret.key -in foo -out foo.enc

## Run `foo.enc` for 50 clocks to get an encrypted result (`result.enc`).
$ kvsp run -in foo.enc -out result.enc -clock 50

## Decrypt `result.enc` with `secret.key` to get its plaintext form (`result.data`).
$ kvsp dec -inkey secret.key -in result.enc -out result.data

## Show the result.
$ cat result.data
```
