package yellowstone

import (
	"fmt"

	"github.com/bbredesen/go-vk"
)

const MaxFramesInFlight = 2

type Renderer struct {
	SwapChain *Swapchain
	Pipeline  *Pipeline
	Device    *VulkanDevice

	// -- private
	commandPool  vk.CommandPool
	framebuffers []vk.Framebuffer

	frames       [MaxFramesInFlight]FrameData
	currentFrame int
}

func (r *Renderer) SetupRenderer(w *Window) error {

	if err := r.createFramebuffers(); err != nil {
		return fmt.Errorf("Failed to create framebuffers: %w", err)
	}

	if err := r.createCommandPool(w); err != nil {
		return fmt.Errorf("Failed to create command pool: %w", err)
	}

	for i := range MaxFramesInFlight {
		if err := r.frames[i].createCommandBuffer(r); err != nil {
			return err
		}

		if err := r.frames[i].createSyncObjects(r); err != nil {
			return err
		}
	}

	return nil
}

func (r *Renderer) DrawFrame() error {
	frame := &r.frames[r.currentFrame]

	if err := vk.WaitForFences(r.Device.logical, []vk.Fence{frame.InFlightFence}, true, ^uint64(0)); err != nil {
		return fmt.Errorf("WaitForFences failed: %w", err)
	}

	if err := vk.ResetFences(r.Device.logical, []vk.Fence{frame.InFlightFence}); err != nil {
		return fmt.Errorf("ResetFences failed: %w", err)
	}

	imageIndex, err := vk.AcquireNextImageKHR(
		r.Device.logical,
		r.SwapChain.handle,
		^uint64(0),
		frame.ImageAvailableSemaphore,
		vk.Fence(vk.NULL_HANDLE),
	)

	if err != nil {
		if res, ok := err.(vk.Result); ok && res == vk.SUBOPTIMAL_KHR {
		} else {
			return fmt.Errorf("AcquireNextImageKHR failed: %w", err)
		}
	}

	if err := vk.ResetCommandBuffer(frame.CommandBuffer, 0); err != nil {
		return fmt.Errorf("ResetCommandBuffer failed: %w", err)
	}

	if err := r.recordCommandBuffer(frame.CommandBuffer, imageIndex); err != nil {
		return fmt.Errorf("recordCommandBuffer failed: %w", err)
	}

	waitSemaphores := []vk.Semaphore{frame.ImageAvailableSemaphore}
	waitStages := []vk.PipelineStageFlags{vk.PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT}

	currentRenderFinishedSemaphore := frame.RenderFinishedSemaphores[imageIndex]
	signalSemaphores := []vk.Semaphore{currentRenderFinishedSemaphore}

	submitInfo := vk.SubmitInfo{
		PWaitSemaphores:   waitSemaphores,
		PWaitDstStageMask: waitStages,
		PCommandBuffers:   []vk.CommandBuffer{frame.CommandBuffer},
		PSignalSemaphores: signalSemaphores,
	}

	if err := vk.QueueSubmit(r.Device.graphicsQueue, []vk.SubmitInfo{submitInfo}, frame.InFlightFence); err != nil {
		return err
	}

	swapChains := []vk.SwapchainKHR{r.SwapChain.handle}

	presentInfo := vk.PresentInfoKHR{
		PWaitSemaphores: signalSemaphores,
		PSwapchains:     swapChains,
		PImageIndices:   []uint32{imageIndex},
		PResults:        nil,
	}

	err = vk.QueuePresentKHR(r.Device.presentQueue, &presentInfo)

	if err != nil {
		if res, ok := err.(vk.Result); ok && res == vk.SUBOPTIMAL_KHR {
		} else {
			return fmt.Errorf("QueuePresentKHR failed: %w", err)
		}
	}

	r.currentFrame = (r.currentFrame + 1) % MaxFramesInFlight
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
	vk.CmdBindPipeline(cb, vk.PIPELINE_BIND_POINT_GRAPHICS, r.Pipeline.handle)

	viewport := vk.Viewport{
		X: 0, Y: 0,
		Width:    float32(r.SwapChain.extent.Width),
		Height:   float32(r.SwapChain.extent.Height),
		MinDepth: 0,
		MaxDepth: 1,
	}

	vk.CmdSetViewport(cb, 0, []vk.Viewport{viewport})

	scissor := vk.Rect2D{
		Offset: vk.Offset2D{},
		Extent: r.SwapChain.extent,
	}

	vk.CmdSetScissor(cb, 0, []vk.Rect2D{scissor})
	vk.CmdDraw(cb, 3, 1, 0, 0)

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

func (r *Renderer) Destroy() {
	for _, f := range r.frames {
		f.Destroy(r.Device.logical)
	}

	vk.DestroyCommandPool(r.Device.logical, r.commandPool, nil)

	for _, fb := range r.framebuffers {
		vk.DestroyFramebuffer(r.Device.logical, fb, nil)
	}
}

// frames ---

type FrameData struct {
	CommandBuffer vk.CommandBuffer

	ImageAvailableSemaphore  vk.Semaphore
	RenderFinishedSemaphores []vk.Semaphore
	InFlightFence            vk.Fence
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

	f.RenderFinishedSemaphores = make([]vk.Semaphore, len(r.SwapChain.images))
	for i := range len(r.SwapChain.images) {
		renderSemaphore, err := vk.CreateSemaphore(r.Device.logical, &semaphoreInfo, nil)
		if err != nil {
			return err
		}
		f.RenderFinishedSemaphores[i] = renderSemaphore
	}

	return nil
}

func (f *FrameData) Destroy(logicalDev vk.Device) {
	vk.DestroySemaphore(logicalDev, f.ImageAvailableSemaphore, nil)
	vk.DestroyFence(logicalDev, f.InFlightFence, nil)

	for _, sem := range f.RenderFinishedSemaphores {
		vk.DestroySemaphore(logicalDev, sem, nil)
	}
}
