libvpx-go
=========

The package provides Go bindings for [libvpx](http://www.webmproject.org/code/), the WebM Project VPx codec implementation.
All the binding code has automatically been generated with rules defined in [vpx.yml](/vpx.yml).

### Usage

```
$ brew install libvpx
(or use your package manager)

$ go get github.com/xlab/libvpx-go/libvpx
```

### Rebuilding the package

You will need to get the [cgogen](https://git.io/cgogen) tool installed first.

```
$ git clone https://github.com/xlab/libvpx-go && cd libvpx-go
$ make clean
$ make
```
