package main

import (
	"time"

	"github.com/ebml-go/webm"
)

type Samples struct {
	Data     []float32
	Timecode time.Duration
	Rebase   bool
	EOS      bool
}

type ADecoder struct {
	src <-chan webm.Packet
}

type ACodec string

const (
	CodecVorbis ACodec = "A_VORBIS"
)

func NewADecoder(codec ACodec, src <-chan webm.Packet) *ADecoder {
	return &ADecoder{
		src: src,
	}
}

func (a *ADecoder) Process(out chan<- Samples) {
	defer close(out)
	for range a.src {

	}
}
