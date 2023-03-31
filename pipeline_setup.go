package main

//go:generate glslc.exe shaders/shader.vert -o shaders/vert.spv
//go:generate glslc.exe shaders/shader.frag -o shaders/frag.spv

import (
	"os"
	"unsafe"

	"github.com/bbredesen/gltf"
	"github.com/bbredesen/gltf-viewer/vkctx"
	"github.com/bbredesen/go-vk"
	"github.com/bbredesen/vkm"
)

type VulkanPipeline struct {
	ctx              *vkctx.Context
	pipelineLayout   vk.PipelineLayout
	graphicsPipeline vk.Pipeline

	// Renderpass
	renderPass vk.RenderPass

	stencilSubpass, colorSubpass     vk.SubpassDescription
	stencilImage, colorImage         vk.Image
	stencilMemory, colorMemory       vk.DeviceMemory
	stencilImageView, colorImageView vk.ImageView

	vertShaderModule, fragShaderModule vk.ShaderModule

	accessorBindings map[gltf.AttributeKey]vk.VertexInputBindingDescription
	accessorAttrs    map[gltf.AttributeKey]vk.VertexInputAttributeDescription
}

func (vp *VulkanPipeline) Initialize(ctx *vkctx.Context) {
	vp.ctx = ctx
	// vp.stencilImage, vp.stencilMemory = ctx.CreateImage(ctx.SwapchainExtent, vk.FORMAT_S8_UINT, vk.IMAGE_TILING_OPTIMAL, vk.IMAGE_USAGE_DEPTH_STENCIL_ATTACHMENT_BIT, vk.MEMORY_PROPERTY_DEVICE_LOCAL_BIT)
	// vp.stencilImageView = ctx.CreateImageView(vp.stencilImage, vk.FORMAT_S8_UINT, vk.IMAGE_ASPECT_STENCIL_BIT)

	vp.colorImage, vp.colorMemory = ctx.CreateImage(ctx.SwapchainExtent, vk.FORMAT_R32G32B32A32_SFLOAT, vk.IMAGE_TILING_OPTIMAL, vk.IMAGE_USAGE_COLOR_ATTACHMENT_BIT, vk.MEMORY_PROPERTY_DEVICE_LOCAL_BIT)
	vp.colorImageView = ctx.CreateImageView(vp.colorImage, vk.FORMAT_R32G32B32A32_SFLOAT, vk.IMAGE_ASPECT_COLOR_BIT)

	vp.CreateRenderPass()

	vp.CreateFramebuffers()

	vp.CreateGraphicsPipelines()
}

func (vp *VulkanPipeline) standardViewport() *vk.PipelineViewportStateCreateInfo {

	viewport := vk.Viewport{
		X:        0.0,
		Y:        0.0,
		Width:    float32(vp.ctx.SwapchainExtent.Width),
		Height:   float32(vp.ctx.SwapchainExtent.Height),
		MinDepth: 0.0,
		MaxDepth: 1.0,
	}

	scissor := vk.Rect2D{
		Offset: vk.Offset2D{X: 0, Y: 0},
		Extent: vp.ctx.SwapchainExtent,
	}

	return &vk.PipelineViewportStateCreateInfo{
		PViewports: []vk.Viewport{viewport},
		PScissors:  []vk.Rect2D{scissor},
	}

}

func (vp *VulkanPipeline) prebuildVertexInputDescriptions() {
	vp.accessorBindings = make(map[gltf.AttributeKey]vk.VertexInputBindingDescription)
	vp.accessorAttrs = make(map[gltf.AttributeKey]vk.VertexInputAttributeDescription)

	vp.accessorBindings[gltf.POSITION] = vk.VertexInputBindingDescription{
		Binding:   0,
		Stride:    3 * 4, // POSITION is always a VEC3 of FLOAT, TODO handle interleaved data
		InputRate: vk.VERTEX_INPUT_RATE_VERTEX,
	}
	vp.accessorAttrs[gltf.POSITION] = vk.VertexInputAttributeDescription{
		Location: 0,
		Binding:  0,
		Format:   vk.FORMAT_R32G32B32_SFLOAT,
		Offset:   0, // TODO handle offset from base of vertex?
	}

	vp.accessorBindings[gltf.NORMAL] = vk.VertexInputBindingDescription{
		Binding:   1,
		Stride:    3 * 4, // NORMAL is always a VEC3 of FLOAT, TODO handle interleaved data
		InputRate: vk.VERTEX_INPUT_RATE_VERTEX,
	}
	vp.accessorAttrs[gltf.NORMAL] = vk.VertexInputAttributeDescription{
		Location: 1,
		Binding:  1,
		Format:   vk.FORMAT_R32G32B32_SFLOAT,
		Offset:   0, // TODO handle offset from base of vertex?
	}
}

func (vp *VulkanPipeline) CreateGraphicsPipelines() {
	vp.prebuildVertexInputDescriptions()

	vp.vertShaderModule = vp.createShaderModule("shaders/vert.spv")
	vp.fragShaderModule = vp.createShaderModule("shaders/frag.spv")

	vertShaderStageCreateInfo := vk.PipelineShaderStageCreateInfo{
		Stage:               vk.SHADER_STAGE_VERTEX_BIT,
		Module:              vp.vertShaderModule,
		PName:               "main",
		PSpecializationInfo: &vk.SpecializationInfo{},
	}

	fragShaderStageCreateInfo := vk.PipelineShaderStageCreateInfo{
		Stage:               vk.SHADER_STAGE_FRAGMENT_BIT,
		Module:              vp.fragShaderModule,
		PName:               "main",
		PSpecializationInfo: &vk.SpecializationInfo{},
	}

	shaderStages := []vk.PipelineShaderStageCreateInfo{
		vertShaderStageCreateInfo, fragShaderStageCreateInfo,
	}

	vertexBindings, vertexAttrs := []vk.VertexInputBindingDescription{}, []vk.VertexInputAttributeDescription{}

	vertexBindings = append(vertexBindings, vp.accessorBindings[gltf.POSITION], vp.accessorBindings[gltf.NORMAL])
	vertexAttrs = append(vertexAttrs, vp.accessorAttrs[gltf.POSITION], vp.accessorAttrs[gltf.NORMAL])

	vertexInputCreateInfo := vk.PipelineVertexInputStateCreateInfo{
		PNext:                        nil,
		Flags:                        0,
		PVertexBindingDescriptions:   vertexBindings,
		PVertexAttributeDescriptions: vertexAttrs,
	}

	inputAssemblyCreateInfo := vk.PipelineInputAssemblyStateCreateInfo{
		Topology:               vk.PRIMITIVE_TOPOLOGY_TRIANGLE_LIST, // TODO
		PrimitiveRestartEnable: false,
	}

	viewport := vk.Viewport{
		X:        0.0,
		Y:        0.0,
		Width:    float32(vp.ctx.SwapchainExtent.Width),
		Height:   float32(vp.ctx.SwapchainExtent.Height),
		MinDepth: 0.0,
		MaxDepth: 1.0,
	}

	scissor := vk.Rect2D{
		Offset: vk.Offset2D{X: 0, Y: 0},
		Extent: vp.ctx.SwapchainExtent,
	}

	viewportStateCreateInfo := vk.PipelineViewportStateCreateInfo{
		PViewports: []vk.Viewport{viewport},
		PScissors:  []vk.Rect2D{scissor},
	}

	rasterizerCreateInfo := vk.PipelineRasterizationStateCreateInfo{
		DepthClampEnable:        false,
		RasterizerDiscardEnable: false,
		PolygonMode:             vk.POLYGON_MODE_FILL,
		LineWidth:               1.0,
		CullMode:                vk.CULL_MODE_NONE,
		FrontFace:               vk.FRONT_FACE_COUNTER_CLOCKWISE,
		DepthBiasEnable:         false,
	}

	multisampleCreateInfo := vk.PipelineMultisampleStateCreateInfo{
		SampleShadingEnable:  false,
		RasterizationSamples: vk.SAMPLE_COUNT_1_BIT,
		MinSampleShading:     1.0,
	}

	writeMask := vk.COLOR_COMPONENT_R_BIT |
		vk.COLOR_COMPONENT_G_BIT |
		vk.COLOR_COMPONENT_B_BIT |
		vk.COLOR_COMPONENT_A_BIT

	colorBlendAttachment := vk.PipelineColorBlendAttachmentState{
		ColorWriteMask: writeMask,
		BlendEnable:    false,

		// All ignored, b/c blend enable is false above
		// SrcColorBlendFactor: vk.BLEND_FACTOR_ONE,
		// DstColorBlendFactor: vk.BLEND_FACTOR_ZERO,
		// ColorBlendOp:        vk.BLEND_OP_ADD,
		// SrcAlphaBlendFactor: vk.BLEND_FACTOR_ONE,
		// DstAlphaBlendFactor: vk.BLEND_FACTOR_ZERO,
		// AlphaBlendOp:        vk.BLEND_OP_ADD,
	}

	colorBlendStateCreateInfo := vk.PipelineColorBlendStateCreateInfo{
		PAttachments: []vk.PipelineColorBlendAttachmentState{colorBlendAttachment},
	}

	depthStencilStateCreateInfo := vk.PipelineDepthStencilStateCreateInfo{
		DepthTestEnable:       true,
		DepthWriteEnable:      true,
		DepthCompareOp:        vk.COMPARE_OP_LESS,
		DepthBoundsTestEnable: false,
		StencilTestEnable:     false,
		Front:                 vk.StencilOpState{},
		Back:                  vk.StencilOpState{},
		MinDepthBounds:        0,
		MaxDepthBounds:        1.0,
	}

	// dynamicStateCreateInfo := vk.PipelineDynamicStateCreateInfo{
	// 	PDynamicStates: []vk.DynamicState{vk.DYNAMIC_STATE_VIEWPORT, vk.DYNAMIC_STATE_SCISSOR},
	// }

	pipelineLayoutCreateInfo := vk.PipelineLayoutCreateInfo{
		// PSetLayouts: []vk.DescriptorSetLayout{app.descriptorSetLayout}, // TODO
		PPushConstantRanges: []vk.PushConstantRange{
			{
				StageFlags: vk.SHADER_STAGE_VERTEX_BIT,
				Offset:     0,
				Size:       2 * uint32(unsafe.Sizeof(vkm.Mat{})),
			},
		},
	}

	p, err := vk.CreatePipelineLayout(vp.ctx.Device, &pipelineLayoutCreateInfo, nil)
	if err != nil {
		panic(err)
	}
	vp.pipelineLayout = p

	pipelineCreateInfo := vk.GraphicsPipelineCreateInfo{
		PStages: shaderStages,
		// Fixed function stage information
		PVertexInputState:   &vertexInputCreateInfo,
		PInputAssemblyState: &inputAssemblyCreateInfo,
		PViewportState:      &viewportStateCreateInfo,
		PRasterizationState: &rasterizerCreateInfo,
		PMultisampleState:   &multisampleCreateInfo,
		PColorBlendState:    &colorBlendStateCreateInfo,
		PDepthStencilState:  &depthStencilStateCreateInfo,

		PTessellationState: &vk.PipelineTessellationStateCreateInfo{},
		PDynamicState:      &vk.PipelineDynamicStateCreateInfo{}, // dynamicStateCreateInfo,

		Layout:     vp.pipelineLayout,
		RenderPass: vp.renderPass,
		Subpass:    0,
	}

	if gp, err := vk.CreateGraphicsPipelines(
		vp.ctx.Device,
		0, // vk.NULL_HANDLE missing
		[]vk.GraphicsPipelineCreateInfo{pipelineCreateInfo},
		nil,
	); err != nil {
		panic(err)
	} else {
		vp.graphicsPipeline = gp[0]
	}
}

func (vp *VulkanPipeline) CreateRenderPass() {

	colorAttachmentDescription := vk.AttachmentDescription{
		Format:  vp.ctx.SwapchainImageFormat,
		Samples: vk.SAMPLE_COUNT_1_BIT,
		LoadOp:  vk.ATTACHMENT_LOAD_OP_CLEAR,
		StoreOp: vk.ATTACHMENT_STORE_OP_STORE,

		StencilLoadOp:  vk.ATTACHMENT_LOAD_OP_DONT_CARE,
		StencilStoreOp: vk.ATTACHMENT_STORE_OP_DONT_CARE,

		InitialLayout: vk.IMAGE_LAYOUT_UNDEFINED,
		FinalLayout:   vk.IMAGE_LAYOUT_PRESENT_SRC_KHR,
	}

	colorAttachmentRef := vk.AttachmentReference{
		Attachment: 0,
		Layout:     vk.IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL,
	}

	depthAttachmentDescription := vk.AttachmentDescription{
		Format:        vk.FORMAT_D32_SFLOAT,
		Samples:       vk.SAMPLE_COUNT_1_BIT,
		LoadOp:        vk.ATTACHMENT_LOAD_OP_CLEAR,
		StoreOp:       vk.ATTACHMENT_STORE_OP_DONT_CARE,
		InitialLayout: vk.IMAGE_LAYOUT_UNDEFINED,
		FinalLayout:   vk.IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL,
	}
	depthAttachmentRef := vk.AttachmentReference{
		Attachment: 1,
		Layout:     vk.IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL,
	}

	// stencilAttachmentDescription := vk.AttachmentDescription{
	// 	Format:  vk.FORMAT_S8_UINT,
	// 	Samples: vk.SAMPLE_COUNT_1_BIT,

	// 	// Applies to depth component
	// 	LoadOp:  vk.ATTACHMENT_LOAD_OP_CLEAR,
	// 	StoreOp: vk.ATTACHMENT_STORE_OP_DONT_CARE,

	// 	StencilLoadOp:  vk.ATTACHMENT_LOAD_OP_CLEAR,
	// 	StencilStoreOp: vk.ATTACHMENT_STORE_OP_DONT_CARE,

	// 	InitialLayout: vk.IMAGE_LAYOUT_UNDEFINED,
	// 	FinalLayout:   vk.IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL,
	// }

	// stencilAttachmentRef := vk.AttachmentReference{
	// 	Attachment: 1,
	// 	Layout:     vk.IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL,
	// }

	colorSubpassDescription := vk.SubpassDescription{
		PipelineBindPoint:       vk.PIPELINE_BIND_POINT_GRAPHICS,
		PColorAttachments:       []vk.AttachmentReference{colorAttachmentRef},
		PDepthStencilAttachment: &depthAttachmentRef,
	}

	// See
	// https://vulkan-tutorial.com/en/Drawing_a_triangle/Drawing/Rendering_and_presentation
	// https://registry.khronos.org/vulkan/specs/1.3-extensions/html/vkspec.html#VkSubpassDependency
	// This creates an execution/timing dependency between this render pass and the "implied" subpass (the prior renderpass) before this
	// renderpass begins. It specifiesd that the the color attachment output and depth testing stages in the prior pass
	// need to be completed before we attempt to write the color and depth attachments in this pass.

	dependencyToColor := vk.SubpassDependency{
		SrcSubpass:    vk.SUBPASS_EXTERNAL,
		DstSubpass:    0,
		SrcStageMask:  vk.PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT | vk.PIPELINE_STAGE_EARLY_FRAGMENT_TESTS_BIT,
		SrcAccessMask: 0,
		DstStageMask:  vk.PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT | vk.PIPELINE_STAGE_EARLY_FRAGMENT_TESTS_BIT,
		DstAccessMask: vk.ACCESS_COLOR_ATTACHMENT_WRITE_BIT | vk.ACCESS_DEPTH_STENCIL_ATTACHMENT_READ_BIT,
	}

	renderPassCreateInfo := vk.RenderPassCreateInfo{
		PAttachments:  []vk.AttachmentDescription{colorAttachmentDescription, depthAttachmentDescription},
		PSubpasses:    []vk.SubpassDescription{colorSubpassDescription},
		PDependencies: []vk.SubpassDependency{dependencyToColor},
	}

	var err error
	if vp.renderPass, err = vk.CreateRenderPass(vp.ctx.Device, &renderPassCreateInfo, nil); err != nil {
		panic(err)
	}
}

func (vp *VulkanPipeline) CreateFramebuffers() {

	vp.ctx.SwapChainFramebuffers = make([]vk.Framebuffer, len(vp.ctx.SwapchainImageViews))

	for i, iv := range vp.ctx.SwapchainImageViews {
		framebufferCreateInfo := vk.FramebufferCreateInfo{
			RenderPass:   vp.renderPass,
			PAttachments: []vk.ImageView{iv, vp.ctx.DepthImageView},
			Width:        vp.ctx.SwapchainExtent.Width,
			Height:       vp.ctx.SwapchainExtent.Height,
			Layers:       1,
		}

		fb, err := vk.CreateFramebuffer(vp.ctx.Device, &framebufferCreateInfo, nil)
		if err != nil {
			panic(err)
		}
		vp.ctx.SwapChainFramebuffers[i] = fb
	}
}

func (vp *VulkanPipeline) destroyFramebuffers() {
	for _, fb := range vp.ctx.SwapChainFramebuffers {
		vk.DestroyFramebuffer(vp.ctx.Device, fb, nil)
	}
	vp.ctx.SwapChainFramebuffers = nil
}

func (vp *VulkanPipeline) Teardown() {

	vk.DestroyShaderModule(vp.ctx.Device, vp.vertShaderModule, nil)
	vk.DestroyShaderModule(vp.ctx.Device, vp.fragShaderModule, nil)

	vk.DestroyImageView(vp.ctx.Device, vp.colorImageView, nil)
	// vk.DestroyImageView(vp.ctx.Device, vp.stencilImageView, nil)

	vk.DestroyImage(vp.ctx.Device, vp.colorImage, nil)
	// vk.DestroyImage(vp.ctx.Device, vp.stencilImage, nil)

	vk.FreeMemory(vp.ctx.Device, vp.colorMemory, nil)
	vk.FreeMemory(vp.ctx.Device, vp.stencilMemory, nil)

	vp.destroyFramebuffers()

	// for _, gp := range vp.graphicsPipelines {
	vk.DestroyPipeline(vp.ctx.Device, vp.graphicsPipeline, nil)
	// }
	vp.graphicsPipeline = vk.Pipeline(vk.NULL_HANDLE)

	vk.DestroyPipelineLayout(vp.ctx.Device, vp.pipelineLayout, nil)
	vp.pipelineLayout = vk.PipelineLayout(vk.NULL_HANDLE)

	// vk.DestroyShaderModule(app.ctx.Device, app.fragShaderModule, nil)
	// app.fragShaderModule = vk.ShaderModule(vk.NULL_HANDLE)
	// vk.DestroyShaderModule(app.device, app.vertShaderModule, nil)
	// app.vertShaderModule = vk.ShaderModule(vk.NULL_HANDLE)

	vk.DestroyRenderPass(vp.ctx.Device, vp.renderPass, nil)
	vp.renderPass = vk.RenderPass(vk.NULL_HANDLE)
}

func (vp *VulkanPipeline) createShaderModule(filename string) vk.ShaderModule {
	smCI := vk.ShaderModuleCreateInfo{
		CodeSize: 0,
		PCode:    new(uint32),
	}

	if dat, err := os.ReadFile(filename); err != nil {
		panic("Failed to read shader file " + filename + ": " + err.Error())
	} else {
		smCI.CodeSize = uintptr(len(dat))
		smCI.PCode = (*uint32)(unsafe.Pointer(&dat[0]))
	}

	if mod, err := vk.CreateShaderModule(vp.ctx.Device, &smCI, nil); err != nil {
		panic(err)
	} else {
		return mod
	}
}
