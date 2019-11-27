# KVSP; Kyoto Virtual Secure Platform

KVSP is a small interface to use a virtual secure platform simply,
which makes your life better.

Caveat: this project is under construction.

## Usage

```
## Compile C code (`foo.c`) to an executable file (`foo`).
$ kvsp cc foo.c -o foo

## Generate a secret key (`secret.key`).
$ kvsp genkey -o secret.key

## Encrypt `foo` with `secret.key` to get an encrypted executable file (`foo.enc`).
$ kvsp enc -k secret.key -i foo -o foo.enc

## Run `foo.enc` for 50 clocks to get an encrypted result (`result.enc`).
$ kvsp run -i foo.enc -o result.enc -c 50

## Decrypt `result.enc` with `secret.key` to get its plaintext form (`result.data`).
$ kvsp dec -k secret.key -i result.enc -o result.data

## Show the result.
$ cat result.data
```
