package yellowstone

import (
	"fmt"

	"github.com/bbredesen/go-vk"
)

func findMemoryType(
	typeFilter uint32,
	properties vk.MemoryPropertyFlags,
	memProps vk.PhysicalDeviceMemoryProperties,
) (uint32, error) {
	for i := uint32(0); i < memProps.MemoryTypeCount; i++ {
		if (typeFilter & (1 << i)) != 0 {
			if (memProps.MemoryTypes[i].PropertyFlags & properties) == properties {
				return i, nil
			}
		}
	}

	return 0, fmt.Errorf("failed to find suitable memory type")
}

func copyBuffer(
	src, dst vk.Buffer, size vk.DeviceSize, vd *VulkanDevice, commandPool vk.CommandPool,
) error {
	cb, err := beginSingleTimeCommands(commandPool, vd)
	if err != nil {
		return err
	}

	copyRegion := vk.BufferCopy{
		SrcOffset: 0,
		DstOffset: 0,
		Size:      size,
	}

	vk.CmdCopyBuffer(cb, src, dst, []vk.BufferCopy{copyRegion})

	if err := endSingleTimeCommands(cb, commandPool, vd); err != nil {
		return err
	}

	return nil
}

func createBuffer(
	size vk.DeviceSize,
	usage vk.BufferUsageFlags,
	props vk.MemoryPropertyFlags,

	device *VulkanDevice,
) (buffer vk.Buffer, bufmem vk.DeviceMemory, err error) {
	bufferInfo := vk.BufferCreateInfo{
		Size:        size,
		Usage:       usage,
		SharingMode: vk.SHARING_MODE_EXCLUSIVE,
	}

	buffer, err = vk.CreateBuffer(device.logical, &bufferInfo, nil)
	if err != nil {
		return buffer, bufmem, fmt.Errorf("CreateBuffer failed: %w", err)
	}

	memRequirements := vk.GetBufferMemoryRequirements(device.logical, buffer)
	memProps := vk.GetPhysicalDeviceMemoryProperties(device.physical)

	memTypeIndex, err := findMemoryType(memRequirements.MemoryTypeBits, props, memProps)
	if err != nil {
		vk.DestroyBuffer(device.logical, buffer, nil)
		return buffer, bufmem, err
	}

	allocInfo := vk.MemoryAllocateInfo{
		AllocationSize:  memRequirements.Size,
		MemoryTypeIndex: memTypeIndex,
	}

	bufmem, err = vk.AllocateMemory(device.logical, &allocInfo, nil)
	if err != nil {
		vk.DestroyBuffer(device.logical, buffer, nil)
		return buffer, bufmem, fmt.Errorf("AllocateMemory failed: %w", err)
	}

	if err := vk.BindBufferMemory(device.logical, buffer, bufmem, 0); err != nil {
		vk.FreeMemory(device.logical, bufmem, nil)
		vk.DestroyBuffer(device.logical, buffer, nil)
		return buffer, bufmem, fmt.Errorf("BindBufferMemory failed: %w", err)
	}

	return buffer, bufmem, nil
}

func beginSingleTimeCommands(commandPool vk.CommandPool, dev *VulkanDevice) (vk.CommandBuffer, error) {
	allocInfo := vk.CommandBufferAllocateInfo{
		Level:              vk.COMMAND_BUFFER_LEVEL_PRIMARY,
		CommandPool:        commandPool,
		CommandBufferCount: 1,
	}

	cbs, err := vk.AllocateCommandBuffers(dev.logical, &allocInfo)
	if err != nil {
		return 0, err
	}

	beginInfo := vk.CommandBufferBeginInfo{
		Flags: vk.COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT,
	}

	if err := vk.BeginCommandBuffer(cbs[0], &beginInfo); err != nil {
		return 0, err
	}

	return cbs[0], nil
}

func endSingleTimeCommands(cb vk.CommandBuffer, cpool vk.CommandPool, dev *VulkanDevice) error {
	if err := vk.EndCommandBuffer(cb); err != nil {
		return err
	}

	sinfo := vk.SubmitInfo{
		PCommandBuffers: []vk.CommandBuffer{cb},
	}

	if err := vk.QueueSubmit(dev.graphicsQueue, []vk.SubmitInfo{sinfo}, 0); err != nil {
		return err
	}

	if err := vk.QueueWaitIdle(dev.graphicsQueue); err != nil {
		return err
	}

	vk.FreeCommandBuffers(dev.logical, cpool, []vk.CommandBuffer{cb})
	return nil
}
