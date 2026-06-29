package yellowstone

import (
	"fmt"
	"unsafe"

	"github.com/bbredesen/go-vk"
)

type IndexBuffer struct {
	indices []uint32
	buffer  vk.Buffer
	memory  vk.DeviceMemory

	dev *VulkanDevice
}

func NewIndexBuffer(data []uint32, renderer *Renderer) (*IndexBuffer, error) {
	dataBytesN := uint64(unsafe.Sizeof(data[0])) * uint64(len(data))

	stagingBuf, stagingBufMem, err := createBuffer(
		vk.DeviceSize(dataBytesN),
		vk.BUFFER_USAGE_TRANSFER_SRC_BIT,
		vk.MEMORY_PROPERTY_HOST_VISIBLE_BIT|vk.MEMORY_PROPERTY_HOST_COHERENT_BIT,
		renderer.Device,
	)
	if err != nil {
		return nil, fmt.Errorf("Failed to create buffer: %w", err)
	}

	pdata, err := vk.MapMemory(renderer.Device.logical, stagingBufMem, 0, vk.DeviceSize(dataBytesN), 0)
	if err != nil {
		return nil, fmt.Errorf("MapMemory failed: %w", err)
	}

	mappedMemory := unsafe.Slice(pdata, dataBytesN)
	indexDataPtr := (*byte)(unsafe.Pointer(&data[0]))
	sourceData := unsafe.Slice(indexDataPtr, dataBytesN)

	copy(mappedMemory, sourceData)

	vk.UnmapMemory(renderer.Device.logical, stagingBufMem)

	indexBuffer, indexBufMem, err := createBuffer(
		vk.DeviceSize(dataBytesN),
		vk.BUFFER_USAGE_TRANSFER_DST_BIT|vk.BUFFER_USAGE_INDEX_BUFFER_BIT,
		vk.MEMORY_PROPERTY_DEVICE_LOCAL_BIT,
		renderer.Device,
	)
	if err := copyBuffer(stagingBuf, indexBuffer, vk.DeviceSize(dataBytesN), renderer.Device, renderer.commandPool); err != nil {
		return nil, err
	}

	vk.DestroyBuffer(renderer.Device.logical, stagingBuf, nil)
	vk.FreeMemory(renderer.Device.logical, stagingBufMem, nil)

	return &IndexBuffer{
		indices: data,
		buffer:  indexBuffer,
		memory:  indexBufMem,
		dev:     renderer.Device,
	}, nil
}

func (ib *IndexBuffer) Destroy() {
	vk.DestroyBuffer(ib.dev.logical, ib.buffer, nil)
	vk.FreeMemory(ib.dev.logical, ib.memory, nil)
}
