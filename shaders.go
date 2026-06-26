package yellowstone

import (
	"os"
	"unsafe"

	"github.com/bbredesen/go-vk"
)

type VulkanShader struct {
	byteCode []byte
}

func LoadSPVShader(spvPath string) (VulkanShader, error) {
	cont, err := os.ReadFile(spvPath)
	if err != nil {
		return VulkanShader{}, err
	}

	return VulkanShader{
		byteCode: cont,
	}, nil
}

func (s *VulkanShader) CreateShaderModule(dev vk.Device) (vk.ShaderModule, error) {
	createInfo := vk.ShaderModuleCreateInfo{
		PCode:    (*uint32)(unsafe.Pointer(&s.byteCode[0])),
		CodeSize: uintptr(len(s.byteCode)),
	}

	mod, err := vk.CreateShaderModule(dev, &createInfo, nil)
	if err != nil {
		return 0, err
	}

	return mod, nil
}
