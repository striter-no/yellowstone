package yellowstone

import (
	"fmt"
	"unsafe"

	"github.com/bbredesen/go-vk"
)

type VertexBuffer struct {
	data   []Vertex
	buffer vk.Buffer
	memory vk.DeviceMemory

	dev *VulkanDevice
}

func NewVertexBuffer(data []Vertex, device *VulkanDevice) (*VertexBuffer, error) {
	dataBytesN := uint64(unsafe.Sizeof(data[0])) * uint64(len(data))
	bufferInfo := vk.BufferCreateInfo{
		Size:        vk.DeviceSize(dataBytesN),
		Usage:       vk.BUFFER_USAGE_VERTEX_BUFFER_BIT,
		SharingMode: vk.SHARING_MODE_EXCLUSIVE,
	}

	buf, err := vk.CreateBuffer(device.logical, &bufferInfo, nil)
	if err != nil {
		return nil, fmt.Errorf("CreateBuffer failed: %w", err)
	}

	memRequirements := vk.GetBufferMemoryRequirements(device.logical, buf)
	memProps := vk.GetPhysicalDeviceMemoryProperties(device.physical)

	properties := vk.MEMORY_PROPERTY_HOST_VISIBLE_BIT | vk.MEMORY_PROPERTY_HOST_COHERENT_BIT

	memTypeIndex, err := findMemoryType(memRequirements.MemoryTypeBits, properties, memProps)
	if err != nil {
		vk.DestroyBuffer(device.logical, buf, nil)
		return nil, err
	}

	allocInfo := vk.MemoryAllocateInfo{
		AllocationSize:  memRequirements.Size,
		MemoryTypeIndex: memTypeIndex,
	}

	bufferMemory, err := vk.AllocateMemory(device.logical, &allocInfo, nil)
	if err != nil {
		vk.DestroyBuffer(device.logical, buf, nil)
		return nil, fmt.Errorf("AllocateMemory failed: %w", err)
	}

	if err := vk.BindBufferMemory(device.logical, buf, bufferMemory, 0); err != nil {
		vk.FreeMemory(device.logical, bufferMemory, nil)
		vk.DestroyBuffer(device.logical, buf, nil)
		return nil, fmt.Errorf("BindBufferMemory failed: %w", err)
	}

	pdata, err := vk.MapMemory(device.logical, bufferMemory, 0, vk.DeviceSize(dataBytesN), 0)
	if err != nil {
		return nil, fmt.Errorf("MapMemory failed: %w", err)
	}

	mappedMemory := unsafe.Slice(pdata, dataBytesN)
	vertexDataPtr := (*byte)(unsafe.Pointer(&data[0]))
	sourceData := unsafe.Slice(vertexDataPtr, dataBytesN)

	copy(mappedMemory, sourceData)

	vk.UnmapMemory(device.logical, bufferMemory)

	return &VertexBuffer{
		data:   data,
		buffer: buf,
		memory: bufferMemory,
		dev:    device,
	}, nil
}

func (vb *VertexBuffer) Destroy() {
	vk.DestroyBuffer(vb.dev.logical, vb.buffer, nil)
	vk.FreeMemory(vb.dev.logical, vb.memory, nil)
}

// -- utils

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
