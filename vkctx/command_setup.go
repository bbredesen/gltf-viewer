package vkctx

import (
	"github.com/bbredesen/go-vk"
)

// Create command pool, associated command buffers, and record commands to clear
// the screen.
func (ctx *Context) createCommandPool() {
	// 1) Create the command pool
	poolCreateInfo := vk.CommandPoolCreateInfo{
		Flags:            vk.COMMAND_POOL_CREATE_RESET_COMMAND_BUFFER_BIT,
		QueueFamilyIndex: ctx.PresentQueueFamilyIndex,
	}
	commandPool, err := vk.CreateCommandPool(ctx.Device, &poolCreateInfo, nil)
	if err != nil {
		panic("Could not create command pool! " + err.Error())
	}
	ctx.CommandPool = commandPool

	// 2) Allocate primary command buffers, one for each swapchain image, from the pool
	allocInfo := vk.CommandBufferAllocateInfo{
		CommandPool:        ctx.CommandPool,
		Level:              vk.COMMAND_BUFFER_LEVEL_PRIMARY,
		CommandBufferCount: uint32(len(ctx.SwapchainImages)),
	}
	commandBuffers, err := vk.AllocateCommandBuffers(ctx.Device, &allocInfo)
	if err != nil {
		panic("Could not allocate command buffers! " + err.Error())
	}
	ctx.CommandBuffers = commandBuffers
}

func (ctx *Context) destroyCommandPool() {
	vk.FreeCommandBuffers(ctx.Device, ctx.CommandPool, ctx.CommandBuffers)
	vk.DestroyCommandPool(ctx.Device, ctx.CommandPool, nil)
}

func (ctx *Context) createSyncObjects() {
	createInfo := vk.SemaphoreCreateInfo{}

	imgSem, err := vk.CreateSemaphore(ctx.Device, &createInfo, nil)
	if err != nil {
		panic("Could not create semaphore! " + err.Error())
	}
	ctx.ImageAvailableSemaphore = imgSem

	renSem, err := vk.CreateSemaphore(ctx.Device, &createInfo, nil)
	if err != nil {
		panic("Could not create semaphore! " + err.Error())
	}
	ctx.RenderFinishedSemaphore = renSem

	fenceCreateInfo := vk.FenceCreateInfo{
		Flags: vk.FENCE_CREATE_SIGNALED_BIT,
	}
	if ctx.InFlightFence, err = vk.CreateFence(ctx.Device, &fenceCreateInfo, nil); err != nil {
		panic("Could not create fence! " + err.Error())
	}
}

func (app *Context) destroySyncObjects() {
	vk.DestroyFence(app.Device, app.InFlightFence, nil)

	vk.DestroySemaphore(app.Device, app.ImageAvailableSemaphore, nil)
	vk.DestroySemaphore(app.Device, app.RenderFinishedSemaphore, nil)
}

func (app *Context) BeginOneTimeCommands() vk.CommandBuffer {
	bufferAlloc := vk.CommandBufferAllocateInfo{
		CommandPool:        app.CommandPool,
		Level:              vk.COMMAND_BUFFER_LEVEL_PRIMARY,
		CommandBufferCount: 1,
	}

	var err error
	var bufs []vk.CommandBuffer

	if bufs, err = vk.AllocateCommandBuffers(app.Device, &bufferAlloc); err != nil {
		panic("Could not allocate one-time command buffer: " + err.Error())
	}

	cbbInfo := vk.CommandBufferBeginInfo{
		Flags: vk.COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT,
	}

	if err = vk.BeginCommandBuffer(bufs[0], &cbbInfo); err != nil {
		panic("Could not begin recording one-time command buffer: " + err.Error())
	}

	return bufs[0]
}

func (app *Context) EndOneTimeCommands(buf vk.CommandBuffer) {
	if err := vk.EndCommandBuffer(buf); err != nil {
		panic("Could not end one-time command buffer: " + err.Error())
	}

	submitInfo := vk.SubmitInfo{
		PCommandBuffers: []vk.CommandBuffer{buf},
	}

	if err := vk.QueueSubmit(app.GraphicsQueue, []vk.SubmitInfo{submitInfo}, vk.Fence(vk.NULL_HANDLE)); err != nil {
		panic("Could not submit one-time command buffer: " + err.Error())
	}
	if err := vk.QueueWaitIdle(app.GraphicsQueue); err != nil {
		panic("QueueWaitIdle failed: " + err.Error())
	}

	vk.FreeCommandBuffers(app.Device, app.CommandPool, []vk.CommandBuffer{buf})
}
