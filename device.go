package yellowstone

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"unsafe"

	"github.com/bbredesen/go-vk"
	"github.com/striter-no/yellowstone/internal"
)

type AppInfo struct {
	Name, EngineName                         string
	AppVersion, EngineVersion, VulkanVersion string
}

type VulkanDevice struct {
	Window           *Window
	EnableValidation bool

	// -- private
	instance vk.Instance
	physical vk.PhysicalDevice
	logical  vk.Device

	graphicsQueue vk.Queue
	presentQueue  vk.Queue
	queueIndices  QueueFamilyIndices
}

func (d *VulkanDevice) SetupVulkanDevice(
	userAppInfo AppInfo,
) error {
	if exists, err := d.checkValidationLayerSupport(); (err != nil || !exists) && d.EnableValidation {
		return fmt.Errorf("No validation support, error: %w", err)
	}

	parsedAppVersion, err := parseVersion(userAppInfo.AppVersion)
	if err != nil {
		return err
	}

	parsedEngineVersion, err := parseVersion(userAppInfo.EngineVersion)
	if err != nil {
		return err
	}

	parsedVkVersion, err := parseVersion(userAppInfo.VulkanVersion)
	if err != nil {
		return err
	}

	appInfo := vk.ApplicationInfo{
		PApplicationName:   userAppInfo.Name,
		PEngineName:        userAppInfo.EngineName,
		ApplicationVersion: vk.MAKE_VERSION(parsedAppVersion[0], parsedAppVersion[1], parsedAppVersion[2]),
		EngineVersion:      vk.MAKE_VERSION(parsedEngineVersion[0], parsedEngineVersion[1], parsedEngineVersion[2]),
		ApiVersion:         vk.MAKE_VERSION(parsedVkVersion[0], parsedVkVersion[1], parsedVkVersion[2]),
	}

	layerNames := make([]string, 0)
	if d.EnableValidation {
		layerNames = append(layerNames, "VK_LAYER_KHRONOS_validation")
	}

	glfwExtensions := d.Window.glfwWindow.GetRequiredInstanceExtensions()

	if d.EnableValidation {
		glfwExtensions = append(glfwExtensions, vk.EXT_DEBUG_UTILS_EXTENSION_NAME)
	}

	icInfo := vk.InstanceCreateInfo{
		PApplicationInfo:        &appInfo,
		PpEnabledExtensionNames: glfwExtensions,
		PpEnabledLayerNames:     layerNames,
	}

	// log.Printf("Before CreateInstance: icInfo=%+v", icInfo)
	instance, err := vk.CreateInstance(&icInfo, nil)
	// log.Printf("After CreateInstance: instance=%#x err=%v (type=%T)", uint64(instance), err, err)
	if err != vk.SUCCESS {
		return fmt.Errorf("vk.CreateInstance() failed: %w", err)
	}
	d.instance = instance

	devs, err := vk.EnumeratePhysicalDevices(d.instance)
	log.Printf("First EnumeratePhysicalDevices: count=%d err=%v", len(devs), err)

	if err := d.Window.createSurface(d.instance); err != nil {
		return fmt.Errorf("Failed to create surface")
	}

	extensions := []string{vk.KHR_SWAPCHAIN_EXTENSION_NAME}
	if err := d.pickPhysicalDevice(extensions, d.Window.surface); err != nil {
		return fmt.Errorf("Failed to pick physical device: %w", err)
	}

	if err := d.createLogicalDevice(extensions, d.Window.surface); err != nil {
		return fmt.Errorf("Failed to create logical device: %w", err)
	}

	return nil
}

func (d *VulkanDevice) createLogicalDevice(neededExtensions []string, surface vk.SurfaceKHR) error {
	layerNames := make([]string, 0)
	if d.EnableValidation {
		layerNames = append(layerNames, "VK_LAYER_KHRONOS_validation")
	}

	indices := FindQueueFamilies(d.physical, surface)
	uniqueQueueFamilies := internal.MakeUniqueSlice([]int{
		indices.graphicsFamily.Index,
		indices.presentFamily.Index,
	})

	queueCreateInfos := make([]vk.DeviceQueueCreateInfo, 0)
	queuePriority := float32(1.0)

	for _, qFamily := range uniqueQueueFamilies {
		queueCreateInfo := vk.DeviceQueueCreateInfo{
			QueueFamilyIndex: uint32(qFamily),
			PQueuePriorities: []float32{queuePriority},
		}

		queueCreateInfos = append(queueCreateInfos, queueCreateInfo)
	}

	deviceFeatures := vk.PhysicalDeviceFeatures{
		SamplerAnisotropy: true,
	}

	features12 := vk.PhysicalDeviceVulkan12Features{
		ScalarBlockLayout: true,
	}

	features12Ptr := features12.Vulkanize()
	createInfo := vk.DeviceCreateInfo{
		PQueueCreateInfos:       queueCreateInfos,
		PEnabledFeatures:        &deviceFeatures,
		PpEnabledLayerNames:     layerNames,
		PpEnabledExtensionNames: neededExtensions,
		PNext:                   unsafe.Pointer(features12Ptr),
	}

	dev, err := vk.CreateDevice(d.physical, &createInfo, nil)
	if err != nil {
		return err
	}
	d.logical = dev

	d.graphicsQueue = vk.GetDeviceQueue(dev, uint32(indices.graphicsFamily.Index), 0)
	d.presentQueue = vk.GetDeviceQueue(dev, uint32(indices.presentFamily.Index), 0)
	d.queueIndices = indices
	return nil
}

func (d *VulkanDevice) pickPhysicalDevice(neededExtensions []string, surface vk.SurfaceKHR) error {
	devs, err := vk.EnumeratePhysicalDevices(d.instance)
	log.Printf("EnumeratePhysicalDevices: count=%d, err=%v", len(devs), err)
	for i, pd := range devs {
		props := vk.GetPhysicalDeviceProperties(pd)
		log.Printf("  [%d] %s", i, props.DeviceName)
	}
	if err != nil {
		return err
	}

	if len(devs) == 0 {
		return errors.New("No physical devices with Vulkan support")
	}

	var dev vk.PhysicalDevice
	for _, pd := range devs {
		if d.isDeviceSuitable(pd, neededExtensions, surface) {
			dev = pd
			break
		}
	}

	if dev == vk.PhysicalDevice(vk.NULL_HANDLE) {
		return errors.New("No physical devices with Vulkan support")
	}

	d.physical = dev
	return nil
}

func (d *VulkanDevice) isDeviceSuitable(dev vk.PhysicalDevice, neededExtensions []string, surface vk.SurfaceKHR) bool {
	indices := FindQueueFamilies(dev, surface)

	extensionsSupported := d.checkDeviceExtensionSupport(neededExtensions, dev)

	swapchainAdequate := false
	if extensionsSupported {
		dets := QuerySwapChainSupport(dev, surface)
		swapchainAdequate = len(dets.formats) != 0 && len(dets.presentModes) != 0
	}

	feats := vk.GetPhysicalDeviceFeatures(dev)

	return indices.IsComplete() && extensionsSupported && swapchainAdequate && feats.SamplerAnisotropy
}

func (d *VulkanDevice) checkDeviceExtensionSupport(neededDeviceExtensions []string, dev vk.PhysicalDevice) bool {

	props, err := vk.EnumerateDeviceExtensionProperties(dev, "")
	if err != nil {
		log.Printf("checkDeviceExtensionSupport: vk.EnumerateDeviceExtensionProperties(): error: %v", err)
		return false
	}

	c := 0
	for _, p := range props {
		if internal.CountInSlice(neededDeviceExtensions, p.ExtensionName) != 0 {
			c++
		}
	}

	return c == len(neededDeviceExtensions)
}

func (d *VulkanDevice) checkValidationLayerSupport() (bool, error) {
	props, err := vk.EnumerateInstanceLayerProperties()
	if err != nil {
		return false, err
	}

	for _, p := range props {
		if p.LayerName == "VK_LAYER_KHRONOS_validation" {
			return true, nil
		}
	}

	return false, nil
}

func (d *VulkanDevice) WaitIdle() error {
	return vk.DeviceWaitIdle(d.logical)
}

func (d *VulkanDevice) Destroy() {
	vk.DestroyDevice(d.logical, nil)
	vk.DestroySurfaceKHR(d.instance, d.Window.surface, nil)
	vk.DestroyInstance(d.instance, nil)
}

func parseVersion(v string) ([]uint32, error) {
	verStr := strings.Split(v, ".")
	nVer := [3]uint32{}

	if len(verStr) != 3 {
		return nil, fmt.Errorf("Malformed version formating")
	}

	for i, p := range verStr {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse app version")
		}

		nVer[i] = uint32(n)
	}

	return nVer[:], nil
}
