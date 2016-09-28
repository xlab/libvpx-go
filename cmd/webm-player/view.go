package main

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"log"
	"os"
	"sync"
	"time"
	"unsafe"

	"github.com/go-gl/gl/v2.1/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/golang-ui/nuklear/nk"
)

const (
	winWidth  = 800
	winHeight = 500

	maxVertexBuffer  = 512 * 1024
	maxElementBuffer = 128 * 1024
)

const (
	assetBg = "assets/bg.png"
)

type View struct {
	win   *glfw.Window
	ctx   *nk.Context
	bgImg nk.Image

	width  uint
	height uint

	frame    *Frame
	frameTex uint32
	frameImg nk.Image
	frameMux *sync.RWMutex

	onPause     func()
	onSeek      func(d time.Duration)
	dur         time.Duration
	pos         time.Duration
	updatingPos bool
	vPos        time.Duration // video pos updated by decoder
	vPosLock    *sync.RWMutex
}

func NewView(win *glfw.Window, ctx *nk.Context, width, height uint, dur time.Duration) *View {
	v := &View{
		win:    win,
		ctx:    ctx,
		width:  width,
		height: height,
		dur:    dur,

		frameMux: new(sync.RWMutex),
		vPosLock: new(sync.RWMutex),
	}
	imgMap := loadImages(assetBg)
	if bgImg, ok := imgMap[assetBg]; ok {
		v.bgImg = bgImg
	}
	return v
}

func (v *View) ShowFrame(f *Frame) {
	v.frameMux.Lock()
	v.frame = f
	v.frameMux.Unlock()
}

func (v *View) GUILoop(exitC chan struct{}, doneC chan<- struct{}) {
	defer close(doneC)

	fpsTicker := time.NewTicker(rateLimitDur)
	for {
		select {
		case <-exitC:
			nk.NkGLFW3Shutdown()
			glfw.Terminate()
			fpsTicker.Stop()
			return
		case <-fpsTicker.C:
			if v.win.ShouldClose() {
				exitC <- struct{}{}
				continue
			}
			glfw.PollEvents()
			v.nkStep()
		}
	}
}

var windowName = s("view")

const panelHeight = 30

func (v *View) nkStep() {
	width, height := v.win.GetSize()
	nk.NkGLFW3NewFrame()

	// Layout
	panel := nk.NewPanel()
	bounds := nk.NkRect(0, 0, float32(width), float32(height))
	if nk.NkBegin(v.ctx, panel, windowName, bounds, nk.WindowNoScrollbar) > 0 {
		nk.NkWindowSetBounds(v.ctx, bounds)
		nk.NkWindowCollapse(v.ctx, windowName, nk.Maximized)
		viewWidth, viewHeight := letterbox(float32(v.width), float32(v.height),
			float32(width), float32(height)-panelHeight)

		v.frameMux.RLock()
		if v.frame == nil {
			// Draw logo if no frame yet
			nk.NkLayoutRowStatic(v.ctx, 200, 200, 1)
			nk.NkImage(v.ctx, v.bgImg)
		} else {
			// Display frame as image
			v.frameImg = rgbaTex(&v.frameTex, v.frame.RGBA)
			nk.NkLayoutRowStatic(v.ctx, viewHeight, int32(viewWidth), 1)
			nk.NkImage(v.ctx, v.frameImg)
		}
		v.frameMux.RUnlock()

		nk.NkLayoutRow(v.ctx, nk.Dynamic, 30, 2, []float32{0.85, 0.15})
		vPos, curPos, maxPos := v.getPos()
		v.makeSlider(v.ctx, vPos, curPos, maxPos)
		nk.NkLabel(v.ctx, v.time(), nk.TextAlignCentered)
	}
	nk.NkEnd(v.ctx)

	// Render
	gl.Viewport(0, 0, int32(width), int32(height))
	gl.Clear(gl.COLOR_BUFFER_BIT)
	gl.ClearColor(0, 0, 0, 255)
	nk.NkGLFW3Render(nk.AntiAliasingOn, maxVertexBuffer, maxElementBuffer)
	v.win.SwapBuffers()
}

func (v *View) time() string {
	return fmt.Sprintf("%v/%v\x00", v.pos, v.dur)
}

func (v *View) makeSlider(ctx *nk.Context, vPos, curPos, maxPos int32) {
	if !v.updatingPos {
		curPos = vPos
	}
	value := nk.NkSlideInt(v.ctx, 0, curPos, maxPos, 1)
	if value == 0 {
		return
	}
	if v.updatingPos && nk.NkInputIsMouseDown(v.ctx.Input(), 0) == 0 {
		v.updatingPos = false
		if v.onSeek != nil {
			v.onSeek(v.pos)
		}
	}
	v.setPos(time.Duration(value) * time.Second)
	if nk.NkInputIsMouseDown(v.ctx.Input(), 0) > 0 {
		v.updatingPos = true
	}
}

func (v *View) SetOnPause(fn func()) {
	v.onPause = fn
}

func (v *View) SetOnSeek(fn func(d time.Duration)) {
	v.onSeek = fn
}

func (v *View) UpdatePos(d time.Duration) {
	v.vPosLock.Lock()
	v.vPos = d
	v.vPosLock.Unlock()
}

func (v *View) setPos(d time.Duration) bool {
	if d > v.dur {
		v.pos = v.dur
		return false
	}
	if d != v.pos {
		v.pos = d
		return true
	}
	return false
}

func (v *View) getPos() (vPos, curPos, maxPos int32) {
	v.vPosLock.RLock()
	vPos = int32(v.vPos / time.Second)
	v.vPosLock.RUnlock()
	maxPos = int32(v.dur / time.Second)
	curPos = int32(v.pos / time.Second)
	return
}

func letterbox(contentW, contentH float32, boxW, boxH float32) (float32, float32) {
	ratio := contentH / contentW
	if contentW > boxW {
		contentH -= ratio * (contentW - boxW)
		contentW = boxW
		return contentW, contentH
	} else if contentW < boxW {
		contentH += ratio * (boxW - contentW)
		contentW = boxW
		return contentW, contentH
	}
	return boxW, boxW * ratio
}

func rgbaTex(tex *uint32, rgba *image.RGBA) nk.Image {
	if tex == nil {
		gl.GenTextures(1, tex)
	}
	gl.BindTexture(gl.TEXTURE_2D, *tex)
	gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_NEAREST)
	gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR_MIPMAP_NEAREST)
	gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, int32(rgba.Bounds().Dx()), int32(rgba.Bounds().Dy()),
		0, gl.RGBA, gl.UNSIGNED_BYTE, unsafe.Pointer(&rgba.Pix[0]))
	gl.GenerateMipmap(gl.TEXTURE_2D)
	return nk.NkImageId(int32(*tex))
}

func imgRGBA(img image.Image) *image.RGBA {
	switch trueim := img.(type) {
	case *image.RGBA:
		return trueim
	default:
		copy := image.NewRGBA(trueim.Bounds())
		draw.Draw(copy, trueim.Bounds(), trueim, image.Pt(0, 0), draw.Src)
		return copy
	}
}

func pngRGBA(path string) *image.RGBA {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return nil
	}
	return imgRGBA(img)
}

func loadImages(pngs ...string) map[string]nk.Image {
	imgMap := make(map[string]nk.Image, len(pngs))
	gl.Enable(gl.TEXTURE_2D)
	for _, path := range pngs {
		var tex uint32
		if img := pngRGBA(path); img != nil {
			imgMap[path] = rgbaTex(&tex, img)
		} else {
			log.Println("[WARN] failed to load", path)
		}
	}
	return imgMap
}
