package buttons

// All coordinates in screen coord system e.g
// (0, 0) = (left, top) of the current window

import (
	"videoplayer/shaders"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

type ButtonsBar struct {
	visible        bool
	sh             uint32
	vao            uint32
	width          float32
	height         float32
	scroller       *Scroller
	scrollerHandle *Handle
	soundVolume    *SoundVolume
	soundHandle    *SoundHandle
	buttons        map[string]*Button
	window         *glfw.Window
	pos            *mgl32.Vec2 // (x1, y1) center of the buttonsBar in window coord system
	color          *mgl32.Vec4
	matrix         *mgl32.Mat4
}

func NewButtonsBar(
	w *glfw.Window,
	sh, vao uint32,
	visible bool,
) *ButtonsBar {
	wWidth, _ := w.GetSize()
	buttons := make(map[string]*Button)
	buttons["play"] = NewButton(
		w,
		"play",
		60,
		60,
		0, // offset from (width/2, y)
		&mgl32.Vec4{0, 1, 0, 1},
	)
	buttons["pause"] = NewButton(
		w,
		"pause",
		60,
		60,
		-90,
		&mgl32.Vec4{1, 0, 0, 1},
	)
	buttons["stop"] = NewButton(
		w,
		"stop",
		60,
		60,
		90,
		&mgl32.Vec4{0, 0, 1, 1},
	)
	scroller := NewScroller(
		w,
		float32(wWidth),
		10,
		&mgl32.Vec4{1, 0.1, 0.8, 1},
	)
	scrollerHandle := NewHandle(
		w,
		10,
		15,
		&mgl32.Vec4{0.2, 0, 1, 1},
	)
	soundVolume := NewSoundVolume(
		w,
		120,
		10,
		&mgl32.Vec4{0.2, 0.2, 1, 1},
	)
	soundHandle := NewSoundHandle(
		w,
		10,
		15,
		&mgl32.Vec4{1, 0.1, 0.8, 1},
	)
	buttonsBar := &ButtonsBar{
		width:          float32(wWidth),
		height:         60,
		visible:        true,
		scroller:       scroller,
		scrollerHandle: scrollerHandle,
		soundVolume:    soundVolume,
		soundHandle:    soundHandle,
		buttons:        buttons,
		window:         w,
		sh:             sh,
		vao:            vao,
		color:          &mgl32.Vec4{1, 1, 1, 1},
	}
	buttonsBar.UpdatePos()
	return buttonsBar
}

func (bb *ButtonsBar) Draw() {
	if !bb.visible {
		return
	}

	shaders.Use(bb.sh)
	gl.BindVertexArray(bb.vao)

	// render buttons bar
	shaders.SetMat4(bb.sh, "view", bb.matrix)
	shaders.SetVec4(bb.sh, "fColor", bb.color)
	gl.DrawArrays(gl.TRIANGLES, 0, 6)

	// render scroller
	scroller := bb.scroller
	shaders.SetMat4(bb.sh, "view", scroller.matrix)
	shaders.SetVec4(bb.sh, "fColor", scroller.color)
	gl.DrawArrays(gl.TRIANGLES, 0, 6)

	// render scroller handle
	scrollerHandle := bb.scrollerHandle
	shaders.SetMat4(bb.sh, "view", scrollerHandle.matrix)
	shaders.SetVec4(bb.sh, "fColor", scrollerHandle.color)
	gl.DrawArrays(gl.TRIANGLES, 0, 6)

	soundVolume := bb.soundVolume
	shaders.SetMat4(bb.sh, "view", soundVolume.matrix)
	shaders.SetVec4(bb.sh, "fColor", soundVolume.color)
	gl.DrawArrays(gl.TRIANGLES, 0, 6)

	soundHandle := bb.soundHandle
	shaders.SetMat4(bb.sh, "view", soundHandle.matrix)
	shaders.SetVec4(bb.sh, "fColor", soundHandle.color)
	gl.DrawArrays(gl.TRIANGLES, 0, 6)

	// render Play, Pause, Stop buttons
	for _, button := range bb.buttons {
		shaders.SetMat4(bb.sh, "view", button.matrix)
		shaders.SetVec4(bb.sh, "fColor", button.color)
		gl.DrawArrays(gl.TRIANGLES, 0, 6)
	}

	gl.BindVertexArray(0)
}

// Transform screen coordinates to OpenGL world view coordinates
func (bb *ButtonsBar) SetMatrix() {
	wWidth, wHeight := bb.window.GetSize()
	scaleX := 2 * bb.width / float32(wWidth)
	scaleY := bb.height / float32(wHeight)

	// transforms (0, 0) in screen coord to (-1, 1) in OpenGL coords
	translateX := 2*bb.pos.X()/float32(wWidth) - 1
	translateY := -2*bb.pos.Y()/float32(wHeight) + 1

	matrix := mgl32.Translate3D(translateX, translateY, 0).
		Mul4(mgl32.Scale3D(scaleX, scaleY, 1))
	bb.matrix = &matrix
}

func (bb *ButtonsBar) IsMouseOver(x, y float32) bool {
	xLeftEdge := bb.pos.X() - bb.width/2
	xRightEdge := bb.pos.X() + bb.width/2
	yTopEdge := bb.pos.Y() - bb.height/2
	yBottomEdge := bb.pos.Y() + bb.height/2

	if x >= xLeftEdge && x <= xRightEdge &&
		y >= yTopEdge && y <= yBottomEdge {
		return true
	}
	return false
}

func (bb *ButtonsBar) GetVisibility() bool {
	return bb.visible
}

func (bb *ButtonsBar) SetVisibility(visible bool) {
	bb.visible = visible
}

func (bb *ButtonsBar) GetButton(name string) *Button {
	return bb.buttons[name]
}

func (bb *ButtonsBar) GetScroller() *Scroller {
	return bb.scroller
}

func (bb *ButtonsBar) GetSoundVolume() *SoundVolume {
	return bb.soundVolume
}

func (bb *ButtonsBar) UpdatePos() {
	wWidth, wHeight := bb.window.GetSize()
	bb.pos = &mgl32.Vec2{float32(wWidth) / 2, float32(wHeight) - 30}
	bb.width = float32(wWidth)
	bb.SetMatrix()
	bb.scroller.UpdatePos()
	bb.scrollerHandle.UpdatePos()
	bb.soundVolume.UpdatePos()
	bb.soundHandle.UpdatePos()
	for _, button := range bb.buttons {
		button.UpdatePos()
	}
}

// Moves scroller handle along X axis
//
// x represents % of the full window width
//
// x=0 -> left corner
//
// x=1 -> right corner
func (bb *ButtonsBar) MoveScrollerHandle(x float32) {
	wWidth, _ := bb.window.GetSize()
	xPos := x * float32(wWidth)
	bb.scrollerHandle.Move(xPos)
}

// Moves sound volume handle along X axis

// x represents window coordinats
func (bb *ButtonsBar) MoveSoundHandle(x float32) {
	wWidth, _ := bb.window.GetSize()
	if int(x) >= wWidth-140 && int(x) <= wWidth-20 {
		bb.soundHandle.Move(x)
	}
}

type Button struct {
	name    string
	width   float32
	height  float32
	offsetX float32
	window  *glfw.Window
	pos     *mgl32.Vec2 // (x1, y1) center of the button in window coord system
	color   *mgl32.Vec4
	matrix  *mgl32.Mat4
}

func NewButton(
	w *glfw.Window,
	name string,
	width, height, offsetX float32,
	color *mgl32.Vec4,
) *Button {
	button := &Button{
		name:    name,
		width:   width,
		height:  height,
		offsetX: offsetX,
		color:   color,
		window:  w,
	}

	button.UpdatePos()
	return button
}

func (b *Button) SetMatrix() {
	wWidth, wHeight := b.window.GetSize()
	scaleX := b.width / float32(wWidth)
	scaleY := b.height / float32(wHeight)
	translateX := 2*b.pos.X()/float32(wWidth) - 1
	translateY := -2*b.pos.Y()/float32(wHeight) + 1
	matrix := mgl32.Translate3D(translateX, translateY, 0).
		Mul4(mgl32.Scale3D(scaleX, scaleY, 1))
	b.matrix = &matrix
}

func (b *Button) UpdatePos() {
	wWidth, wHeight := b.window.GetSize()
	b.pos = &mgl32.Vec2{float32(wWidth)/2 + b.offsetX, float32(wHeight) - 30}
	b.SetMatrix()
}

func (b *Button) IsMouseOver(x, y float32) bool {
	xLeftEdge := b.pos.X() - b.width/2
	xRightEdge := b.pos.X() + b.width/2
	yTopEdge := b.pos.Y() - b.height/2
	yBottomEdge := b.pos.Y() + b.height/2

	if x >= xLeftEdge && x <= xRightEdge &&
		y >= yTopEdge && y <= yBottomEdge {
		return true
	}
	return false
}

type Scroller struct {
	width  float32
	height float32
	window *glfw.Window
	pos    *mgl32.Vec2 // (x1, y1) center of the button in window coord system
	color  *mgl32.Vec4
	matrix *mgl32.Mat4
}

func NewScroller(
	w *glfw.Window,
	width, height float32,
	color *mgl32.Vec4,
) *Scroller {
	scroller := &Scroller{
		width:  width,
		height: height,
		color:  color,
		window: w,
	}

	scroller.UpdatePos()
	return scroller
}

func (s *Scroller) UpdatePos() {
	wWidth, wHeight := s.window.GetSize()
	s.pos = &mgl32.Vec2{float32(wWidth) / 2, float32(wHeight) - 70}
	s.width = float32(wWidth)
	s.SetMatrix()
}

func (s *Scroller) SetMatrix() {
	wWidth, wHeight := s.window.GetSize()
	scaleX := 2 * s.width / float32(wWidth)
	scaleY := s.height / float32(wHeight)

	// transforms (0, 0) in screen coord to (-1, 1) in OpenGL coords
	translateX := 2*s.pos.X()/float32(wWidth) - 1
	translateY := -2*s.pos.Y()/float32(wHeight) + 1

	matrix := mgl32.Translate3D(translateX, translateY, 0).
		Mul4(mgl32.Scale3D(scaleX, scaleY, 1))
	s.matrix = &matrix
}

func (s *Scroller) IsMouseOver(x, y float32) bool {
	xLeftEdge := s.pos.X() - s.width/2
	xRightEdge := s.pos.X() + s.width/2
	yTopEdge := s.pos.Y() - s.height/2
	yBottomEdge := s.pos.Y() + s.height/2

	if x >= xLeftEdge && x <= xRightEdge &&
		y >= yTopEdge && y <= yBottomEdge {
		return true
	}
	return false
}

type Handle struct {
	width  float32
	height float32
	window *glfw.Window
	pos    *mgl32.Vec2 // (x1, y1) center of the button in window coord system
	color  *mgl32.Vec4
	matrix *mgl32.Mat4
}

func NewHandle(
	w *glfw.Window,
	width, height float32,
	color *mgl32.Vec4,
) *Handle {
	handle := &Handle{
		width:  width,
		height: height,
		window: w,
		color:  color,
	}
	handle.UpdatePos()
	return handle
}

func (h *Handle) UpdatePos() {
	_, wHeight := h.window.GetSize()
	h.pos = &mgl32.Vec2{0, float32(wHeight) - 70}
	h.SetMatrix()
}

func (h *Handle) SetMatrix() {
	wWidth, wHeight := h.window.GetSize()
	scaleX := h.width / float32(wWidth)
	scaleY := h.height / float32(wHeight)
	translateX := 2*h.pos.X()/float32(wWidth) - 1
	translateY := -2*h.pos.Y()/float32(wHeight) + 1

	matrix := mgl32.Translate3D(translateX, translateY, 0).
		Mul4(mgl32.Scale3D(scaleX, scaleY, 1))
	h.matrix = &matrix
}

func (h *Handle) Move(x float32) {
	h.pos[0] = x
	h.SetMatrix()
}

type SoundVolume struct {
	width  float32
	height float32
	window *glfw.Window
	pos    *mgl32.Vec2 // (x1, y1) center of the button in window coord system
	color  *mgl32.Vec4
	matrix *mgl32.Mat4
}

func NewSoundVolume(
	w *glfw.Window,
	width, height float32,
	color *mgl32.Vec4,
) *SoundVolume {
	soundVolume := &SoundVolume{
		width:  width,
		height: height,
		color:  color,
		window: w,
	}

	soundVolume.UpdatePos()
	return soundVolume
}

func (sv *SoundVolume) UpdatePos() {
	wWidth, wHeight := sv.window.GetSize()
	sv.pos = &mgl32.Vec2{float32(wWidth) - 80, float32(wHeight) - 30}
	sv.SetMatrix()
}

func (sv *SoundVolume) SetMatrix() {
	wWidth, wHeight := sv.window.GetSize()
	scaleX := sv.width / float32(wWidth)
	scaleY := sv.height / float32(wHeight)
	translateX := 2*sv.pos.X()/float32(wWidth) - 1
	translateY := -2*sv.pos.Y()/float32(wHeight) + 1
	matrix := mgl32.Translate3D(translateX, translateY, 0).
		Mul4(mgl32.Scale3D(scaleX, scaleY, 1))
	sv.matrix = &matrix
}

func (sv *SoundVolume) IsMouseOver(x, y float32) bool {
	xLeftEdge := sv.pos.X() - sv.width/2
	xRightEdge := sv.pos.X() + sv.width/2
	yTopEdge := sv.pos.Y() - sv.height/2
	yBottomEdge := sv.pos.Y() + sv.height/2

	if x >= xLeftEdge && x <= xRightEdge &&
		y >= yTopEdge && y <= yBottomEdge {
		return true
	}
	return false
}

type SoundHandle struct {
	width  float32
	height float32
	window *glfw.Window
	pos    *mgl32.Vec2 // (x1, y1) center of the button in window coord system
	color  *mgl32.Vec4
	matrix *mgl32.Mat4
}

func NewSoundHandle(
	w *glfw.Window,
	width, height float32,
	color *mgl32.Vec4,
) *SoundHandle {
	handle := &SoundHandle{
		width:  width,
		height: height,
		window: w,
		color:  color,
	}
	handle.UpdatePos()
	return handle
}

func (sHandle *SoundHandle) UpdatePos() {
	wWidth, wHeight := sHandle.window.GetSize()
	sHandle.pos = &mgl32.Vec2{float32(wWidth) - 80, float32(wHeight) - 30}
	sHandle.SetMatrix()
}

func (sHandle *SoundHandle) SetMatrix() {
	wWidth, wHeight := sHandle.window.GetSize()
	scaleX := sHandle.width / float32(wWidth)
	scaleY := sHandle.height / float32(wHeight)
	translateX := 2*sHandle.pos.X()/float32(wWidth) - 1
	translateY := -2*sHandle.pos.Y()/float32(wHeight) + 1

	matrix := mgl32.Translate3D(translateX, translateY, 0).
		Mul4(mgl32.Scale3D(scaleX, scaleY, 1))
	sHandle.matrix = &matrix
}

func (sHandle *SoundHandle) Move(x float32) {
	sHandle.pos[0] = x
	sHandle.SetMatrix()
}
