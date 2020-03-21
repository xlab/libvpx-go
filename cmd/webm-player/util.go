package main

import (
	"unsafe"

	"github.com/xlab/portaudio-go/portaudio"
)

func s(v string) string {
	return v + "\x00"
}

func b(v int32) bool {
	return v == 1
}

func paError(err portaudio.Error) bool {
	return portaudio.ErrorCode(err) != portaudio.PaNoError

}

func paErrorText(err portaudio.Error) string {
	return portaudio.GetErrorText(err)
}

func data(b []byte) string {
	hdr := (*sliceHeader)(unsafe.Pointer(&b))
	return *(*string)(unsafe.Pointer(&stringHeader{
		Data: hdr.Data,
		Len:  hdr.Len,
	}))
}

type sliceHeader struct {
	Data uintptr
	Len  int
	Cap  int
}

type stringHeader struct {
	Data uintptr
	Len  int
}
