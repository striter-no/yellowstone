package yellowstone

import (
	"fmt"
	"log"

	"github.com/bbredesen/go-vk"
	"github.com/go-gl/glfw/v3.3/glfw"
)

const MaxFramesInFlight = 2

type Renderer struct {
	SwapChain *Swapchain
	Pipeline  *Pipeline
	Device    *VulkanDevice

	Vbuffer VertexBuffer

	// -- private
	commandPool         vk.CommandPool
	framebuffers        []vk.Framebuffer
	renderFinSemaphores []vk.Semaphore

	frames       [MaxFramesInFlight]FrameData
	currentFrame int
	fbResized    bool
}

func (r *Renderer) SetupRenderer(w *Window) error {

	if err := r.createFramebuffers(); err != nil {
		return fmt.Errorf("Failed to create framebuffers: %w", err)
	}

	if err := r.createCommandPool(w); err != nil {
		return fmt.Errorf("Failed to create command pool: %w", err)
	}

	if err := r.createSyncObjects(); err != nil {
		return fmt.Errorf("Failed to create render finished semaphores: %w", err)
	}

	for i := range MaxFramesInFlight {
		if err := r.frames[i].createCommandBuffer(r); err != nil {
			return err
		}

		if err := r.frames[i].createSyncObjects(r); err != nil {
			return err
		}
	}

	w.glfwWindow.SetFramebufferSizeCallback(
		func(w *glfw.Window, width, height int) {
			r.fbResized = true
		},
	)
	return nil
}

func (r *Renderer) recordCommandBuffer(
	cb vk.CommandBuffer,
	imageIndex uint32,
) error {
	beginInfo := vk.CommandBufferBeginInfo{
		Flags:            0,
		PInheritanceInfo: nil,
	}

	if err := vk.BeginCommandBuffer(cb, &beginInfo); err != nil {
		return err
	}

	clearColor := vk.ClearValue{
		Color: vk.ClearColorValue{
			TypeFloat32: [4]float32{0.0, 0.0, 0.0, 1.0},
		},
	}

	renderPassInfo := vk.RenderPassBeginInfo{
		RenderPass:  r.Pipeline.renderPass,
		Framebuffer: r.framebuffers[imageIndex],
		RenderArea: vk.Rect2D{
			Offset: vk.Offset2D{X: 0, Y: 0},
			Extent: r.SwapChain.extent,
		},
		PClearValues: []vk.ClearValue{clearColor},
	}

	vk.CmdBeginRenderPass(cb, &renderPassInfo, vk.SUBPASS_CONTENTS_INLINE)
	viewport := vk.Viewport{
		X: 0, Y: 0,
		Width:    float32(r.SwapChain.extent.Width),
		Height:   float32(r.SwapChain.extent.Height),
		MinDepth: 0,
		MaxDepth: 1,
	}

	scissor := vk.Rect2D{
		Offset: vk.Offset2D{},
		Extent: r.SwapChain.extent,
	}

	vk.CmdSetViewport(cb, 0, []vk.Viewport{viewport})
	vk.CmdSetScissor(cb, 0, []vk.Rect2D{scissor})

	vk.CmdBindPipeline(cb, vk.PIPELINE_BIND_POINT_GRAPHICS, r.Pipeline.handle)

	vBuffers := []vk.Buffer{r.Vbuffer.buffer}
	offsets := []vk.DeviceSize{0}
	vk.CmdBindVertexBuffers(cb, 0, vBuffers, offsets)

	vk.CmdDraw(cb, uint32(len(r.Vbuffer.data)), 1, 0, 0)

	vk.CmdEndRenderPass(cb)

	if err := vk.EndCommandBuffer(cb); err != nil {
		return err
	}

	return nil
}

func (r *Renderer) createCommandPool(window *Window) error {
	indices := FindQueueFamilies(r.Device.physical, window.surface)

	poolInfo := vk.CommandPoolCreateInfo{
		Flags:            vk.COMMAND_POOL_CREATE_RESET_COMMAND_BUFFER_BIT,
		QueueFamilyIndex: uint32(indices.graphicsFamily.Index),
	}

	cpool, err := vk.CreateCommandPool(r.Device.logical, &poolInfo, nil)
	if err != nil {
		return err
	}

	r.commandPool = cpool
	return nil
}

func (r *Renderer) createFramebuffers() error {
	r.framebuffers = make([]vk.Framebuffer, len(r.SwapChain.imageViews))

	for i, v := range r.SwapChain.imageViews {
		attachments := []vk.ImageView{v}

		framebufferInfo := vk.FramebufferCreateInfo{
			RenderPass:   r.Pipeline.renderPass,
			PAttachments: attachments,
			Width:        uint32(r.SwapChain.extent.Width),
			Height:       uint32(r.SwapChain.extent.Height),
			Layers:       1,
		}

		frameBuffer, err := vk.CreateFramebuffer(r.Device.logical, &framebufferInfo, nil)
		if err != nil {
			return err
		}

		r.framebuffers[i] = frameBuffer
	}

	return nil
}

func (r *Renderer) createSyncObjects() error {
	r.renderFinSemaphores = make([]vk.Semaphore, len(r.SwapChain.images))
	semaphoreInfo := vk.SemaphoreCreateInfo{}

	for i := range r.renderFinSemaphores {
		sem, err := vk.CreateSemaphore(r.Device.logical, &semaphoreInfo, nil)
		if err != nil {
			return err
		}
		r.renderFinSemaphores[i] = sem
	}
	return nil
}

func (r *Renderer) recreateSwapChain() error {
	w, h := r.Device.Window.glfwWindow.GetFramebufferSize()
	for w == 0 || h == 0 {
		log.Printf("got 0x0 size, waiting...")
		w, h = r.Device.Window.glfwWindow.GetFramebufferSize()
		glfw.WaitEvents()
	}

	vk.DeviceWaitIdle(r.Device.logical)

	r.cleanupSwapChain()

	// recreating
	if err := r.SwapChain.SetupSwapchain(r.Device.Window); err != nil {
		return err
	}

	if err := r.createFramebuffers(); err != nil {
		return err
	}

	if err := r.createSyncObjects(); err != nil {
		return err
	}

	return nil
}

func (r *Renderer) DrawFrame() error {
	frame := &r.frames[r.currentFrame]

	if err := vk.WaitForFences(r.Device.logical, []vk.Fence{frame.InFlightFence}, true, ^uint64(0)); err != nil {
		return fmt.Errorf("WaitForFences failed: %w", err)
	}

	imageIndex, err := vk.AcquireNextImageKHR(
		r.Device.logical,
		r.SwapChain.handle,
		^uint64(0),
		frame.ImageAvailableSemaphore,
		vk.Fence(vk.NULL_HANDLE),
	)

	if err != nil {
		res, ok := err.(vk.Result)
		if ok && res == vk.ERROR_OUT_OF_DATE_KHR {
			if err := r.recreateSwapChain(); err != nil {
				return fmt.Errorf("Recreate swap chain failed: %w", err)
			}
			return nil
		} else if !ok || (ok && res != vk.SUBOPTIMAL_KHR) {
			return fmt.Errorf("AcquireNextImageKHR failed: %w", err)
		}
	}

	if err := vk.ResetFences(r.Device.logical, []vk.Fence{frame.InFlightFence}); err != nil {
		return fmt.Errorf("ResetFences failed: %w", err)
	}

	if err := vk.ResetCommandBuffer(frame.CommandBuffer, 0); err != nil {
		return fmt.Errorf("ResetCommandBuffer failed: %w", err)
	}

	if err := r.recordCommandBuffer(frame.CommandBuffer, imageIndex); err != nil {
		return fmt.Errorf("recordCommandBuffer failed: %w", err)
	}

	waitSemaphores := []vk.Semaphore{frame.ImageAvailableSemaphore}
	waitStages := []vk.PipelineStageFlags{vk.PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT}
	signalSemaphores := []vk.Semaphore{r.renderFinSemaphores[imageIndex]}

	submitInfo := vk.SubmitInfo{
		PWaitSemaphores:   waitSemaphores,
		PWaitDstStageMask: waitStages,
		PCommandBuffers:   []vk.CommandBuffer{frame.CommandBuffer},
		PSignalSemaphores: signalSemaphores,
	}

	if err := vk.QueueSubmit(r.Device.graphicsQueue, []vk.SubmitInfo{submitInfo}, frame.InFlightFence); err != nil {
		return fmt.Errorf("QueueSubmit failed: %w", err)
	}

	swapChains := []vk.SwapchainKHR{r.SwapChain.handle}

	presentInfo := vk.PresentInfoKHR{
		PWaitSemaphores: signalSemaphores,
		PSwapchains:     swapChains,
		PImageIndices:   []uint32{imageIndex},
		PResults:        nil,
	}

	err = vk.QueuePresentKHR(r.Device.presentQueue, &presentInfo)

	var needRecreate bool
	if err != nil {
		res, ok := err.(vk.Result)
		if ok && (res == vk.SUBOPTIMAL_KHR || res == vk.ERROR_OUT_OF_DATE_KHR) {
			needRecreate = true
		} else if !ok || res != vk.SUCCESS {
			return fmt.Errorf("QueuePresentKHR failed: %w", err)
		}
	}

	if r.fbResized || needRecreate {
		r.fbResized = false
		if err := r.recreateSwapChain(); err != nil {
			return fmt.Errorf("recreateSwapChain failed: %w", err)
		}
	}

	r.currentFrame = (r.currentFrame + 1) % MaxFramesInFlight
	return nil
}

func (r *Renderer) cleanupSwapChain() {
	for _, fb := range r.framebuffers {
		vk.DestroyFramebuffer(r.Device.logical, fb, nil)
	}
	r.framebuffers = nil

	for _, sem := range r.renderFinSemaphores {
		vk.DestroySemaphore(r.Device.logical, sem, nil)
	}
	r.renderFinSemaphores = nil

	r.SwapChain.Destroy()
}

func (r *Renderer) Destroy() {
	r.cleanupSwapChain()

	for _, f := range r.frames {
		f.Destroy(r.Device.logical)
	}

	vk.DestroyCommandPool(r.Device.logical, r.commandPool, nil)
}

// frames ---

type FrameData struct {
	CommandBuffer vk.CommandBuffer

	ImageAvailableSemaphore vk.Semaphore
	InFlightFence           vk.Fence
}

func (f *FrameData) createCommandBuffer(r *Renderer) error {
	allocInfo := vk.CommandBufferAllocateInfo{
		CommandPool:        r.commandPool,
		Level:              vk.COMMAND_BUFFER_LEVEL_PRIMARY,
		CommandBufferCount: 1,
	}

	cbuffers, err := vk.AllocateCommandBuffers(r.Device.logical, &allocInfo)
	if err != nil {
		return err
	}

	f.CommandBuffer = cbuffers[0]
	return nil
}

func (f *FrameData) createSyncObjects(r *Renderer) error {
	semaphoreInfo := vk.SemaphoreCreateInfo{}
	fenceInfo := vk.FenceCreateInfo{
		Flags: vk.FENCE_CREATE_SIGNALED_BIT,
	}

	imgSemaphore, err := vk.CreateSemaphore(r.Device.logical, &semaphoreInfo, nil)
	if err != nil {
		return err
	}

	fence, err := vk.CreateFence(r.Device.logical, &fenceInfo, nil)
	if err != nil {
		return err
	}

	f.ImageAvailableSemaphore = imgSemaphore
	f.InFlightFence = fence

	return nil
}

func (f *FrameData) Destroy(logicalDev vk.Device) {
	vk.DestroySemaphore(logicalDev, f.ImageAvailableSemaphore, nil)
	vk.DestroyFence(logicalDev, f.InFlightFence, nil)
}
