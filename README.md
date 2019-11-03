# KVSP; Kyoto Virtual Secure Platform

KVSP is a small interface to use a virtual secure platform simply,
which makes your life better.

Caveat: this project is under construction.

## Usage

```
$ kvsp compile foo.c -o foo
$ kvsp encrypt foo cloud.enc secret.key
$ kvsp run cloud.enc result.enc
$ kvsp decrypt secret.key result.enc result.data
$ cat result.data
```
