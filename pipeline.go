package yellowstone

import (
	"github.com/bbredesen/go-vk"
)

type Pipeline struct {
	renderPass vk.RenderPass
	layout     vk.PipelineLayout
	handle     vk.Pipeline

	Device *VulkanDevice
}

func (p *Pipeline) SetupPipeline(
	vertShaderCompiled,
	fragShaderCompiled string,
	swapchain *Swapchain,
) error {
	if err := p.createRenderPass(swapchain.imageFormat, p.Device.logical); err != nil {
		return err
	}

	if err := p.createGraphicsPipeline(vertShaderCompiled, fragShaderCompiled, p.Device.logical, swapchain.extent); err != nil {
		return err
	}

	return nil
}

func (p *Pipeline) createRenderPass(imgLayout vk.Format, logicalDev vk.Device) error {
	colorAttachment := vk.AttachmentDescription{
		Format:        imgLayout,
		Samples:       vk.SAMPLE_COUNT_1_BIT,
		LoadOp:        vk.ATTACHMENT_LOAD_OP_CLEAR,
		StoreOp:       vk.ATTACHMENT_STORE_OP_STORE,
		InitialLayout: vk.IMAGE_LAYOUT_UNDEFINED,
		FinalLayout:   vk.IMAGE_LAYOUT_PRESENT_SRC_KHR,
	}

	colorAttachmentRef := vk.AttachmentReference{
		Attachment: 0,
		Layout:     vk.IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL,
	}

	subpass := vk.SubpassDescription{
		PipelineBindPoint: vk.PIPELINE_BIND_POINT_GRAPHICS,
		PColorAttachments: []vk.AttachmentReference{colorAttachmentRef},
	}

	dependency := vk.SubpassDependency{
		SrcSubpass:    vk.SUBPASS_EXTERNAL,
		DstSubpass:    0,
		SrcStageMask:  vk.PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT,
		SrcAccessMask: 0,
		DstStageMask:  vk.PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT,
		DstAccessMask: vk.ACCESS_COLOR_ATTACHMENT_WRITE_BIT,
	}

	renderPassInfo := vk.RenderPassCreateInfo{
		PAttachments:  []vk.AttachmentDescription{colorAttachment},
		PSubpasses:    []vk.SubpassDescription{subpass},
		PDependencies: []vk.SubpassDependency{dependency},
	}

	renderPass, err := vk.CreateRenderPass(logicalDev, &renderPassInfo, nil)
	if err != nil {
		return err
	}

	p.renderPass = renderPass
	return nil
}

func (p *Pipeline) createGraphicsPipeline(vertexCompiled, fragmentCompiled string, logicalDev vk.Device, extent vk.Extent2D) error {
	vertShader, err := LoadSPVShader(vertexCompiled)
	if err != nil {
		return err
	}

	fragShader, err := LoadSPVShader(fragmentCompiled)
	if err != nil {
		return err
	}

	vertShaderModule, err := vertShader.CreateShaderModule(logicalDev)
	if err != nil {
		return err
	}

	fragShaderModule, err := fragShader.CreateShaderModule(logicalDev)
	if err != nil {
		return err
	}

	vertShaderStageInfo := vk.PipelineShaderStageCreateInfo{
		Stage:  vk.SHADER_STAGE_VERTEX_BIT,
		Module: vertShaderModule,
		PName:  "main",
	}

	fragShaderStageInfo := vk.PipelineShaderStageCreateInfo{
		Stage:  vk.SHADER_STAGE_FRAGMENT_BIT,
		Module: fragShaderModule,
		PName:  "main",
	}

	shaderStages := []vk.PipelineShaderStageCreateInfo{vertShaderStageInfo, fragShaderStageInfo}

	v := Vertex{}
	bindingDescriptions := v.getBindingDescription()
	attributeDescriptions := v.getAttributeDescriptions()
	vertexInputInfo := vk.PipelineVertexInputStateCreateInfo{}

	vertexInputInfo.PVertexBindingDescriptions = []vk.VertexInputBindingDescription{
		bindingDescriptions,
	}
	vertexInputInfo.PVertexAttributeDescriptions = attributeDescriptions[:]

	inputAssembly := vk.PipelineInputAssemblyStateCreateInfo{
		Topology:               vk.PRIMITIVE_TOPOLOGY_TRIANGLE_LIST,
		PrimitiveRestartEnable: false,
	}

	viewport := vk.Viewport{
		X:        0,
		Y:        0,
		Width:    float32(extent.Width),
		Height:   float32(extent.Height),
		MinDepth: 0,
		MaxDepth: 1,
	}

	scissor := vk.Rect2D{
		Offset: vk.Offset2D{X: 0, Y: 0},
		Extent: extent,
	}

	dynamicStates := []vk.DynamicState{
		vk.DYNAMIC_STATE_VIEWPORT,
		vk.DYNAMIC_STATE_SCISSOR,
	}

	dynamicState := vk.PipelineDynamicStateCreateInfo{
		PDynamicStates: dynamicStates,
	}

	viewportState := vk.PipelineViewportStateCreateInfo{
		PViewports: []vk.Viewport{viewport},
		PScissors:  []vk.Rect2D{scissor},
	}

	rasterizer := vk.PipelineRasterizationStateCreateInfo{
		DepthClampEnable:        false,
		RasterizerDiscardEnable: false,
		PolygonMode:             vk.POLYGON_MODE_FILL,
		CullMode:                vk.CULL_MODE_BACK_BIT,
		FrontFace:               vk.FRONT_FACE_CLOCKWISE,
		DepthBiasEnable:         false,
		DepthBiasConstantFactor: 0.0,
		DepthBiasClamp:          0.0,
		DepthBiasSlopeFactor:    0.0,
		LineWidth:               1.0,
	}

	multisampling := vk.PipelineMultisampleStateCreateInfo{
		SampleShadingEnable:  false,
		RasterizationSamples: vk.SAMPLE_COUNT_1_BIT,
		MinSampleShading:     1.0,
	}

	colorBlendAttachment := vk.PipelineColorBlendAttachmentState{
		ColorWriteMask: vk.COLOR_COMPONENT_R_BIT |
			vk.COLOR_COMPONENT_G_BIT |
			vk.COLOR_COMPONENT_B_BIT |
			vk.COLOR_COMPONENT_A_BIT,
		BlendEnable: false,
	}

	colorBlending := vk.PipelineColorBlendStateCreateInfo{
		LogicOpEnable:  false,
		LogicOp:        vk.LOGIC_OP_COPY,
		PAttachments:   []vk.PipelineColorBlendAttachmentState{colorBlendAttachment},
		BlendConstants: [4]float32{0, 0, 0, 0},
	}

	pipelineLayoutInfo := vk.PipelineLayoutCreateInfo{}
	pipelineLayout, err := vk.CreatePipelineLayout(logicalDev, &pipelineLayoutInfo, nil)
	if err != nil {
		return err
	}

	p.layout = pipelineLayout

	pipelineInfo := vk.GraphicsPipelineCreateInfo{
		PStages:             shaderStages,
		PVertexInputState:   &vertexInputInfo,
		PInputAssemblyState: &inputAssembly,
		PViewportState:      &viewportState,
		PRasterizationState: &rasterizer,
		PMultisampleState:   &multisampling,
		PColorBlendState:    &colorBlending,
		PDynamicState:       &dynamicState,
		Layout:              pipelineLayout,
		RenderPass:          p.renderPass,
		Subpass:             0,
		BasePipelineHandle:  vk.Pipeline(vk.NULL_HANDLE),
		BasePipelineIndex:   -1,
	}

	graphicsPipelines, err := vk.CreateGraphicsPipelines(
		logicalDev,
		vk.PipelineCache(vk.NULL_HANDLE),
		[]vk.GraphicsPipelineCreateInfo{pipelineInfo},
		nil,
	)

	if err != nil {
		return err
	}

	p.handle = graphicsPipelines[0]

	vk.DestroyShaderModule(logicalDev, fragShaderModule, nil)
	vk.DestroyShaderModule(logicalDev, vertShaderModule, nil)
	return nil
}

func (p *Pipeline) Destroy() {
	vk.DestroyPipeline(p.Device.logical, p.handle, nil)
	vk.DestroyPipelineLayout(p.Device.logical, p.layout, nil)
	vk.DestroyRenderPass(p.Device.logical, p.renderPass, nil)
}
