package yellowstone

import "github.com/bbredesen/go-vk"

type TextureSampler struct {
	Device *VulkanDevice

	sampler vk.Sampler
}

func (s *TextureSampler) SetupTextureSampler() error {
	pprops := vk.GetPhysicalDeviceProperties(s.Device.physical)

	samplerInfo := vk.SamplerCreateInfo{
		MagFilter:               vk.FILTER_LINEAR,
		MinFilter:               vk.FILTER_LINEAR,
		AddressModeU:            vk.SAMPLER_ADDRESS_MODE_REPEAT,
		AddressModeV:            vk.SAMPLER_ADDRESS_MODE_REPEAT,
		AddressModeW:            vk.SAMPLER_ADDRESS_MODE_REPEAT,
		AnisotropyEnable:        true,
		MaxAnisotropy:           pprops.Limits.MaxSamplerAnisotropy,
		UnnormalizedCoordinates: false,
		BorderColor:             vk.BORDER_COLOR_INT_OPAQUE_BLACK,
		CompareEnable:           false,
		CompareOp:               vk.COMPARE_OP_ALWAYS,
		MipmapMode:              vk.SAMPLER_MIPMAP_MODE_LINEAR,
		MipLodBias:              0,
		MinLod:                  0,
		MaxLod:                  0,
	}

	sampler, err := vk.CreateSampler(s.Device.logical, &samplerInfo, nil)
	if err != nil {
		return err
	}

	s.sampler = sampler
	return nil
}

func (s *TextureSampler) Destroy() {
	vk.DestroySampler(s.Device.logical, s.sampler, nil)
}
