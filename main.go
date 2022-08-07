package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"image"
	"math"
	"os"
	"runtime"
	"time"
	"videoplayer/buttons"
	"videoplayer/multithread"
	"videoplayer/shaders"

	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/speaker"
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/zergon321/reisen"
)

var renderVertices = []float32{
	// top triangle
	-1.0, 1.0, 0.0, 1.0,
	1.0, 1.0, 1.0, 1.0,
	1.0, -1.0, 1.0, 0.0,

	// bottom triangle
	-1.0, 1.0, 0.0, 1.0,
	1.0, -1.0, 1.0, 0.0,
	-1.0, -1.0, 0.0, 0.0,
}

const (
	frameBufferSize                   = 24
	sampleRate                        = 44100
	channelCount                      = 2
	bitDepth                          = 8
	sampleBufferSize                  = 32 * channelCount * bitDepth * 24
	SpeakerSampleRate beep.SampleRate = 44100
)

var sampleSource *multithread.SharedBuffer

var isPlaying = true
var stopped = false
var soundCtrl *beep.Ctrl
var soundVolume *effects.Volume

type Video struct {
	ticker                 <-chan time.Time
	errs                   <-chan error
	frameBuffer            *multithread.SharedBuffer
	fps                    int
	videoTotalFramesPlayed int64
	videoTotalFrames       int64
	videoPlaybackFPS       int
	videoDuration          float64
	perSecond              <-chan time.Time
	last                   time.Time
	width                  int32
	height                 int32
}

var videoStream *reisen.VideoStream // temporary solution

var video = &Video{}

var buttonsBar *buttons.ButtonsBar

var videoPath string

func main() {
	filePath := flag.String("file", "", "path to the video file")
	flag.Parse()
	videoPath = *filePath
	if videoPath == "" {
		fmt.Println("File path is not specified")
		fmt.Println("Print `--help` to get more info")
		return
	}
	_, err := os.Stat(videoPath)
	if os.IsNotExist(err) {
		fmt.Printf("%v file does not exist\nPrint `--help` to get more info\n", videoPath)
		return
	}

	runtime.LockOSThread()

	err = glfw.Init()
	if err != nil {
		panic(err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.Resizable, glfw.True)
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

	window, err := glfw.CreateWindow(600, 600, "Video-Player", nil, nil)
	if err != nil {
		panic(err)
	}

	window.MakeContextCurrent()
	window.SetFramebufferSizeCallback(changeViewportSize)
	window.SetKeyCallback(keyCallback)
	window.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
	window.SetMouseButtonCallback(mouseCallback)
	// window.SetCursorPosCallback(mouseCallback)
	// window.SetScrollCallback(scrollCallback)

	err = gl.Init()
	if err != nil {
		panic(err)
	}

	videoShader := shaders.New("shaders/video.vs", "shaders/video.fs")
	buttonsShader := shaders.New("shaders/buttons.vs", "shaders/buttons.fs")

	var videoVAO, videoVBO, buttonsVAO, buttonsVBO uint32

	gl.GenVertexArrays(1, &videoVAO)
	gl.GenBuffers(1, &videoVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, videoVBO)
	gl.BufferData(gl.ARRAY_BUFFER, 4*len(renderVertices), gl.Ptr(renderVertices), gl.STATIC_DRAW)
	gl.BindVertexArray(videoVAO)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, 4*4, nil)
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointerWithOffset(1, 2, gl.FLOAT, false, 4*4, uintptr(8))
	gl.BindVertexArray(0)

	gl.GenVertexArrays(1, &buttonsVAO)
	gl.GenBuffers(1, &buttonsVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, buttonsVBO)
	gl.BufferData(gl.ARRAY_BUFFER, 4*len(renderVertices), gl.Ptr(renderVertices), gl.STATIC_DRAW)
	gl.BindVertexArray(buttonsVAO)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, 4*4, nil)
	// gl.EnableVertexAttribArray(1)
	// gl.VertexAttribPointerWithOffset(1, 2, gl.FLOAT, false, 4*4, uintptr(8))
	gl.BindVertexArray(0)

	err = video.Start(videoPath)
	handleError(err)

	// wWidth, wHeight := window.GetSize()
	// setViewport(int32(wWidth), int32(wHeight), video.width, video.height)
	texture := initTexture(video.width, video.height)

	var firstFrame, lastFrame *image.RGBA

	buttonsBar = buttons.NewButtonsBar(
		window,
		buttonsShader,
		buttonsVAO,
		true,
	)

	for !window.ShouldClose() {
		gl.ClearColor(0.0, 0.0, 0.0, 1.0)
		gl.Clear(gl.COLOR_BUFFER_BIT)

		// Render video
		shaders.Use(videoShader)
		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, texture)
		<-video.ticker // need to refactor
		if isPlaying {
			frame, _ := video.frameBuffer.Read()
			if frame != nil {
				video.videoTotalFramesPlayed++ // increase number of already played frames
				lastFrame = frame.(*image.RGBA)
			}
			// TODO:
			// After stream reaches the end it will be closed
			// In order to be able to start video from the beginning stream should be opened
		}
		if firstFrame == nil {
			firstFrame = lastFrame
		}
		if stopped {
			lastFrame = firstFrame
			video.videoTotalFramesPlayed = 0
		}
		gl.TexSubImage2D(gl.TEXTURE_2D, 0, 0, 0, video.width, video.height, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(lastFrame.Pix))
		gl.BindVertexArray(videoVAO)
		videoMatrix := getVideoMatrix(window, video.width, video.height)
		shaders.SetMat4(videoShader, "view", &videoMatrix)
		gl.DrawArrays(gl.TRIANGLES, 0, 6)
		gl.ActiveTexture(0)
		gl.BindVertexArray(0)

		// Render buttons
		videoProgress := getVideoProgress()
		buttonsBar.MoveScrollerHandle(videoProgress)
		buttonsBar.Draw()
		// buttons.DrawButtonsBar(window, buttonsShader, buttonsVAO)

		window.SwapBuffers()
		glfw.SwapInterval(1)
		glfw.PollEvents()
	}
}

func initTexture(width, height int32) uint32 {
	var textureID uint32
	gl.GenTextures(1, &textureID)

	gl.BindTexture(gl.TEXTURE_2D, textureID)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, width, height, 0, gl.RGBA, gl.UNSIGNED_BYTE, nil)
	gl.GenerateMipmap(gl.TEXTURE_2D)

	// gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
	// gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)

	return textureID
}

func changeViewportSize(window *glfw.Window, width, height int) {
	gl.Viewport(0, 0, int32(width), int32(height))
	buttonsBar.UpdatePos()
	// setViewport(int32(width), int32(height), video.width, video.height)
}

func getVideoMatrix(window *glfw.Window, vWidth, vHeight int32) mgl32.Mat4 {
	wWidth, wHeight := window.GetSize()
	windowAspectRatio := float64(wWidth) / float64(wHeight)
	videoAspectRatio := float64(vWidth) / float64(vHeight)
	var scaleX, scaleY float32
	if windowAspectRatio >= videoAspectRatio {
		scaleY = 1
		scaleX = float32(videoAspectRatio) / float32(windowAspectRatio)
	} else {
		scaleX = 1
		scaleY = float32(windowAspectRatio) / float32(videoAspectRatio)
	}
	viewMatrix := mgl32.Scale3D(scaleX, scaleY, 1)
	return viewMatrix
}

// func setViewport(wWidth, wHeight, vWidth, vHeight int32) {
// 	windowAspectRatio := float64(wWidth) / float64(wHeight)
// 	videoAspectRatio := float64(vWidth) / float64(vHeight)
// 	var startX, startY, width, height int32 = 0, 0, 0, 0
// 	if windowAspectRatio >= videoAspectRatio {
// 		height = wHeight
// 		width = int32(videoAspectRatio * float64(height))
// 		startX = (wWidth - width) / 2
// 	} else {
// 		width = wWidth
// 		height = int32(float64(width) / videoAspectRatio)
// 		startY = (wHeight - height) / 2
// 	}
// 	gl.Viewport(startX, startY, width, height)
// }

func readVideoAndAudio(media *reisen.Media) (
	*multithread.SharedBuffer,
	*multithread.SharedBuffer,
	int32,
	int32,
	chan error,
	error,
) {
	frameBuffer := multithread.NewSharedBuffer(frameBufferSize)
	sampleBuffer := multithread.NewSharedBuffer(sampleBufferSize)
	errs := make(chan error)

	err := media.OpenDecode()

	if err != nil {
		return nil, nil, 0, 0, nil, err
	}

	videoStream = media.VideoStreams()[0]
	err = videoStream.Open()

	if err != nil {
		return nil, nil, 0, 0, nil, err
	}

	videoWidth := int32(videoStream.Width())
	videoHeight := int32(videoStream.Height())

	audioStream := media.AudioStreams()[0]
	err = audioStream.Open()

	if err != nil {
		return nil, nil, videoWidth, videoHeight, nil, err
	}

	/*err = media.Streams()[0].Rewind(60 * time.Second)
	if err != nil {
		return nil, nil, nil, err
	}*/

	/*err = media.Streams()[0].ApplyFilter("h264_mp4toannexb")
	if err != nil {
		return nil, nil, nil, err
	}*/

	go func() {
		for {
			packet, gotPacket, err := media.ReadPacket()

			if err != nil {
				go func(err error) {
					errs <- err
				}(err)
			}

			if !gotPacket {
				break
			}

			/*hash := sha256.Sum256(packet.Data())
			fmt.Println(base58.Encode(hash[:]))*/

			switch packet.Type() {
			case reisen.StreamVideo:
				s := media.Streams()[packet.StreamIndex()].(*reisen.VideoStream)
				videoFrame, gotFrame, err := s.ReadVideoFrame()

				if err != nil {
					go func(err error) {
						errs <- err
					}(err)
				}

				if !gotFrame {
					break
				}

				if videoFrame == nil {
					continue
				}

				// frameBuffer <- videoFrame.Image()
				frameBuffer.Write(videoFrame.Image())

			case reisen.StreamAudio:
				s := media.Streams()[packet.StreamIndex()].(*reisen.AudioStream)
				audioFrame, gotFrame, err := s.ReadAudioFrame()

				if err != nil {
					go func(err error) {
						errs <- err
					}(err)
				}

				if !gotFrame {
					break
				}

				if audioFrame == nil {
					continue
				}

				// Turn the raw byte data into
				// audio samples of type [2]float64.
				reader := bytes.NewReader(audioFrame.Data())

				// See the README.md file for
				// detailed scheme of the sample structure.
				for reader.Len() > 0 {
					sample := [2]float64{0, 0}
					var result float64
					err = binary.Read(reader, binary.LittleEndian, &result)

					if err != nil {
						go func(err error) {
							errs <- err
						}(err)
					}

					sample[0] = result

					err = binary.Read(reader, binary.LittleEndian, &result)

					if err != nil {
						go func(err error) {
							errs <- err
						}(err)
					}

					sample[1] = result
					sampleBuffer.Write(sample)
				}
			}
		}

		videoStream.Close()
		audioStream.Close()
		media.CloseDecode()
		frameBuffer.Close()
		sampleBuffer.Close()
		close(errs)
	}()

	return frameBuffer, sampleBuffer, videoWidth, videoHeight, errs, nil
}

func (video *Video) Start(fname string) error {
	// Initialize the audio speaker.
	err := speaker.Init(sampleRate,
		SpeakerSampleRate.N(time.Second/10))

	if err != nil {
		return err
	}

	// Sprite for drawing video frames.
	// video.videoSprite, err = ebiten.NewImage(
	// 	width, height, ebiten.FilterDefault)

	// if err != nil {
	// 	return err
	// }

	// Open the media file.
	media, err := reisen.NewMedia(fname)

	if err != nil {
		return err
	}

	// Get the FPS for playing
	// video frames.
	videoFPS, _ := media.Streams()[0].FrameRate()

	// Get the total frames count
	videoTotalFramesCount := media.Streams()[0].FrameCount()

	if err != nil {
		return err
	}

	// SPF for frame ticker.
	spf := 1.0 / float64(videoFPS)
	frameDuration, err := time.
		ParseDuration(fmt.Sprintf("%fs", spf))

	if err != nil {
		return err
	}

	// Get video duration in seconds
	videoDuration := spf * float64(videoTotalFramesCount)

	// Start decoding streams.
	video.frameBuffer, sampleSource,
		video.width, video.height, video.errs, err = readVideoAndAudio(media)

	if err != nil {
		return err
	}

	// Start playing audio samples.
	streamer := streamSamples(sampleSource)
	soundCtrl = &beep.Ctrl{Streamer: streamer, Paused: false}
	soundVolume = &effects.Volume{
		Streamer: soundCtrl,
		Base:     2,
		Volume:   0,
		Silent:   false,
	}
	speaker.Play(soundVolume)

	video.ticker = time.Tick(frameDuration)

	// Setup metrics.
	video.last = time.Now()
	video.fps = 0
	video.perSecond = time.Tick(time.Second)
	video.videoTotalFramesPlayed = 0
	video.videoTotalFrames = videoTotalFramesCount
	video.videoDuration = videoDuration
	video.videoPlaybackFPS = 0

	return nil
}

func handleError(err error) {
	if err != nil {
		panic(err)
	}
}

func streamSamples(sampleSource *multithread.SharedBuffer) beep.Streamer {
	return beep.StreamerFunc(func(samples [][2]float64) (n int, ok bool) {
		numRead := 0

		for i := 0; i < len(samples); i++ {
			sample, ok := sampleSource.Read()

			if !ok {
				numRead = i + 1
				break
			}

			samples[i] = sample.([2]float64)
			numRead++
		}

		if numRead < len(samples) {
			return numRead, false
		}

		return numRead, true
	})
}

func keyCallback(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	if key == glfw.KeySpace && action == glfw.Release {
		playPause()
	}
}

func mouseCallback(w *glfw.Window, button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
	playButton := buttonsBar.GetButton("play")
	pauseButton := buttonsBar.GetButton("pause")
	stopButton := buttonsBar.GetButton("stop")
	scroller := buttonsBar.GetScroller()
	soundVolume := buttonsBar.GetSoundVolume()

	if button == glfw.MouseButtonLeft && action == glfw.Press {
		mouseX, mouseY := w.GetCursorPos()
		x := float32(mouseX)
		y := float32(mouseY)
		switch {
		case playButton.IsMouseOver(x, y) && !isPlaying:
			playPause()
		case pauseButton.IsMouseOver(x, y) && isPlaying:
			playPause()
		case stopButton.IsMouseOver(x, y):
			stop()
		// move video to position specified by mouse click
		// (maybe position will be set via dragging scroller button in the future)
		case scroller.IsMouseOver(x, y):
			scrollVideo(w, mouseX)
		case soundVolume.IsMouseOver(x, y):
			soundLevel := getSoundLevel(w, x)
			changeSoundVolume(soundLevel)
			buttonsBar.MoveSoundHandle(x)
		}
	}
}

func scrollVideo(w *glfw.Window, mouseX float64) {
	wWidth, _ := w.GetSize()
	// position of the scroller handle relative to window width in percents (from 0 to 1)
	scrollerHandlePos := mouseX / float64(wWidth)
	setVideoFramesPlayed(scrollerHandlePos)
	videoTimePos := getVideoTimePos(scrollerHandlePos)
	rewind(videoTimePos)
}

func playPause() {
	speaker.Lock()
	isPlaying = !isPlaying
	if stopped {
		stopped = false
	}
	soundCtrl.Paused = !soundCtrl.Paused
	speaker.Unlock()
}

func stop() {
	speaker.Lock()
	isPlaying = false
	stopped = true
	soundCtrl.Paused = true
	rewind(0)
	speaker.Unlock()
}

func rewind(t time.Duration) {
	// drain existing buffers
	video.frameBuffer.Purge()
	sampleSource.Purge()
	// next command will rewind video and audio streams
	videoStream.Rewind(t)
}

func getVideoProgress() float32 {
	// To bypass moving scroller inaccuracies related to differences between screen coords and number of frames
	if video.videoTotalFramesPlayed > video.videoTotalFrames {
		return 1
	}
	return float32(video.videoTotalFramesPlayed) / float32(video.videoTotalFrames)
}

func setVideoFramesPlayed(percent float64) {
	video.videoTotalFramesPlayed = int64(math.Ceil((float64(video.videoTotalFrames) * percent)))
}

func getVideoTimePos(percent float64) time.Duration {
	videoPosMs := video.videoDuration * percent * 1000
	videoTimePos, err := time.
		ParseDuration(fmt.Sprintf("%fms", videoPosMs))
	if err != nil {
		panic(err)
	}
	return videoTimePos
}

// level from 0 to 100
func changeSoundVolume(level float32) {
	if level <= 5 {
		soundVolume.Silent = true
	} else {
		soundVolume.Silent = false
	}
	soundVolume.Volume = float64(0.04*level - 2)
}

func getSoundLevel(w *glfw.Window, volumePos float32) float32 {
	wWidth, _ := w.GetSize()
	k := float32(100) / 120
	b := float32(140-wWidth) * k
	level := k*volumePos + b
	return level
}
