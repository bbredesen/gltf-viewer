package main

import (
	"unsafe"

	"github.com/bbredesen/gltf"
	"github.com/bbredesen/gltf-viewer/shared"
	"github.com/bbredesen/gltf-viewer/vkctx"
	"github.com/bbredesen/go-vk"
	"github.com/bbredesen/vkm"
	"golang.org/x/sys/windows"
)

// func init() {

// 	flag.Parse()
// }

type App struct {
	winapp   *shared.Win32App
	messages chan shared.WindowMessage

	vkctx.Context
	VulkanPipeline

	currentImage uint32

	// vertexBuffer, indexBuffer             vk.Buffer
	// vertexBufferMemory, indexBufferMemory vk.DeviceMemory

	// indexCount int

	// quadVertStart, quadIndsStart int

	buffers        []vk.Buffer
	bufferMemories []vk.DeviceMemory

	modelDoc *gltf.ResolvedGlTF

	cameras []vkm.Mat
	scene   *SceneNode
}

func NewApp() *App {
	c := make(chan shared.WindowMessage, 32)

	return &App{
		winapp:   shared.NewWin32App(c),
		messages: c,
	}
}

func (app *App) Initialize() {
	app.winapp.ClassName = "gltf-viewer"
	app.winapp.Width, app.winapp.Height = 800, 800
	app.winapp.Initialize("gltf-viewer")

	app.EnableApiLayers = append(app.EnableApiLayers, "VK_LAYER_KHRONOS_validation")
	app.EnableInstanceExtensions = app.winapp.GetRequiredInstanceExtensions()

	app.EnableDeviceExtensions = append(app.EnableDeviceExtensions, vk.KHR_SWAPCHAIN_EXTENSION_NAME, vk.EXT_ROBUSTNESS_2_EXTENSION_NAME)

	app.Context.Initialize(windows.Handle(app.winapp.HInstance), windows.HWND(app.winapp.HWnd))

	app.VulkanPipeline.Initialize(&app.Context)
}

func (app *App) Teardown() {
	vk.DeviceWaitIdle(app.ctx.Device)

	app.destroyBuffers()

	app.VulkanPipeline.Teardown()
	app.Context.Teardown()
}

func (app *App) drawFrame() {
	vk.WaitForFences(app.ctx.Device, []vk.Fence{app.ctx.InFlightFence}, true, ^uint64(0))

	var err error
	if app.currentImage, err = vk.AcquireNextImageKHR(app.ctx.Device, app.ctx.Swapchain, ^uint64(0), app.ctx.ImageAvailableSemaphore, vk.Fence(vk.NULL_HANDLE)); err != nil {
		if err == vk.SUBOPTIMAL_KHR || err == vk.ERROR_OUT_OF_DATE_KHR {
			// app.vp.recreateSwapchain()
			return
		} else {
			panic("Could not acquire next image! " + err.Error())
		}
	}

	vk.ResetFences(app.ctx.Device, []vk.Fence{app.ctx.InFlightFence})

	// Somewhere in here update animations before recording commands

	vk.ResetCommandBuffer(app.ctx.CommandBuffers[app.currentImage], 0)
	app.recordRenderingCommands(app.ctx.CommandBuffers[app.currentImage])

	// app.updateUniformBuffer(app.currentImage)

	submitInfo := vk.SubmitInfo{
		PWaitSemaphores:   []vk.Semaphore{app.ctx.ImageAvailableSemaphore},
		PWaitDstStageMask: []vk.PipelineStageFlags{vk.PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT},
		PCommandBuffers:   []vk.CommandBuffer{app.ctx.CommandBuffers[app.currentImage]},
		PSignalSemaphores: []vk.Semaphore{app.ctx.RenderFinishedSemaphore},
	}

	if err := vk.QueueSubmit(app.ctx.GraphicsQueue, []vk.SubmitInfo{submitInfo}, app.ctx.InFlightFence); err != nil {
		panic("Could not submit to graphics queue! " + err.Error())
	}

	// Present the drawn image
	presentInfo := vk.PresentInfoKHR{
		PWaitSemaphores: []vk.Semaphore{app.ctx.RenderFinishedSemaphore},
		PSwapchains:     []vk.SwapchainKHR{app.ctx.Swapchain},
		PImageIndices:   []uint32{app.currentImage},
	}

	if r := vk.QueuePresentKHR(app.ctx.PresentQueue, &presentInfo); err != nil && r != vk.SUBOPTIMAL_KHR && r != vk.ERROR_OUT_OF_DATE_KHR {
		panic("Could not submit to presentation queue! " + err.Error())
	}

}

func (app *App) recordRenderingCommands(cb vk.CommandBuffer) {
	cbBeginInfo := vk.CommandBufferBeginInfo{
		Flags: vk.COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT,
	}

	colorCV := vk.ClearValue{}
	ccv := vk.ClearColorValue{}
	ccv.AsTypeFloat32([4]float32{0.0, 0.0, 0.0, 1.0})
	colorCV.AsColor(ccv)
	depthCv := vk.ClearValue{}
	depthCv.AsDepthStencil(vk.ClearDepthStencilValue{Depth: 1.0})

	rpBeginInfo := vk.RenderPassBeginInfo{
		RenderPass:  app.renderPass,
		Framebuffer: app.SwapChainFramebuffers[app.currentImage],
		RenderArea: vk.Rect2D{
			Offset: vk.Offset2D{X: 0, Y: 0},
			Extent: app.SwapchainExtent,
		},
		PClearValues: []vk.ClearValue{colorCV, depthCv},
	}

	vk.BeginCommandBuffer(cb, &cbBeginInfo)

	// Static camera for now, TODO read from nodes
	camBytes := make([]byte, unsafe.Sizeof(app.cameras[0]))
	vk.MemCopyObj(unsafe.Pointer(&camBytes[0]), &app.cameras[0])
	vk.CmdPushConstants(cb, app.pipelineLayout, vk.SHADER_STAGE_VERTEX_BIT, 0, camBytes)

	vk.CmdBeginRenderPass(cb, &rpBeginInfo, vk.SUBPASS_CONTENTS_INLINE)
	// Need to set up a uniform buffer for the camera+perspective matrix?

	vk.CmdBindPipeline(cb, vk.PIPELINE_BIND_POINT_GRAPHICS, app.graphicsPipeline)
	// bind vert, index bufs

	app.RenderNode(app.scene, cb)
	// app.recordMeshCommands(cb, app.modelDoc.Scene.Nodes[0].Mesh)

	// vk.CmdBindVertexBuffers(cb, 0, app.buffers, []vk.DeviceSize{0})

	// vk.CmdDraw(cb, 3, 2, 0, 0)

	// vk.CmdBindIndexBuffer(cb, app.indexBuffer, 0, vk.INDEX_TYPE_UINT16)

	// vk.CmdDrawIndexed(cb, uint32(app.quadIndsStart), 1, 0, 0, 0)

	// vk.CmdBindPipeline(cb, vk.PIPELINE_BIND_POINT_GRAPHICS, app.graphicsPipelines[1]) // stencil quad portion pipeline
	// vk.CmdDrawIndexed(cb, uint32(app.indexCount-app.quadIndsStart)-4, 1, uint32(app.quadIndsStart), int32(app.quadVertStart), 0)

	// vk.CmdNextSubpass(cb, vk.SUBPASS_CONTENTS_INLINE)

	// vk.CmdBindPipeline(cb, vk.PIPELINE_BIND_POINT_GRAPHICS, app.graphicsPipelines[2]) // Color pass

	// // vk.CmdDrawIndexed(cb, 5, 1, uint32(app.indexCount)-5, 0, 0)
	// vk.CmdDrawIndexed(cb, 4, 1, uint32(app.indexCount)-4, int32(app.quadVertStart), 0)

	// draw

	vk.CmdEndRenderPass(cb)
	vk.EndCommandBuffer(cb)

}

var attrKeys = []gltf.AttributeKey{gltf.POSITION, gltf.NORMAL, gltf.TANGENT, gltf.TEXCOORD_0, gltf.TEXCOORD_1, gltf.COLOR_0}

// func (app *App) recordMeshCommands(cb vk.CommandBuffer, mesh *gltf.ResolvedMesh) {
// 	for _, p := range mesh.Primitives {
// 		bufs := make([]vk.Buffer, len(attrKeys))
// 		offsets := make([]vk.DeviceSize, len(attrKeys))

// 		for i, attrKey := range attrKeys {
// 			if ra, ok := p.Attributes[attrKey]; !ok {
// 				bufs[i] = vk.Buffer(vk.NULL_HANDLE)
// 			} else {
// 				bufs[i] = app.buffers[ra.BufferView.BufferView.Buffer]
// 				offsets[i] = vk.DeviceSize(ra.ByteOffset + ra.BufferView.ByteOffset)
// 			}

// 		}

// 		vk.CmdBindVertexBuffers(cb, 0, bufs, offsets)

// 		if p.Indices != nil {
// 			bufIdx := p.Indices.BufferView.BufferView.Buffer

// 			var idxType vk.IndexType
// 			switch p.Indices.ComponentType {
// 			case gltf.UNSIGNED_BYTE:
// 				idxType = vk.INDEX_TYPE_UINT8_EXT
// 				panic("unsported index type UINT8")
// 			case gltf.UNSIGNED_SHORT:
// 				idxType = vk.INDEX_TYPE_UINT16
// 			case gltf.UNSIGNED_INT:
// 				idxType = vk.INDEX_TYPE_UINT32
// 			}

// 			model := vkm.NewMatTranslate(vkm.NewVec(-1, 0, 0))
// 			modelBytes := make([]byte, unsafe.Sizeof(model))
// 			vk.MemCopyObj(unsafe.Pointer(&modelBytes[0]), &model)

// 			vk.CmdPushConstants(cb, app.pipelineLayout, vk.SHADER_STAGE_VERTEX_BIT, uint32(unsafe.Sizeof(model)), modelBytes)

// 			vk.CmdBindIndexBuffer(cb, app.buffers[bufIdx], vk.DeviceSize(p.Indices.ByteOffset+p.Indices.BufferView.ByteOffset), idxType)
// 			vk.CmdDrawIndexed(cb, uint32(p.Indices.Count), 1, 0, 0, 0)
// 		} else {
// 			vk.CmdDraw(cb, uint32(p.Attributes[gltf.POSITION].Count), 1, 0, 0)
// 		}
// 	}
// }

/*
Scene Tree:

- Map to gltf structure.
- Root is gltf scene, each branch is a gltf node
- Track base and effective transforms at each treenode. Base is the transform specified in the gltf node,
  effective is the composition of transforms to this point.
- Animation updates will modify the effective transform at a node. Animation must also update the effectives at all
  child nodes? Or is effective at a node only subject to animation/morphs, and the rendering process will calculate the
  model matrix as it steps through?

*/

type SceneNode struct {
	Parent   *SceneNode
	Children []*SceneNode

	ModelNode *gltf.ResolvedNode
	// isCamera?

	BaseTransform    vkm.Mat
	CurrentTransform vkm.Mat
}

func NewScene(s *gltf.ResolvedScene) *SceneNode {
	root := NewSceneNode(nil, nil)
	for i := range s.Nodes {
		root.Children = append(root.Children, NewSceneNode(root, s.Nodes[i]))
	}

	return root
}

func NewSceneNode(parent *SceneNode, model *gltf.ResolvedNode) *SceneNode {
	rval := &SceneNode{
		BaseTransform:    vkm.Identity(), // TODO
		CurrentTransform: vkm.Identity(), // TODO
		Parent:           parent,
		ModelNode:        model,
	}

	if parent != nil {
		rval.ApplyTransform(parent.CurrentTransform)

		for i := range model.Children {
			next := NewSceneNode(rval, model.Children[i])
			rval.Children = append(rval.Children, next)
		}

	}

	return rval
}

func (n *SceneNode) ApplyTransform(txfm vkm.Mat) {
	n.CurrentTransform = txfm.MultM(n.BaseTransform)

}

const _modelPCOffset = uint32(unsafe.Sizeof(vkm.Mat{}))

func (app *App) RenderNode(n *SceneNode, cb vk.CommandBuffer) {
	if n.ModelNode != nil && n.ModelNode.Camera != nil {
		projViewMat := n.ModelNode.Camera.ProjMatrix.MultM(n.CurrentTransform)
		// This won't work as is...need to put projection in a bound uniform buffer before executing the command buffer
		vk.CmdPushConstants(cb, app.pipelineLayout, vk.SHADER_STAGE_VERTEX_BIT, 0, projViewMat.AsBytes())
	}

	// self render...
	vk.CmdPushConstants(cb, app.pipelineLayout, vk.SHADER_STAGE_VERTEX_BIT, _modelPCOffset, n.CurrentTransform.AsBytes())

	if n.ModelNode != nil && n.ModelNode.Mesh != nil {
		for _, p := range n.ModelNode.Mesh.Primitives {
			bufs := make([]vk.Buffer, len(attrKeys))
			offsets := make([]vk.DeviceSize, len(attrKeys))

			for i, attrKey := range attrKeys {
				if ra, ok := p.Attributes[attrKey]; !ok {
					bufs[i] = vk.Buffer(vk.NULL_HANDLE)
				} else {
					bufs[i] = app.buffers[ra.BufferView.BufferView.Buffer]
					offsets[i] = vk.DeviceSize(ra.ByteOffset + ra.BufferView.ByteOffset)
				}

			}

			vk.CmdBindVertexBuffers(cb, 0, bufs, offsets)

			if p.Indices != nil {
				bufIdx := p.Indices.BufferView.BufferView.Buffer

				var idxType vk.IndexType
				switch p.Indices.ComponentType {
				case gltf.UNSIGNED_BYTE:
					idxType = vk.INDEX_TYPE_UINT8_EXT
					panic("unsported index type UINT8")
				case gltf.UNSIGNED_SHORT:
					idxType = vk.INDEX_TYPE_UINT16
				case gltf.UNSIGNED_INT:
					idxType = vk.INDEX_TYPE_UINT32
				}

				vk.CmdBindIndexBuffer(cb, app.buffers[bufIdx], vk.DeviceSize(p.Indices.ByteOffset+p.Indices.BufferView.ByteOffset), idxType)
				vk.CmdDrawIndexed(cb, uint32(p.Indices.Count), 1, 0, 0, 0)
			} else {
				vk.CmdDraw(cb, uint32(p.Attributes[gltf.POSITION].Count), 1, 0, 0)
			}
		}
	}

	for _, child := range n.Children {
		child.ApplyTransform(n.CurrentTransform)
		app.RenderNode(child, cb)
	}
}
