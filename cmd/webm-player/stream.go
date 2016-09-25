package main

import (
	"io"
	"log"
	"time"

	"github.com/ebml-go/webm"
	"github.com/xlab/closer"
)

func webmStream(r io.ReadSeeker, seek time.Duration) (meta webm.WebM, v *VDecoder, a *ADecoder) {
	reader, err := webm.Parse(r, &meta)
	if err != nil {
		closer.Fatalln("webm: stream parsing error:", err)
	}
	reader.Seek(seek)

	vtrack := meta.FindFirstVideoTrack()
	atrack := meta.FindFirstAudioTrack()
	vPackets := make(chan webm.Packet, 64)
	aPackets := make(chan webm.Packet, 64)
	if vtrack != nil {
		log.Printf("webm: found video track: %dx%d dur: %v %s", vtrack.DisplayWidth,
			vtrack.DisplayHeight, meta.Segment.GetDuration(), vtrack.CodecID)

		v = NewVDecoder(VCodec(vtrack.CodecID), vPackets)
	}
	if atrack != nil {
		log.Printf("webm: found audio track: ch: %d %.1fHz %d-bit, codec: %s", atrack.Channels,
			atrack.SamplingFrequency, atrack.BitDepth, atrack.CodecID)

		a = NewADecoder(ACodec(atrack.CodecID), aPackets)
	}
	go func() { // demuxer
		for pkt := range reader.Chan {
			switch {
			case vtrack == nil:
				aPackets <- pkt // audio only
			case atrack == nil:
				vPackets <- pkt // video only
			default:
				switch pkt.TrackNumber {
				case vtrack.TrackNumber:
					vPackets <- pkt
				case atrack.TrackNumber:
					aPackets <- pkt
				}
			}
		}
		close(vPackets)
		close(aPackets)
		reader.Shutdown()
	}()
	return
}
