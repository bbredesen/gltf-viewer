package vkctx

import (
	"fmt"
	"unsafe"

	"github.com/bbredesen/go-vk"
	"golang.org/x/sys/windows"
)

func (ctx *Context) createInstance() {
	appInfo := vk.ApplicationInfo{
		PApplicationName:   "Context",
		ApplicationVersion: vk.MAKE_VERSION(1, 0, 0),
		EngineVersion:      vk.MAKE_VERSION(1, 0, 0),
		ApiVersion:         vk.MAKE_VERSION(1, 2, 0),
	}

	icInfo := vk.InstanceCreateInfo{
		PApplicationInfo:        &appInfo,
		PpEnabledExtensionNames: ctx.EnableInstanceExtensions,
		PpEnabledLayerNames:     ctx.EnableApiLayers,
	}

	var err error
	if ctx.Instance, err = vk.CreateInstance(&icInfo, nil); err != nil {
		fmt.Println("Could not create instance!")
		panic(err.Error())
	}
}

func (app *Context) createSurface(hInstance windows.Handle, hWnd windows.HWND) {
	ci := vk.Win32SurfaceCreateInfoKHR{
		Hinstance: hInstance,
		Hwnd:      hWnd,
	}

	var err error
	app.Surface, err = vk.CreateWin32SurfaceKHR(app.Instance, &ci, nil)
	if err != nil {
		panic("Could not create surface: " + err.Error())
	}
}

func (app *Context) selectPhysicalDevice() {
	devices, err := vk.EnumeratePhysicalDevices(app.Instance)
	if err != nil {
		panic("Could not enumerate physical devices: " + err.Error())
	}

	for _, dev := range devices {
		if app.isDeviceSuitable(dev) {
			app.PhysicalDevice = dev
			return
		}
	}

	panic("Could not find a suitable physical device!")
}

func (app *Context) isDeviceSuitable(device vk.PhysicalDevice) bool {
	// props := vk.GetPhysicalDeviceProperties(device)

	// fmt.Printf("Found Physical Device:\n")
	// fmt.Printf("  Device Name:\t\t%s\n", props.DeviceName)
	// fmt.Printf("  Vendor/Device ID:\t0x%x / 0x%x \n", props.VendorID, props.DeviceID)
	// fmt.Printf("  Device Type:\t\t%v\n", props.DeviceType)
	// fmt.Printf("  Device API Version:\t%s\n", versionToString(props.ApiVersion))
	// fmt.Printf("  Driver Version:\t%s\n", versionToString(props.DriverVersion))

	/* Suitability is:
	1) Support for the queue families we want to use (graphics)
	2) Support for the surface presentation extensions we want to use
	3) Support for swap chains // TODO
	4) Support for sampler anisotropy
	*/

	features := vk.GetPhysicalDeviceFeatures(device)

	extSupport := app.checkDeviceExtensionSupport(device)
	if !extSupport {
		panic("extensions not supported!")
	}
	inds := app.analyzeQueueFamilies(device)

	return extSupport && inds.graphicsIndex.HasValue() && inds.presentIndex.HasValue() && features.SamplerAnisotropy
}

func (app *Context) checkDeviceExtensionSupport(device vk.PhysicalDevice) bool {
	devExtensions, err := vk.EnumerateDeviceExtensionProperties(device, "")
	if err != nil {
		panic(err.Error() + ": Could not enumerate device extension properties!")
	}

	foundProps := make(map[string]bool, len(app.EnableDeviceExtensions))
	// fmt.Println("Searching for device extensions:")
	for _, name := range app.EnableDeviceExtensions {
		// init the found map with a false entry for each required extension
		foundProps[name] = false
	}

	for _, exProp := range devExtensions {
		if _, ok := foundProps[exProp.ExtensionName]; ok {
			foundProps[exProp.ExtensionName] = true
		}
	}

	haveAllExtensions := true
	for _, v := range foundProps {
		haveAllExtensions = haveAllExtensions && v
	}
	return haveAllExtensions
}

func (app *Context) analyzeQueueFamilies(device vk.PhysicalDevice) queueFamIndices {
	qfp := vk.GetPhysicalDeviceQueueFamilyProperties(device)

	var inds queueFamIndices

	for i, p := range qfp {
		if (p.QueueFlags & vk.QUEUE_GRAPHICS_BIT) != 0 {
			inds.graphicsIndex.Set(uint32(i))
		}
		surf, err := vk.GetPhysicalDeviceSurfaceSupportKHR(device, uint32(i), app.Surface)
		if err != nil {
			panic(err)
		}

		if surf {
			inds.presentIndex.Set(uint32(i))
		}

		if inds.isComplete() {
			break
		}
	}
	return inds
}

func (app *Context) createLogicalDevice() {
	// Re-analyze for the selected device
	qfInds := app.analyzeQueueFamilies(app.PhysicalDevice)

	// creates one or two entries, depending on how many queue families are needed
	uniqueQueueFams := make(map[uint32]bool)
	if !qfInds.graphicsIndex.HasValue() {
		panic("no graphics index found!")
	}
	uniqueQueueFams[qfInds.graphicsIndex.Value()] = true

	if !qfInds.presentIndex.HasValue() {
		panic("no presentation index found!")
	}
	uniqueQueueFams[qfInds.presentIndex.Value()] = true

	var dqCreateInfos []vk.DeviceQueueCreateInfo
	for k, v := range uniqueQueueFams {
		if v {
			// This family is selected as (one of possibly many) needed queues
			dqCreateInfos = append(dqCreateInfos,
				vk.DeviceQueueCreateInfo{
					QueueFamilyIndex: k,
					PQueuePriorities: []float32{1.0},
				})
		}
	}

	deviceFeatures := vk.PhysicalDeviceFeatures{
		RobustBufferAccess: true,
		SamplerAnisotropy:  true,
		FillModeNonSolid:   true,
	}

	createInfo := vk.DeviceCreateInfo{
		PQueueCreateInfos:       dqCreateInfos,
		PEnabledFeatures:        &deviceFeatures,
		PpEnabledExtensionNames: app.EnableDeviceExtensions,
		// EnabledLayerNames:     (deprecated)
	}

	f2 := vk.GetPhysicalDeviceFeatures2(app.PhysicalDevice)
	if !f2.Features.RobustBufferAccess {
		panic("robust buffer access not supported")
	}
	f2n := vk.PhysicalDeviceRobustness2FeaturesEXT{
		// PNext:               nil,
		// RobustBufferAccess2: false,
		// RobustImageAccess2:  false,
		NullDescriptor: true,
	}
	f2.PNext = unsafe.Pointer(f2n.Vulkanize())
	// Enabling all features in f2
	createInfo.PNext = unsafe.Pointer(f2.Vulkanize())
	createInfo.PEnabledFeatures = nil

	device, err := vk.CreateDevice(app.PhysicalDevice, &createInfo, nil)

	if err != nil {
		fmt.Printf("Logical device creation failed! (%s)\n", err.Error())
		panic(err)
	}
	app.Device = device

	app.GraphicsQueueFamilyIndex = qfInds.graphicsIndex.Value()
	app.GraphicsQueue = vk.GetDeviceQueue(app.Device, qfInds.graphicsIndex.Value(), 0)
	app.PresentQueueFamilyIndex = qfInds.presentIndex.Value()
	app.PresentQueue = vk.GetDeviceQueue(app.Device, qfInds.presentIndex.Value(), 0)
}

// ---------------------
type optUint32 struct {
	hasValue bool
	value    uint32
}

func (s *optUint32) Value() uint32 {
	if !s.hasValue {
		panic("Value not set on optUint32!")
	}
	return s.value
}

func (s *optUint32) Set(val uint32) {
	s.value = val
	s.hasValue = true
}

func (s *optUint32) HasValue() bool {
	return s.hasValue
}

type queueFamIndices struct {
	graphicsIndex optUint32
	presentIndex  optUint32
}

func (q *queueFamIndices) isComplete() bool {
	return q.graphicsIndex.HasValue() && q.presentIndex.HasValue()
}
