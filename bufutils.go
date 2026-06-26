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
	allocInfo := vk.CommandBufferAllocateInfo{
		Level:              vk.COMMAND_BUFFER_LEVEL_PRIMARY,
		CommandPool:        commandPool,
		CommandBufferCount: 1,
	}

	cbs, err := vk.AllocateCommandBuffers(vd.logical, &allocInfo)
	if err != nil {
		return err
	}

	beginInfo := vk.CommandBufferBeginInfo{
		Flags: vk.COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT,
	}

	if err := vk.BeginCommandBuffer(cbs[0], &beginInfo); err != nil {
		return err
	}

	copyRegion := vk.BufferCopy{
		SrcOffset: 0,
		DstOffset: 0,
		Size:      size,
	}

	vk.CmdCopyBuffer(cbs[0], src, dst, []vk.BufferCopy{copyRegion})
	vk.EndCommandBuffer(cbs[0])

	submitInfo := vk.SubmitInfo{
		PCommandBuffers: cbs,
	}

	if err := vk.QueueSubmit(vd.graphicsQueue, []vk.SubmitInfo{submitInfo}, vk.Fence(vk.NULL_HANDLE)); err != nil {
		return err
	}

	if err := vk.QueueWaitIdle(vd.graphicsQueue); err != nil {
		return err
	}

	vk.FreeCommandBuffers(vd.logical, commandPool, cbs)
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
