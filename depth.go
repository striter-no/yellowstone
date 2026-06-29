package yellowstone

import (
	"errors"

	"github.com/bbredesen/go-vk"
)

func findSupportedFormat(candidates []vk.Format, tiling vk.ImageTiling, feats vk.FormatFeatureFlags, dev *VulkanDevice) (vk.Format, error) {
	for _, format := range candidates {
		props := vk.GetPhysicalDeviceFormatProperties(dev.physical, format)

		if tiling == vk.IMAGE_TILING_LINEAR && (props.LinearTilingFeatures&feats) == feats {
			return format, nil
		} else if tiling == vk.IMAGE_TILING_OPTIMAL && (props.OptimalTilingFeatures&feats) == feats {
			return format, nil
		}
	}

	return 0, errors.New("Failed to find supported format")
}

func findDepthFormat(dev *VulkanDevice) (vk.Format, error) {
	return findSupportedFormat(
		[]vk.Format{vk.FORMAT_D32_SFLOAT, vk.FORMAT_D32_SFLOAT_S8_UINT, vk.FORMAT_D24_UNORM_S8_UINT},
		vk.IMAGE_TILING_OPTIMAL,
		vk.FORMAT_FEATURE_DEPTH_STENCIL_ATTACHMENT_BIT,
		dev,
	)
}

func hasStencilComponent(format vk.Format) bool {
	return format == vk.FORMAT_D32_SFLOAT_S8_UINT || format == vk.FORMAT_D24_UNORM_S8_UINT
}
