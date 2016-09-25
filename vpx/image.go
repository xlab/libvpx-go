package vpx

import (
	"image"
	"unsafe"
)

/*
#include <stdint.h>

void yuv420_to_rgb(uint16_t width, uint16_t height,
                 const uint8_t *y, const uint8_t *u, const uint8_t *v,
                 unsigned int ystride,
                 unsigned int ustride,
                 unsigned int vstride,
                 uint8_t *out)
{
    unsigned long int i, j;
    for (i = 0; i < height; ++i) {
        for (j = 0; j < width; ++j) {
            uint8_t *point = out + 4 * ((i * width) + j);
            int t_y = y[((i * ystride) + j)];
            int t_u = u[(((i / 2) * ustride) + (j / 2))];
            int t_v = v[(((i / 2) * vstride) + (j / 2))];
            t_y = t_y < 16 ? 16 : t_y;

            int r = (298 * (t_y - 16) + 409 * (t_v - 128) + 128) >> 8;
            int g = (298 * (t_y - 16) - 100 * (t_u - 128) - 208 * (t_v - 128) + 128) >> 8;
            int b = (298 * (t_y - 16) + 516 * (t_u - 128) + 128) >> 8;

            point[0] = r>255? 255 : r<0 ? 0 : r;
            point[1] = g>255? 255 : g<0 ? 0 : g;
            point[2] = b>255? 255 : b<0 ? 0 : b;
            point[3] = ~0;
        }
    }
}
*/
import "C"

func (img *Image) ImageRGBA() *image.RGBA {
	out := make([]uint8, img.DW*img.DH*4)
	C.yuv420_to_rgb(
		(C.uint16_t)(img.DW),
		(C.uint16_t)(img.DH),
		(*C.uint8_t)(img.Planes[PlaneY]),
		(*C.uint8_t)(img.Planes[PlaneU]),
		(*C.uint8_t)(img.Planes[PlaneV]),
		(C.uint)(img.Stride[PlaneY]),
		(C.uint)(img.Stride[PlaneU]),
		(C.uint)(img.Stride[PlaneV]),
		(*C.uint8_t)(unsafe.Pointer((*sliceHeader)(unsafe.Pointer(&out)).Data)),
	)
	return &image.RGBA{
		Pix:  out,
		Rect: image.Rect(0, 0, int(img.DW), int(img.DH)),
	}
}

func (img *Image) ImageYCbCr() *image.YCbCr {
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
