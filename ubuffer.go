package yellowstone

import (
	"unsafe"

	"github.com/bbredesen/go-vk"
)

type UniformBuffer struct {
	data   *byte
	buffer vk.Buffer
	memory vk.DeviceMemory

	dev *VulkanDevice
}

func NewUniformBuffer(renderer *Renderer) (*UniformBuffer, error) {

	bufferSize := unsafe.Sizeof(UniformBufferObject{})

	buf, mem, err := createBuffer(
		vk.DeviceSize(bufferSize),
		vk.BUFFER_USAGE_UNIFORM_BUFFER_BIT,
		vk.MEMORY_PROPERTY_HOST_VISIBLE_BIT|vk.MEMORY_PROPERTY_HOST_COHERENT_BIT,
		renderer.Device,
	)
	if err != nil {
		return nil, err
	}

	pdata, err := vk.MapMemory(renderer.Device.logical, mem, 0, vk.DeviceSize(bufferSize), 0)
	if err != nil {
		return nil, err
	}

	return &UniformBuffer{
		dev:    renderer.Device,
		data:   pdata,
		buffer: buf,
		memory: mem,
	}, nil
}

func (ub *UniformBuffer) Fill(ubo *UniformBufferObject) {
	bytes := getUBOBytes(ubo)
	n := unsafe.Sizeof(UniformBufferObject{})

	mappedMemory := unsafe.Slice(ub.data, n)
	indexDataPtr := (*byte)(unsafe.Pointer(&bytes[0]))
	sourceData := unsafe.Slice(indexDataPtr, n)

	copy(mappedMemory, sourceData)
}

func (ub *UniformBuffer) Destroy() {
	vk.DestroyBuffer(ub.dev.logical, ub.buffer, nil)
	vk.FreeMemory(ub.dev.logical, ub.memory, nil)
}
