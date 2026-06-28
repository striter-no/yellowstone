package main

import (
	"log"
	"runtime"
	"runtime/debug"

	"github.com/go-gl/mathgl/mgl32"
	yst "github.com/striter-no/yellowstone"
)

func main() {
	runtime.LockOSThread()
	debug.SetGCPercent(-1)

	// -- data setup
	verts := []yst.Vertex{
		{Pos: mgl32.Vec2{-0.5, -0.5}, Color: mgl32.Vec3{1, 0, 0}, UV: mgl32.Vec2{1, 0}},
		{Pos: mgl32.Vec2{0.5, -0.5}, Color: mgl32.Vec3{0, 1, 0}, UV: mgl32.Vec2{0, 0}},
		{Pos: mgl32.Vec2{0.5, 0.5}, Color: mgl32.Vec3{0, 0, 1}, UV: mgl32.Vec2{0, 1}},
		{Pos: mgl32.Vec2{-0.5, 0.5}, Color: mgl32.Vec3{1, 1, 1}, UV: mgl32.Vec2{1, 1}},
	}

	indices := []uint16{0, 1, 2, 2, 3, 0}

	// -- rendering
	app := yst.AppInfo{
		Name:          "Yellowstone app",
		EngineName:    "Yellowstone",
		AppVersion:    "1.0.0",
		EngineVersion: "1.0.0",
		VulkanVersion: "1.3.0",
	}

	window := &yst.Window{
		Width:     800,
		Height:    600,
		Title:     "Yellowstone test",
		Resizable: true,
	}

	vdev := &yst.VulkanDevice{
		Window:           window,
		EnableValidation: false,
	}

	swapchain := &yst.Swapchain{
		Device: vdev,
		VSync:  true,
	}

	pipeline := &yst.Pipeline{
		Device: vdev,
	}

	sampler := &yst.TextureSampler{
		Device: vdev,
	}

	renderer := &yst.Renderer{
		Sampler:   sampler,
		SwapChain: swapchain,
		Pipeline:  pipeline,
		Device:    vdev,
	}

	check(window.SetupWindow())
	defer window.Destroy()

	check(vdev.SetupVulkanDevice(app))
	defer vdev.Destroy()

	check(swapchain.SetupSwapchain(window))
	defer swapchain.Destroy()

	check(pipeline.SetupPipeline("./assets/shaders/triangle/compiled/vert.spv", "./assets/shaders/triangle/compiled/frag.spv", swapchain))
	defer pipeline.Destroy()

	for i := range yst.MaxFramesInFlight {
		ubuf, err := yst.NewUniformBuffer(renderer)
		check(err)

		renderer.Ubuffers[i] = *ubuf
	}
	defer func() {
		for _, b := range renderer.Ubuffers {
			b.Destroy()
		}
	}()

	check(sampler.SetupTextureSampler())
	defer sampler.Destroy()

	renderer.Sampler = sampler

	check(renderer.SetupRenderer(window))
	defer renderer.Destroy()

	tex, err := yst.NewTextureFromFile("./assets/textures/statue.jpg", renderer)
	check(err)
	defer tex.Destroy()

	vbuf, err := yst.NewVertexBuffer(verts, renderer)
	check(err)
	defer vbuf.Destroy()

	ibuf, err := yst.NewIndexBuffer(indices, renderer)
	check(err)
	defer ibuf.Destroy()

	renderer.Vbuffer = *vbuf
	renderer.Ibuffer = *ibuf
	renderer.Texture = *tex

	check(renderer.SetupDescriptors())

	debug.SetGCPercent(100)
	for window.IsOpen() {
		if err := renderer.DrawFrame(); err != nil {
			log.Fatal(err)
		}

		window.PollEvents()
	}

	vdev.WaitIdle()
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
