package main

import (
	"image"
	"log"
	"time"

	"github.com/ebml-go/webm"
	"github.com/xlab/libvpx-go/vpx"
)

type Frame struct {
	*image.RGBA
	Timecode   time.Duration
	IsKeyframe bool
}

type VDecoder struct {
	enabled bool
	src     <-chan webm.Packet
	ctx     *vpx.CodecCtx
	iface   *vpx.CodecIface
}

type VCodec string

const (
	CodecVP8  VCodec = "V_VP8"
	CodecVP9  VCodec = "V_VP9"
	CodecVP10 VCodec = "V_VP10"
)

func NewVDecoder(codec VCodec, src <-chan webm.Packet) *VDecoder {
	dec := &VDecoder{
		src: src,
		ctx: vpx.NewCodecCtx(),
	}
	switch codec {
	case CodecVP8:
		dec.iface = vpx.DecoderIfaceVP8()
	case CodecVP9:
		dec.iface = vpx.DecoderIfaceVP9()
	default: // others are currently disabled
		log.Println("[WARN] unsupported VPX codec:", codec)
		return dec
	}
	err := vpx.Error(vpx.CodecDecInitVer(dec.ctx, dec.iface, nil, 0, vpx.DecoderABIVersion))
	if err != nil {
		log.Println("[WARN]", err)
		return dec
	}
	dec.enabled = true
	return dec
}

func (v *VDecoder) Process(out chan<- Frame) {
	defer close(out)
	for pkt := range v.src {
		if !v.enabled {
			continue
		}
		dataSize := uint32(len(pkt.Data))
		err := vpx.Error(vpx.CodecDecode(v.ctx, string(pkt.Data), dataSize, nil, 0))
		if err != nil {
			log.Println("[WARN]", err)
			continue
		}
		var iter vpx.CodecIter
		img := vpx.CodecGetFrame(v.ctx, &iter)
		for img != nil {
			img.Deref()
			out <- Frame{
				RGBA:     img.ImageRGBA(),
				Timecode: pkt.Timecode,
			}
			img = vpx.CodecGetFrame(v.ctx, &iter)
		}
	}
}
