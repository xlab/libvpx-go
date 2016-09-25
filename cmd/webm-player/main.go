package main

import (
	"flag"
	"log"
	"os"
	"runtime"
	"time"
	"unsafe"

	"github.com/go-gl/gl/v2.1/gl"
	"github.com/golang-ui/glfw"
	"github.com/golang-ui/nuklear/nk"
	"github.com/xlab/closer"
)

var webmInput = flag.String("webm", "video.webm", "Specify a .webm file to play")
var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var memprofile = flag.String("memprofile", "", "write memory profile to this file")

const appName = "WebM VP8/VP9 Player"

func init() {
	flag.Parse()
	runtime.LockOSThread()
}

func main() {
	defer closer.Close()

	// Init GUI
	glfw.SetErrorCallback(onError)
	if ok := b(glfw.Init()); !ok {
		closer.Fatalln("glfw: init failed")
	}
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenglProfile, glfw.OpenglCoreProfile)
	glfw.WindowHint(glfw.OpenglForwardCompat, glfw.True)
	win := glfw.CreateWindow(winWidth, winHeight, s(appName), nil, nil)
	if win == nil {
		closer.Fatalln("glfw: window creation failed")
	}
	glfw.MakeContextCurrent(win)

	var width, height int32
	glfw.GetWindowSize(win, &width, &height)
	log.Printf("glfw: created window %dx%d", width, height)

	if err := gl.Init(); err != nil {
		closer.Fatalln("opengl: init failed:", err)
	}
	gl.Viewport(0, 0, width, height)

	ctx := nk.NkGLFW3Init((*nk.GLFWwindow)(unsafe.Pointer(win)), nk.GLFW3InstallCallbacks)
	atlas := nk.NewFontAtlas()
	nk.NkGLFW3FontStashBegin(&atlas)
	// sansFont := nk.NkFontAtlasAddFromFile(atlas, s("assets/FreeSans.ttf"), 16, nil)
	nk.NkGLFW3FontStashEnd()
	// if sansFont != nil {
	// 	nk.NkStyleSetFont(ctx, sansFont.Handle())
	// }

	// Open WebM stream
	in, err := os.Open(*webmInput)
	if err != nil {
		closer.Fatalln("webm:", err)
	}
	meta, vDec, aDec := webmStream(in, 0)
	vOut := make(chan Frame, 64)
	aOut := make(chan Samples, 64)

	var view *View
	if vtrack := meta.FindFirstVideoTrack(); vtrack != nil {
		view = NewView(win, ctx, vtrack.DisplayWidth, vtrack.DisplayHeight)
	} else {
		view = NewView(win, ctx, 0, 0)
	}

	// consume video stream
	if vDec != nil {
		go vDec.Process(vOut)
		go func() {
			var start time.Time
			for frame := range vOut {
				if start.IsZero() {
					start = time.Now()
				}
				if d := time.Now().Sub(start); d < frame.Timecode {
					time.Sleep(frame.Timecode - d)
				}
				view.ShowFrame(&frame)
				log.Printf("video frame @ %v bounds = %v", frame.Timecode, frame.Rect)
			}
		}()
	}

	// consume audio stream
	if aDec != nil {
		go aDec.Process(aOut)
		go func() {
			if aDec == nil {
				return
			}

			aDec.Process(aOut)
			for range aOut {

			}
		}()
	}

	exitC := make(chan struct{}, 2)
	doneC := make(chan struct{}, 1)
	closer.Bind(func() {
		exitC <- struct{}{}
		<-doneC
	})
	view.GUILoop(exitC, doneC)
}

func onError(code int32, msg string) {
	log.Printf("[glfw ERR]: error %d: %s", code, msg)
}
