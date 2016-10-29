package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"time"
	"unsafe"

	"github.com/go-gl/gl/v2.1/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/golang-ui/nuklear/nk"
	"github.com/xlab/closer"
)

var maxFps = flag.Int("fps", 30, "Limits the rendering FPS rate. Set this to 60fps for 720p60 videos")

const appName = "WebM VP8/VP9 Player"

var rateLimitDur time.Duration

func init() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "A simple WebM player with support of VP8/VP9 video and Vorbis/Opus audio. Version: v1.0rc1\n")
		fmt.Fprintf(os.Stderr, "Usage: %s <file1.webm> [file2.webm]\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "Specify files to read streams from, sometimes audio is stored in a separate file, use the optional argument for that.")
		flag.PrintDefaults()
	}
	flag.Parse()
	rateLimitDur = time.Second / time.Duration(*maxFps)
	runtime.LockOSThread()
}

func main() {
	// defer closer.Close()
	closer.Bind(func() {
		log.Println("Bye!")
	})

	// Init GUI
	if err := glfw.Init(); err != nil {
		closer.Fatalln(err)
	}
	glfw.WindowHint(glfw.ContextVersionMajor, 2)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	// glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile) // requires >= 3.2
	// glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True) // requires >= 3.0
	win, err := glfw.CreateWindow(winWidth, winHeight, s(appName), nil, nil)
	if err != nil {
		closer.Fatalln(err)
	}
	win.MakeContextCurrent()

	width, height := win.GetSize()
	log.Printf("glfw: created window %dx%d", width, height)

	if err := gl.Init(); err != nil {
		closer.Fatalln("opengl: init failed:", err)
	}
	gl.Viewport(0, 0, int32(width), int32(height))

	glfwWindow := unsafe.Pointer(win.GLFWWindow())
	ctx := nk.NkGLFW3Init((*nk.GLFWwindow)(glfwWindow), nk.GLFW3InstallCallbacks)
	atlas := nk.NewFontAtlas()
	nk.NkGLFW3FontStashBegin(&atlas)
	sansFont := nk.NkFontAtlasAddFromFile(atlas, s("assets/FreeSans.ttf"), 18, nil)
	nk.NkGLFW3FontStashEnd()
	if sansFont != nil {
		nk.NkStyleSetFont(ctx, sansFont.Handle())
	}

	// Open WebM files
	streams := make([]io.ReadSeeker, 0, 2)
	for _, opt := range flag.Args() {
		f, err := os.Open(opt)
		if err != nil {
			log.Println("[ERR] failed to open file:", err)
		}
		streams = append(streams, f)
		if len(streams) >= 2 {
			break
		}
	}
	stream1, stream2 := discoverStreams(streams...)
	vOut := make(chan Frame, 1)
	aOut := make(chan Samples, 1)
	if stream1 == nil {
		closer.Fatalln("[ERR] nothing to play")
	}

	var view *View
	if vtrack := stream1.Meta().FindFirstVideoTrack(); vtrack != nil {
		dur := stream1.Meta().Segment.GetDuration()
		view = NewView(win, ctx, vtrack.DisplayWidth, vtrack.DisplayHeight, dur)
	} else {
		view = NewView(win, ctx, 0, 0, 0)
	}
	view.SetOnSeek(func(d time.Duration) {
		stream1.Seek(d)
		if stream2 != nil {
			stream2.Seek(d)
		}
	})

	syncC := make(chan time.Duration, 10)
	// consume video stream
	if stream1.VDecoder() != nil {
		go stream1.VDecoder().Process(vOut)
		go initVideo(view, stream1.Rebase(), syncC, vOut)
	}

	aDec := stream1.ADecoder()
	if stream2 != nil {
		aDec = stream2.ADecoder()
	}
	// consume audio stream
	if aDec != nil {
		initAudio(aDec.Channels(), aDec.SampleRate(), syncC, aOut)
		go aDec.Process(aOut)
		closer.Bind(func() {
			aDec.Close()
		})
	}

	exitC := make(chan struct{}, 2)
	doneC := make(chan struct{}, 1)
	closer.Bind(func() {
		exitC <- struct{}{}
		<-doneC
	})
	view.GUILoop(exitC, doneC)
}

func initVideo(view *View, rebaseC <-chan time.Duration, syncC chan<- time.Duration, vOut <-chan Frame) {
	var pos time.Duration
	var last time.Time
	for {
		var frame Frame
		select {
		case f, ok := <-vOut:
			if !ok {
				return
			}
			if last.IsZero() {
				last = time.Now()
				pos = frame.Timecode
			} else {
				// advance pos
				now := time.Now()
				pos += now.Sub(last)
				last = now
			}
			frame = f
		case d := <-rebaseC:
			pos = d
			last = time.Now()
			syncC <- d
			continue
		}
		tc := frame.Timecode
		if pos < tc {
			if tc-pos > 10*time.Second {
				continue
			} else {
				time.Sleep(tc - pos)
			}
		} else if pos-tc > time.Second {
			continue
		}
		// draw a frame
		view.ShowFrame(&frame)
		view.UpdatePos(frame.Timecode)
		pos = frame.Timecode - time.Since(last)

		// log.Printf("video frame @ %v bounds = %v", frame.Timecode, frame.Rect)
	}
}

// discoverStreams returns both Video and Audio streams if in separate inputs,
// otherwise only the first stream would be returned (V/A/V+A).
func discoverStreams(streams ...io.ReadSeeker) (Stream, Stream) {
	if len(streams) == 0 {
		log.Println("[WARN] no streams found")
		return nil, nil
	} else if len(streams) == 1 {
		stream, err := NewStream(streams[0])
		if err != nil {
			log.Println("[WARN] failed to open stream:", err)
			return nil, nil
		}
		return stream, nil
	}
	var stream1Video bool
	var stream1Audio bool
	stream1, err := NewStream(streams[0])
	if err == nil {
		stream1Video = stream1.Meta().FindFirstVideoTrack() != nil
		stream1Audio = stream1.Meta().FindFirstAudioTrack() != nil
	} else {
		log.Println("[WARN] failed to open the first stream:", err)
	}
	if stream1Video && stream1Audio {
		log.Println("[INFO] found both Video+Audio in the first stream")
		return stream1, nil
	}
	var stream2Video bool
	var stream2Audio bool
	stream2, err := NewStream(streams[1])
	if err == nil {
		stream2Video = stream2.Meta().FindFirstVideoTrack() != nil
		stream2Audio = stream2.Meta().FindFirstAudioTrack() != nil
	} else {
		log.Println("[WARN] failed to open the second stream:", err)
	}
	switch {
	case stream1Video && stream2Audio:
		log.Println("[INFO] took Video from the first stream, Audio from the second")
		return stream1, stream2
	case stream1Audio && stream2Video:
		log.Println("[INFO] took Audio from the first stream, Video from the second")
		return stream2, stream1
	case stream1Video:
		log.Println("[INFO] took Video from the first stream, no Audio found")
		return stream1, nil
	case stream2Video:
		log.Println("[INFO] took Video from the second stream, no Audio found")
		return stream2, nil
	case stream1Audio:
		log.Println("[INFO] took Audio from the first stream, no Video found")
		return stream1, nil
	case stream2Audio:
		log.Println("[INFO] took Audio from the second stream, no Video found")
		return stream2, nil
	default:
		log.Println("[INFO] neither of Video or Audio found")
		return nil, nil
	}
}
