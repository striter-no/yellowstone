package yellowstone

import (
	"unsafe"

	"github.com/go-gl/mathgl/mgl32"
)

type UniformBufferObject struct {
	Foo   mgl32.Vec2
	Model mgl32.Mat4
	View  mgl32.Mat4
	Proj  mgl32.Mat4
}

func getUBOBytes(ubo *UniformBufferObject) []byte {
	return (*[unsafe.Sizeof(*ubo)]byte)(unsafe.Pointer(ubo))[:]
}
