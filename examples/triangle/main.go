package main

import (
	"log"

	yst "github.com/striter-no/yellowstone"
)

func main() {
	app := yst.AppInfo{
		Name:          "Yellowstone app",
		EngineName:    "Yellowstone",
		AppVersion:    "1.0.0",
		EngineVersion: "1.0.0",
		VulkanVersion: "1.3.0",
	}

	window := &yst.Window{
		Width:  800,
		Height: 600,
		Title:  "Yellowstone test",
	}

	vdev := &yst.VulkanDevice{
		Window:           window,
		EnableValidation: true,
	}

	swapchain := &yst.Swapchain{
		Device: vdev,
	}

	pipeline := &yst.Pipeline{
		Device: vdev,
	}

	renderer := &yst.Renderer{
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
