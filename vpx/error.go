package vpx

import "errors"

func Error(err CodecErr) error {
	switch err {
	case CodecOk:
		return nil
	case CodecError:
		return ErrCodecUnknownError
	case CodecMemError:
		return ErrCodecMemError
	case CodecABIMismatch:
		return ErrCodecABIMismatch
	case CodecIncapable:
		return ErrCodecIncapable
	case CodecUnsupBitstream:
		return ErrCodecUnsupBitstream
	case CodecUnsupFeature:
		return ErrCodecUnsupFeature
	case CodecCorruptFrame:
		return ErrCodecCorruptFrame
	case CodecInvalidParam:
		return ErrCodecInvalidParam
	default:
		return ErrCodecUnknownError
	}
}

var (
	ErrCodecUnknownError   = errors.New("vpx: unknown error")
	ErrCodecMemError       = errors.New("vpx: memory error")
	ErrCodecABIMismatch    = errors.New("vpx: ABI mismatch")
	ErrCodecIncapable      = errors.New("vpx: incapable")
	ErrCodecUnsupBitstream = errors.New("vpx: unsupported bitstream")
	ErrCodecUnsupFeature   = errors.New("vpx: unsupported feature")
	ErrCodecCorruptFrame   = errors.New("vpx: corrupt frame")
	ErrCodecInvalidParam   = errors.New("vpx: invalid param")
)
