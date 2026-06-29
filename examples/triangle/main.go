package main

import (
	"log"
	"runtime"
	"runtime/debug"

	yst "github.com/striter-no/yellowstone"
	assets "github.com/striter-no/yellowstone/loader"
)

func main() {
	runtime.LockOSThread()
	debug.SetGCPercent(-1)

	// -- data setup
	meshes, err := assets.LoadOBJ("./assets/meshes/viking_room.obj")
	check(err)

	var verts []yst.Vertex
	var indices []uint32

	uniqueVertices := make(map[yst.Vertex]uint32)

	for _, m := range meshes.Meshes {
		for _, v := range m {
			vertex := yst.Vertex{
				Pos:   v.Pos,
				Color: v.Color,
				UV:    v.UV,
			}

			if index, exists := uniqueVertices[vertex]; exists {
				indices = append(indices, index)
			} else {
				newIndex := uint32(len(verts))
				uniqueVertices[vertex] = newIndex
				verts = append(verts, vertex)
				indices = append(indices, newIndex)
			}
		}
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

	tex, err := yst.NewTextureFromFile("./assets/textures/viking_room.png", renderer)
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
