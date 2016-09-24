package vpx

import (
	"image"
	"unsafe"
)

func (img *Image) Image() *image.YCbCr {
	yw := uint32(img.Stride[PlaneY])
	cw := uint32(img.Stride[PlaneU])
	ysz := yw * img.DH
	csz := cw * img.DH

	subsampleRatio := image.YCbCrSubsampleRatio420
	switch img.Fmt {
	case ImageFormatI420:
		subsampleRatio = image.YCbCrSubsampleRatio420
		csz = csz / 2
	case ImageFormatI422, ImageFormatI42216:
		subsampleRatio = image.YCbCrSubsampleRatio422
		csz = csz / 2
	case ImageFormatI444, ImageFormat444a, ImageFormatI44416:
		subsampleRatio = image.YCbCrSubsampleRatio444
	case ImageFormatI440, ImageFormatI44016:
		subsampleRatio = image.YCbCrSubsampleRatio440
	}

	norm := &image.YCbCr{
		Y:  copyBytePtr(img.Planes[PlaneY], ysz),
		Cb: copyBytePtr(img.Planes[PlaneU], csz),
		Cr: copyBytePtr(img.Planes[PlaneV], csz),

		YStride:        int(img.Stride[PlaneY]),
		CStride:        int(img.Stride[PlaneU]),
		SubsampleRatio: subsampleRatio,
		Rect:           image.Rect(0, 0, int(img.DW), int(img.DH)),
	}
	return norm
}

func copyBytePtr(buf *byte, size uint32) []uint8 {
	dst := make([]uint8, size)
	src := (*(*[1 << 30]uint8)(unsafe.Pointer(buf)))[:size]
	copy(dst, src)
	return dst
}

// For 4:4:4, CStride == YStride/1 && len(Cb) == len(Cr) == len(Y)/1.
// For 4:2:2, CStride == YStride/2 && len(Cb) == len(Cr) == len(Y)/2.
// For 4:2:0, CStride == YStride/2 && len(Cb) == len(Cr) == len(Y)/4.
// For 4:4:0, CStride == YStride/1 && len(Cb) == len(Cr) == len(Y)/2.
// For 4:1:1, CStride == YStride/4 && len(Cb) == len(Cr) == len(Y)/4.
// For 4:1:0, CStride == YStride/4 && len(Cb) == len(Cr) == len(Y)/8.
