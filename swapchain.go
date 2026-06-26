package yellowstone

import (
	"fmt"
	"log"

	"github.com/bbredesen/go-vk"
)

type Swapchain struct {
	Device *VulkanDevice
	VSync  bool

	handle      vk.SwapchainKHR
	images      []vk.Image
	imageViews  []vk.ImageView
	imageFormat vk.Format
	extent      vk.Extent2D
}

func (s *Swapchain) SetupSwapchain(window *Window) error {
	dets := QuerySwapChainSupport(s.Device.physical, window.surface)

	format := s.chooseSwapSurfaceFormat(dets.formats)
	presentMode := s.chooseSwapPresentMode(dets.presentModes, s.VSync)
	extent := s.chooseSwapExtent(dets.caps, window)

	imageCount := dets.caps.MinImageCount + 1
	if dets.caps.MaxImageCount > 0 && imageCount > dets.caps.MaxImageCount {
		imageCount = dets.caps.MaxImageCount
	}

	createInfo := vk.SwapchainCreateInfoKHR{
		Surface:          window.surface,
		MinImageCount:    imageCount,
		ImageFormat:      format.Format,
		ImageColorSpace:  format.ColorSpace,
		ImageExtent:      extent,
		ImageArrayLayers: 1,
		ImageUsage:       vk.IMAGE_USAGE_COLOR_ATTACHMENT_BIT,
	}

	queueFamilyIndices := []uint32{
		uint32(s.Device.queueIndices.graphicsFamily.Index),
		uint32(s.Device.queueIndices.presentFamily.Index),
	}

	if s.Device.queueIndices.graphicsFamily != s.Device.queueIndices.presentFamily {
		createInfo.ImageSharingMode = vk.SHARING_MODE_CONCURRENT
		createInfo.PQueueFamilyIndices = queueFamilyIndices
	} else {
		createInfo.ImageSharingMode = vk.SHARING_MODE_EXCLUSIVE
	}

	createInfo.PreTransform = dets.caps.CurrentTransform
	createInfo.CompositeAlpha = vk.COMPOSITE_ALPHA_OPAQUE_BIT_KHR
	createInfo.PresentMode = presentMode
	createInfo.Clipped = true
	createInfo.OldSwapchain = vk.SwapchainKHR(vk.NULL_HANDLE)

	swpchain, err := vk.CreateSwapchainKHR(s.Device.logical, &createInfo, nil)
	if err != nil {
		return fmt.Errorf("vk.CreateSwapchainKHR() failed: %v", err)
	}

	imgs, err := vk.GetSwapchainImagesKHR(s.Device.logical, swpchain)
	if err != nil {
		return fmt.Errorf("vk.GetSwapchainImagesKHR() failed: %v", err)
	}

	s.extent = extent
	s.imageFormat = format.Format

	s.images = imgs
	s.handle = swpchain

	if err := s.createImageViews(s.Device.logical); err != nil {
		return fmt.Errorf("Failed to create ImageViews: %w", err)
	}

	return nil
}

func (s *Swapchain) createImageViews(logicalDev vk.Device) error {
	s.imageViews = make([]vk.ImageView, len(s.images))
	for i := range len(s.images) {
		createInfo := vk.ImageViewCreateInfo{
			Image:    s.images[i],
			ViewType: vk.IMAGE_VIEW_TYPE_2D,
			Format:   s.imageFormat,
			Components: vk.ComponentMapping{
				R: vk.COMPONENT_SWIZZLE_IDENTITY,
				G: vk.COMPONENT_SWIZZLE_IDENTITY,
				B: vk.COMPONENT_SWIZZLE_IDENTITY,
				A: vk.COMPONENT_SWIZZLE_IDENTITY,
			},
			SubresourceRange: vk.ImageSubresourceRange{
				AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
				BaseMipLevel:   0,
				LevelCount:     1,
				BaseArrayLayer: 0,
				LayerCount:     1,
			},
		}

		view, err := vk.CreateImageView(logicalDev, &createInfo, nil)
		if err != nil {
			return err
		}

		s.imageViews[i] = view
	}

	return nil
}

func (s *Swapchain) chooseSwapSurfaceFormat(availableFormats []vk.SurfaceFormatKHR) vk.SurfaceFormatKHR {
	for _, f := range availableFormats {
		if f.Format == vk.FORMAT_B8G8R8_SRGB && f.ColorSpace == vk.COLOR_SPACE_SRGB_NONLINEAR_KHR {
			return f
		}
	}

	return availableFormats[0]
}

func (s *Swapchain) chooseSwapPresentMode(availablePresentModes []vk.PresentModeKHR, preferVSync bool) vk.PresentModeKHR {
	if preferVSync {
		return vk.PRESENT_MODE_FIFO_KHR
	}

	for _, m := range availablePresentModes {
		if m == vk.PRESENT_MODE_MAILBOX_KHR {
			return m
		}
	}

	return vk.PRESENT_MODE_FIFO_KHR
}

func (s *Swapchain) chooseSwapExtent(caps vk.SurfaceCapabilitiesKHR, window *Window) vk.Extent2D {
	if caps.CurrentExtent.Width != 4294967295 {
		return caps.CurrentExtent
	}

	w, h := window.glfwWindow.GetFramebufferSize()
	ext := vk.Extent2D{
		Width:  uint32(w),
		Height: uint32(h),
	}

	ext.Width = max(min(ext.Width, caps.MaxImageExtent.Width), caps.MinImageExtent.Width)
	ext.Height = max(min(ext.Height, caps.MaxImageExtent.Height), caps.MinImageExtent.Height)

	return ext
}

func (s *Swapchain) Destroy() {
	for _, view := range s.imageViews {
		vk.DestroyImageView(s.Device.logical, view, nil)
	}
	s.imageViews = nil

	if s.handle != vk.SwapchainKHR(vk.NULL_HANDLE) {
		vk.DestroySwapchainKHR(s.Device.logical, s.handle, nil)
		s.handle = vk.SwapchainKHR(vk.NULL_HANDLE)
	}
}

// utility ---

type SwapChainSupportDetails struct {
	caps         vk.SurfaceCapabilitiesKHR
	formats      []vk.SurfaceFormatKHR
	presentModes []vk.PresentModeKHR
}

func QuerySwapChainSupport(dev vk.PhysicalDevice, surface vk.SurfaceKHR) SwapChainSupportDetails {
	caps, err := vk.GetPhysicalDeviceSurfaceCapabilitiesKHR(dev, surface)
	if err != nil {
		log.Printf("QuerySwapChainSupport: vk.GetPhysicalDeviceSurfaceCapabilitiesKHR(): error: %v", err)
		return SwapChainSupportDetails{}
	}

	formats, err := vk.GetPhysicalDeviceSurfaceFormatsKHR(dev, surface)
	if err != nil {
		log.Printf("QuerySwapChainSupport: vk.GetPhysicalDeviceSurfaceFormatsKHR(): error: %v", err)
		return SwapChainSupportDetails{}
	}

	presentModes, err := vk.GetPhysicalDeviceSurfacePresentModesKHR(dev, surface)
	if err != nil {
		log.Printf("QuerySwapChainSupport: vk.GetPhysicalDeviceSurfacePresentModesKHR(): error: %v", err)
		return SwapChainSupportDetails{}
	}

	return SwapChainSupportDetails{
		caps:         caps,
		formats:      formats,
		presentModes: presentModes,
	}
}
