package yellowstone

import (
	"unsafe"

	"github.com/bbredesen/go-vk"
	"github.com/go-gl/mathgl/mgl32"
)

type Vertex struct {
	Pos   mgl32.Vec2
	Color mgl32.Vec3
}

func (v *Vertex) getBindingDescription() vk.VertexInputBindingDescription {
	descr := vk.VertexInputBindingDescription{
		Binding:   0,
		Stride:    uint32(unsafe.Sizeof(Vertex{})),
		InputRate: vk.VERTEX_INPUT_RATE_VERTEX,
	}

	return descr
}

func (v *Vertex) getAttributeDescriptions() [2]vk.VertexInputAttributeDescription {
	descrs := [2]vk.VertexInputAttributeDescription{
		{
			Binding:  0,
			Location: 0,
			Format:   vk.FORMAT_R32G32_SFLOAT,
			Offset:   uint32(unsafe.Offsetof(Vertex{}.Pos)),
		},
		{
			Binding:  0,
			Location: 1,
			Format:   vk.FORMAT_R32G32B32_SFLOAT,
			Offset:   uint32(unsafe.Offsetof(Vertex{}.Color)),
		},
	}

	return descrs
}
