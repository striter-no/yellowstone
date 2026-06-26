package main

import (
	"log"

	"github.com/go-gl/mathgl/mgl32"
	yst "github.com/striter-no/yellowstone"
)

func main() {
	// -- data setup
	verts := []yst.Vertex{
		{Pos: mgl32.Vec2{0, -0.5}, Color: mgl32.Vec3{1, 0, 1}},
		{Pos: mgl32.Vec2{0.5, 0.5}, Color: mgl32.Vec3{0, 1, 0}},
		{Pos: mgl32.Vec2{-0.5, 0.5}, Color: mgl32.Vec3{0, 0, 1}},
	}

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
		EnableValidation: true,
	}

	check(window.SetupWindow())
	defer window.Destroy()

	check(vdev.SetupVulkanDevice(app))
	defer vdev.Destroy()

	buf, err := yst.NewVertexBuffer(verts, vdev)
	check(err)
	defer buf.Destroy()

	swapchain := &yst.Swapchain{
		Device: vdev,
		VSync:  true,
	}

	pipeline := &yst.Pipeline{
		Device: vdev,
	}

	renderer := &yst.Renderer{
		SwapChain: swapchain,
		Pipeline:  pipeline,
		Device:    vdev,
		Vbuffer:   *buf,
	}

	check(swapchain.SetupSwapchain(window))
	defer swapchain.Destroy()

	check(pipeline.SetupPipeline("./assets/shaders/triangle/compiled/vert.spv", "./assets/shaders/triangle/compiled/frag.spv", swapchain))
	defer pipeline.Destroy()

	check(renderer.SetupRenderer(window))
	defer renderer.Destroy()

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
