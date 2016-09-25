package vpx

/*
#cgo pkg-config: vpx
#include <vpx/vp8dx.h>
#include <vpx/vp8cx.h>
#include <vpx/vp9dx.h>
#include <vpx/vp9cx.h>
#include <stdlib.h>
*/
import "C"

func DecoderIfaceVP8() *CodecIface {
	return (*CodecIface)(C.vpx_codec_vp8_dx())
}

func DecoderIfaceVP9() *CodecIface {
	return (*CodecIface)(C.vpx_codec_vp9_dx())
}

// func DecoderIfaceVP10() *CodecIface {
// 	return (*CodecIface)(C.vpx_codec_vp10_dx())
// }

func DecoderFor(fourcc int) *CodecIface {
	switch fourcc {
	case Vp8Fourcc:
		return DecoderIfaceVP8()
	case Vp9Fourcc:
		return DecoderIfaceVP9()
	}
	return nil
}

func EncoderIfaceVP8() *CodecIface {
	return (*CodecIface)(C.vpx_codec_vp8_cx())
}

func EncoderIfaceVP9() *CodecIface {
	return (*CodecIface)(C.vpx_codec_vp9_cx())
}

// func EncoderIfaceVP10() *CodecIface {
// 	return (*CodecIface)(C.vpx_codec_vp10_cx())
// }

func EncoderFor(fourcc int) *CodecIface {
	switch fourcc {
	case Vp8Fourcc:
		return EncoderIfaceVP8()
	case Vp9Fourcc:
		return EncoderIfaceVP9()
	}
	return nil
}
