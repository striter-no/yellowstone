package yellowstone

import (
	"errors"
	"fmt"
	"image"
	"image/draw"
	"os"
	"unsafe"

	_ "image/jpeg"
	_ "image/png"

	"github.com/bbredesen/go-vk"
)

type Texture struct {
	View      vk.ImageView
	VkTexture vk.Image

	textureMem vk.DeviceMemory

	dev *VulkanDevice
}

func NewTextureFromFile(path string, renderer *Renderer) (*Texture, error) {
	pixels, twidth, theight, err := getImageFromFilePath(path)
	if err != nil {
		return nil, fmt.Errorf("Failed to get image from path (%s): %w", path, err)
	}

	imgSize := vk.DeviceSize(len(pixels))
	stagingBuf, stagingMem, err := createBuffer(
		imgSize,
		vk.BUFFER_USAGE_TRANSFER_SRC_BIT,
		vk.MEMORY_PROPERTY_HOST_VISIBLE_BIT|vk.MEMORY_PROPERTY_HOST_COHERENT_BIT,
		renderer.Device,
	)
	if err != nil {
		return nil, err
	}

	pdata, err := vk.MapMemory(renderer.Device.logical, stagingMem, 0, vk.DeviceSize(len(pixels)), 0)
	if err != nil {
		return nil, fmt.Errorf("MapMemory failed: %w", err)
	}

	mappedMemory := unsafe.Slice(pdata, len(pixels))
	pixelsDataPtr := (*byte)(unsafe.Pointer(&pixels[0]))
	sourceData := unsafe.Slice(pixelsDataPtr, len(pixels))

	copy(mappedMemory, sourceData)

	vk.UnmapMemory(renderer.Device.logical, stagingMem)

	img, mem, err := createImage(
		uint(twidth), uint(theight),
		vk.FORMAT_R8G8B8A8_SRGB,
		vk.IMAGE_TILING_OPTIMAL,
		vk.IMAGE_USAGE_TRANSFER_DST_BIT|vk.IMAGE_USAGE_SAMPLED_BIT,
		vk.MEMORY_PROPERTY_DEVICE_LOCAL_BIT,
		renderer.Device,
	)

	if err != nil {
		return nil, fmt.Errorf("createImage failed: %w", err)
	}

	/*
		transitionImageLayout(textureImage, VK_FORMAT_R8G8B8A8_SRGB, VK_IMAGE_LAYOUT_UNDEFINED, VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL);
			copyBufferToImage(stagingBuffer, textureImage, static_cast<uint32_t>(texWidth), static_cast<uint32_t>(texHeight));
		transitionImageLayout(textureImage, VK_FORMAT_R8G8B8A8_SRGB, VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL, VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL);
	*/

	if err := transitionImageLayout(
		img,
		vk.FORMAT_R8G8B8A8_SRGB,
		vk.IMAGE_LAYOUT_UNDEFINED,
		vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
		renderer.commandPool, renderer.Device,
	); err != nil {
		return nil, err
	}

	if err := copyBufferToImage(
		stagingBuf, img, twidth, theight,
		renderer.commandPool, renderer.Device,
	); err != nil {
		return nil, err
	}

	if err := transitionImageLayout(
		img,
		vk.FORMAT_R8G8B8A8_SRGB,
		vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
		vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
		renderer.commandPool, renderer.Device,
	); err != nil {
		return nil, err
	}

	vk.DestroyBuffer(renderer.Device.logical, stagingBuf, nil)
	vk.FreeMemory(renderer.Device.logical, stagingMem, nil)

	view, err := createImageView(img, vk.FORMAT_R8G8B8A8_SRGB, vk.IMAGE_ASPECT_COLOR_BIT, renderer.Device)
	if err != nil {
		return nil, err
	}

	return &Texture{
		VkTexture:  img,
		View:       view,
		textureMem: mem,
		dev:        renderer.Device,
	}, nil
}

func (t *Texture) Destroy() {
	vk.DestroyImageView(t.dev.logical, t.View, nil)
	vk.DestroyImage(t.dev.logical, t.VkTexture, nil)
	vk.FreeMemory(t.dev.logical, t.textureMem, nil)
}

func transitionImageLayout(image vk.Image, format vk.Format, oldLayout, newLayout vk.ImageLayout, cpool vk.CommandPool, dev *VulkanDevice) error {
	cb, err := beginSingleTimeCommands(cpool, dev)
	if err != nil {
		return err
	}

	barrier := vk.ImageMemoryBarrier{
		OldLayout:           oldLayout,
		NewLayout:           newLayout,
		SrcQueueFamilyIndex: vk.QUEUE_FAMILY_IGNORED,
		DstQueueFamilyIndex: vk.QUEUE_FAMILY_IGNORED,
		Image:               image,
		SubresourceRange: vk.ImageSubresourceRange{
			AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
			BaseMipLevel:   0,
			LevelCount:     1,
			BaseArrayLayer: 0,
			LayerCount:     1,
		},
	}

	if newLayout == vk.IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL {
		barrier.SubresourceRange.AspectMask = vk.IMAGE_ASPECT_DEPTH_BIT

		if hasStencilComponent(format) {
			barrier.SubresourceRange.AspectMask |= vk.IMAGE_ASPECT_STENCIL_BIT
		}
	} else {
		barrier.SubresourceRange.AspectMask = vk.IMAGE_ASPECT_COLOR_BIT
	}

	var sourceStage, destinationStage vk.PipelineStageFlags
	if oldLayout == vk.IMAGE_LAYOUT_UNDEFINED && newLayout == vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL {
		barrier.SrcAccessMask = 0
		barrier.DstAccessMask = vk.ACCESS_TRANSFER_WRITE_BIT

		sourceStage = vk.PIPELINE_STAGE_TOP_OF_PIPE_BIT
		destinationStage = vk.PIPELINE_STAGE_TRANSFER_BIT
	} else if oldLayout == vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL && newLayout == vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL {
		barrier.SrcAccessMask = vk.ACCESS_TRANSFER_WRITE_BIT
		barrier.DstAccessMask = vk.ACCESS_SHADER_READ_BIT

		sourceStage = vk.PIPELINE_STAGE_TRANSFER_BIT
		destinationStage = vk.PIPELINE_STAGE_FRAGMENT_SHADER_BIT
	} else if oldLayout == vk.IMAGE_LAYOUT_UNDEFINED && newLayout == vk.IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL {
		barrier.SrcAccessMask = 0
		barrier.DstAccessMask = vk.ACCESS_DEPTH_STENCIL_ATTACHMENT_READ_BIT | vk.ACCESS_DEPTH_STENCIL_ATTACHMENT_WRITE_BIT

		sourceStage = vk.PIPELINE_STAGE_TOP_OF_PIPE_BIT
		destinationStage = vk.PIPELINE_STAGE_EARLY_FRAGMENT_TESTS_BIT
	} else {
		return errors.New("Unsupported layout transition")
	}

	vk.CmdPipelineBarrier(
		cb,
		sourceStage, destinationStage,
		0,
		[]vk.MemoryBarrier{},
		[]vk.BufferMemoryBarrier{},
		[]vk.ImageMemoryBarrier{barrier},
	)

	return endSingleTimeCommands(cb, cpool, dev)
}

func copyBufferToImage(buffer vk.Buffer, image vk.Image, width, height int, cpool vk.CommandPool, dev *VulkanDevice) error {
	cb, err := beginSingleTimeCommands(cpool, dev)
	if err != nil {
		return err
	}

	region := vk.BufferImageCopy{
		BufferOffset:      0,
		BufferRowLength:   0,
		BufferImageHeight: 0,

		ImageSubresource: vk.ImageSubresourceLayers{
			AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
			MipLevel:       0,
			BaseArrayLayer: 0,
			LayerCount:     1,
		},

		ImageOffset: vk.Offset3D{X: 0, Y: 0, Z: 0},
		ImageExtent: vk.Extent3D{
			Width: uint32(width), Height: uint32(height), Depth: 1,
		},
	}

	vk.CmdCopyBufferToImage(cb, buffer, image, vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL, []vk.BufferImageCopy{region})

	return endSingleTimeCommands(cb, cpool, dev)
}

func createImageView(image vk.Image, format vk.Format, aspectFlags vk.ImageAspectFlags, dev *VulkanDevice) (vk.ImageView, error) {
	viewInfo := vk.ImageViewCreateInfo{
		Image:    image,
		Format:   format,
		ViewType: vk.IMAGE_VIEW_TYPE_2D,
		SubresourceRange: vk.ImageSubresourceRange{
			AspectMask:     aspectFlags,
			BaseMipLevel:   0,
			LevelCount:     1,
			BaseArrayLayer: 0,
			LayerCount:     1,
		},
	}

	imgView, err := vk.CreateImageView(dev.logical, &viewInfo, nil)
	if err != nil {
		return 0, err
	}

	return imgView, nil
}

func createImage(
	width, height uint,
	format vk.Format,
	tiling vk.ImageTiling,
	usage vk.ImageUsageFlags,
	props vk.MemoryPropertyFlags,

	dev *VulkanDevice,
) (vk.Image, vk.DeviceMemory, error) {
	imageInfo := vk.ImageCreateInfo{
		ImageType: vk.IMAGE_TYPE_2D,
		Extent: vk.Extent3D{
			Width:  uint32(width),
			Height: uint32(height),
			Depth:  1,
		},
		MipLevels:     1,
		ArrayLayers:   1,
		Format:        format,
		Tiling:        tiling,
		InitialLayout: vk.IMAGE_LAYOUT_UNDEFINED,
		Usage:         usage,
		SharingMode:   vk.SHARING_MODE_EXCLUSIVE,
		Samples:       vk.SAMPLE_COUNT_1_BIT,
	}

	vkImg, err := vk.CreateImage(dev.logical, &imageInfo, nil)
	if err != nil {
		return vk.Image(0), vk.DeviceMemory(0), err
	}

	memReqs := vk.GetImageMemoryRequirements(dev.logical, vkImg)
	memProps := vk.GetPhysicalDeviceMemoryProperties(dev.physical)

	typeIndex, err := findMemoryType(memReqs.MemoryTypeBits, props, memProps)
	if err != nil {
		return vk.Image(0), vk.DeviceMemory(0), err
	}

	allocInfo := vk.MemoryAllocateInfo{
		AllocationSize:  memReqs.Size,
		MemoryTypeIndex: typeIndex,
	}

	imgMem, err := vk.AllocateMemory(dev.logical, &allocInfo, nil)
	if err != nil {
		return vk.Image(0), vk.DeviceMemory(0), err
	}

	if err := vk.BindImageMemory(dev.logical, vkImg, imgMem, 0); err != nil {
		return vk.Image(0), vk.DeviceMemory(0), err
	}

	return vkImg, imgMem, nil
}

func getImageFromFilePath(filePath string) ([]uint8, int, int, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, 0, 0, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, 0, 0, err
	}

	if rgba, ok := img.(*image.RGBA); ok {
		return rgba.Pix, img.Bounds().Dx(), img.Bounds().Dy(), nil
	}

	b := img.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))

	draw.Draw(dst, dst.Bounds(), img, b.Min, draw.Src)
	return dst.Pix, b.Dx(), b.Dy(), nil
}
