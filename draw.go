package main

import (
	"errors"
	"unsafe"

	"github.com/bbredesen/gltf"
	"github.com/bbredesen/go-vk"
	"github.com/bbredesen/vkm"
	"github.com/chewxy/math32"
)

func (app *App) destroyBuffers() {
	for i := range app.buffers {
		vk.DestroyBuffer(app.Device, app.buffers[i], nil)
		vk.FreeMemory(app.Device, app.bufferMemories[i], nil)
	}
	app.buffers = nil
	app.bufferMemories = nil
}

func (app *App) copyBuffer(srcBuffer, dstBuffer vk.Buffer, size vk.DeviceSize) {
	cbuf := app.BeginOneTimeCommands()

	region := vk.BufferCopy{
		SrcOffset: 0,
		DstOffset: 0,
		Size:      size,
	}

	vk.CmdCopyBuffer(cbuf, srcBuffer, dstBuffer, []vk.BufferCopy{region})

	app.EndOneTimeCommands(cbuf)
}

func (app *App) createBuffer(usage vk.BufferUsageFlags, size vk.DeviceSize, memProps vk.MemoryPropertyFlags) (buffer vk.Buffer, memory vk.DeviceMemory) {

	bufferCI := vk.BufferCreateInfo{
		Size:        size,
		Usage:       usage,
		SharingMode: vk.SHARING_MODE_EXCLUSIVE,
	}

	var r vk.Result

	if r, buffer = vk.CreateBuffer(app.Device, &bufferCI, nil); r != vk.SUCCESS {
		panic("Could not create buffer: " + r.String())
	}

	memReq := vk.GetBufferMemoryRequirements(app.Device, buffer)

	memAllocInfo := vk.MemoryAllocateInfo{
		AllocationSize:  memReq.Size,
		MemoryTypeIndex: uint32(app.FindMemoryType(memReq.MemoryTypeBits, memProps)), //vk.MEMORY_PROPERTY_HOST_VISIBLE_BIT|vk.MEMORY_PROPERTY_HOST_COHERENT_BIT)),
	}

	if r, memory = vk.AllocateMemory(app.Device, &memAllocInfo, nil); r != vk.SUCCESS {
		panic("Could not allocate memory for buffer: " + r.String())
	}
	if r := vk.BindBufferMemory(app.Device, buffer, memory, 0); r != vk.SUCCESS {
		panic("Could not bind memory for buffer: " + r.String())
	}

	return
}

// glTF specifies that the default camera is at the origin, and defines the camera space as looking at -Z, but not much
// else. Picking defaults here that nicely "frame" the range [-1..1] for X and Y in an orthographic projection.
func defaultCamera() vkm.Mat {
	if false {
		const xmag, ymag = 3, 3
		const znear, zfar = 0, 1

		var eye = vkm.Origin()
		var look, up = vkm.UnitVecZ().Invert(), vkm.UnitVecY()

		return vkm.GlTFOrthoProjection(xmag, ymag, znear, zfar).MultM(vkm.Camera(eye, look, up))
	} else {
		return vkm.GlTFPerspective(2*math32.Pi*(60.0/360.0), 1, 1, 10000).MultM(vkm.LookAt(vkm.NewPt(200, 300, 200), vkm.Origin(), vkm.UnitVecY()))
	}
}

func (app *App) loadGlTF(doc *gltf.ResolvedGlTF) error {
	app.modelDoc = doc

	for _, cam := range doc.Cameras {
		if cam.Type == gltf.PERSPECTIVE {
			app.cameras = append(app.cameras, vkm.Perspective(cam.Perspective.Yfov, cam.Perspective.AspectRatio, cam.Perspective.Znear, cam.Perspective.Zfar))
		} else if cam.Type == gltf.ORTHOGRAPHIC {
			app.cameras = append(app.cameras, vkm.GlTFOrthoProjection(cam.Orthographic.Xmag, cam.Orthographic.Ymag, cam.Orthographic.Znear, cam.Orthographic.Zfar))
		}
	}

	// Use the default camera if none have been defined.
	if len(app.cameras) == 0 {
		app.cameras = append(app.cameras, defaultCamera())
	}

	// TODO (Temporarily) override everything with the default camera
	app.cameras = []vkm.Mat{defaultCamera()}

	for _, docBuf := range doc.Buffers {
		vkBuf, bufMem := app.createBuffer(vk.BUFFER_USAGE_VERTEX_BUFFER_BIT|vk.BUFFER_USAGE_INDEX_BUFFER_BIT, vk.DeviceSize(docBuf.ByteLength), vk.MEMORY_PROPERTY_DEVICE_LOCAL_BIT|vk.MEMORY_PROPERTY_HOST_COHERENT_BIT)
		r, ptr := vk.MapMemory(app.Device, bufMem, 0, vk.DeviceSize(docBuf.ByteLength), 0)
		if r != vk.SUCCESS {
			vk.DestroyBuffer(app.Device, vkBuf, nil)
			return errors.New("failed to map memory for buffer, result code was " + r.String())
		}

		vk.MemCopySlice(unsafe.Pointer(ptr), docBuf.Data)
		vk.UnmapMemory(app.Device, bufMem)

		app.buffers = append(app.buffers, vkBuf)
		app.bufferMemories = append(app.bufferMemories, bufMem)
	}

	// app.VulkanPipeline.accessorBinding = vk.VertexInputBindingDescription{
	// 	Binding:   0,
	// 	Stride:    uint32(doc.Accessors[0].Stride()),
	// 	InputRate: vk.VERTEX_INPUT_RATE_VERTEX,
	// }
	// app.VulkanPipeline.accessorAttr = vk.VertexInputAttributeDescription{
	// 	Location: 0,
	// 	Binding:  0,
	// 	Format:   accessorToFormat(doc.Accessors[0].Type, doc.Accessors[0].ComponentType),
	// 	Offset:   0,
	// }

	// TODO

	app.scene = NewScene(doc.Scene)
	return nil
}

func accessorToFormat(accType gltf.AccessorTypeEnum, compType gltf.ComponentTypeEnum) vk.Format {

	switch accType {
	case gltf.SCALAR:
		switch compType {
		case gltf.BYTE:
			return vk.FORMAT_R8_SINT
		case gltf.UNSIGNED_BYTE:
			return vk.FORMAT_R8_UINT
		case gltf.SHORT:
			return vk.FORMAT_R16_SINT
		case gltf.UNSIGNED_SHORT:
			return vk.FORMAT_R16_UINT
		case gltf.UNSIGNED_INT:
			return vk.FORMAT_R32_UINT
		case gltf.FLOAT:
			return vk.FORMAT_R32_SFLOAT
		}

	case gltf.VEC2:
		switch compType {
		case gltf.BYTE:
			return vk.FORMAT_R8G8_SINT
		case gltf.UNSIGNED_BYTE:
			return vk.FORMAT_R8G8_UINT
		case gltf.SHORT:
			return vk.FORMAT_R16G16_SINT
		case gltf.UNSIGNED_SHORT:
			return vk.FORMAT_R16G16_UINT
		case gltf.UNSIGNED_INT:
			return vk.FORMAT_R32G32_UINT
		case gltf.FLOAT:
			return vk.FORMAT_R32G32_SFLOAT
		}

	case gltf.VEC3:
		switch compType {
		case gltf.BYTE:
			return vk.FORMAT_R8G8B8_SINT
		case gltf.UNSIGNED_BYTE:
			return vk.FORMAT_R8G8B8_UINT
		case gltf.SHORT:
			return vk.FORMAT_R16G16B16_SINT
		case gltf.UNSIGNED_SHORT:
			return vk.FORMAT_R16G16B16_UINT
		case gltf.UNSIGNED_INT:
			return vk.FORMAT_R32G32B32_UINT
		case gltf.FLOAT:
			return vk.FORMAT_R32G32B32_SFLOAT
		}

	case gltf.VEC4:
		switch compType {
		case gltf.BYTE:
			return vk.FORMAT_R8G8B8A8_SINT
		case gltf.UNSIGNED_BYTE:
			return vk.FORMAT_R8G8B8A8_UINT
		case gltf.SHORT:
			return vk.FORMAT_R16G16B16A16_SINT
		case gltf.UNSIGNED_SHORT:
			return vk.FORMAT_R16G16B16A16_UINT
		case gltf.UNSIGNED_INT:
			return vk.FORMAT_R32G32B32A32_UINT
		case gltf.FLOAT:
			return vk.FORMAT_R32G32B32A32_SFLOAT
		}
	}

	return vk.FORMAT_UNDEFINED
}
