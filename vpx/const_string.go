package vpx

func (fmt ImageFormat) String() string {
	switch fmt {
	case ImageFormatNone:
		return "NONE"
	case ImageFormatYv12:
		return "YV12"
	case ImageFormatI420:
		return "I420"
	case ImageFormatI422:
		return "I422"
	case ImageFormatI444:
		return "I444"
	case ImageFormatI440:
		return "I440"
	case ImageFormatI42016:
		return "I42016"
	case ImageFormatI42216:
		return "I42216"
	case ImageFormatI44416:
		return "I44416"
	case ImageFormatI44016:
		return "I44016"
	}
	return ""
}
