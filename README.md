glee
====

Glee is a symbolic execution engine for the Go programming language.


## Install

To install Glee, first download and install [Z3] using the `--staticlib` flag:

```sh
$ cd /path/to/z3
$ python scripts/mk_make.py --staticlib
$ cd build
$ make
$ sudo make install
```

Next, you can install `glee` using the Go toolchain:

```sh
$ cd /path/to/glee
$ go install ./cmd/glee
```


[Z3]: https://github.com/Z3Prover/z3