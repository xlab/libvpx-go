package main

import (
	"log"
	"time"
	"unsafe"

	"github.com/xlab/closer"
	"github.com/xlab/portaudio-go/portaudio"
)

const sampleFormat = portaudio.PaFloat32
const samplesPerBuffer = 1024

func initAudio(channels, sampleRate int, syncC <-chan time.Duration, aOut <-chan Samples) {
	if err := portaudio.Initialize(); paError(err) {
		log.Println("PortAudio init error:", paErrorText(err))
		return
	}
	closer.Bind(func() {
		if err := portaudio.Terminate(); paError(err) {
			log.Println("PortAudio term error:", paErrorText(err))
		}
	})
	var stream *portaudio.Stream
	if err := portaudio.OpenDefaultStream(&stream, 0, int32(channels), sampleFormat, float64(sampleRate),
		samplesPerBuffer, paCallback(channels, syncC, aOut), nil); paError(err) {
		log.Println("PortAudio error:", paErrorText(err))
		return
	}
	closer.Bind(func() {
		if err := portaudio.CloseStream(stream); paError(err) {
			log.Println("[WARN] PortAudio error:", paErrorText(err))
		}
	})
	if err := portaudio.StartStream(stream); paError(err) {
		log.Println("PortAudio error:", paErrorText(err))
		return
	}
	closer.Bind(func() {
		if err := portaudio.StopStream(stream); paError(err) {
			closer.Fatalln("[WARN] PortAudio error:", paErrorText(err))
		}
	})
}

func paCallback(channels int, syncC <-chan time.Duration, aOut <-chan Samples) portaudio.StreamCallback {
	var start time.Time
	wait := time.NewTimer(time.Minute)
	wait.Stop()
	getSample := func() *Samples {
		for {
			select {
			case sample, ok := <-aOut:
				if !ok {
					return nil
				}
				if start.IsZero() {
					start = time.Now().Add(-sample.Timecode)
				}
				if d := time.Now().Sub(start); d < sample.Timecode {
					if sample.Timecode-d > 10*time.Second {
						continue
					}
					wait.Reset(sample.Timecode - d)
					select {
					case d := <-syncC:
						wait.Stop()
						start = time.Now().Add(-d)
						continue
					case <-wait.C:
					}
				} else if (d - sample.Timecode) > time.Second {
					continue
				}
				return &sample
			case d := <-syncC:
				start = time.Now().Add(-d)
				continue
			}
		}
	}

	return func(_ unsafe.Pointer, output unsafe.Pointer, sampleCount uint,
		_ *portaudio.StreamCallbackTimeInfo, _ portaudio.StreamCallbackFlags, _ unsafe.Pointer) int32 {

		const (
			statusContinue = int32(portaudio.PaContinue)
			statusComplete = int32(portaudio.PaComplete)
		)

		samples := getSample()
		if samples == nil {
			return statusComplete
		}

		out := (*(*[1 << 32]float32)(output))[:int(sampleCount)*channels]
		if len(samples.DataInterleaved) > 0 {
			copy(out, samples.DataInterleaved[:int(sampleCount)*channels])
			return statusContinue
		}
		if len(samples.Data) > int(sampleCount) {
			samples.Data = samples.Data[:sampleCount]
		}
		var idx int
		for _, sample := range samples.Data {
			if len(sample) > channels {
				sample = sample[:channels]
			}
			for i := range sample {
				out[idx] = sample[i]
				idx++
			}
		}

		return statusContinue
	}
}
