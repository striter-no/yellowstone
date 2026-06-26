package yellowstone

import (
	"log"

	"github.com/bbredesen/go-vk"
)

type VkIndexType struct {
	Index    int
	HasValue bool
}

type QueueFamilyIndices struct {
	graphicsFamily VkIndexType
	presentFamily  VkIndexType
}

func FindQueueFamilies(dev vk.PhysicalDevice, surface vk.SurfaceKHR) QueueFamilyIndices {
	indices := QueueFamilyIndices{}

	qfp := vk.GetPhysicalDeviceQueueFamilyProperties(dev)

	for i, p := range qfp {
		if (p.QueueFlags & vk.QUEUE_GRAPHICS_BIT) != 0 {
			indices.graphicsFamily.Index = i
			indices.graphicsFamily.HasValue = true
		}

		sup, err := vk.GetPhysicalDeviceSurfaceSupportKHR(dev, uint32(i), surface)
		if err != nil {
			log.Printf("FindQueueFamilies: vk.GetPhysicalDeviceSurfaceSupportKHR(): %v", err)
		}

		if err == nil && sup {
			indices.presentFamily.Index = i
			indices.presentFamily.HasValue = true
		}

		if indices.IsComplete() {
			break
		}
	}

	return indices
}

func (q *QueueFamilyIndices) IsComplete() bool {
	return q.graphicsFamily.HasValue && q.presentFamily.HasValue
}
