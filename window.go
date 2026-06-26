package yellowstone

import (
	"unsafe"

	"github.com/bbredesen/go-vk"
	"github.com/go-gl/glfw/v3.3/glfw"
)

type WindowConfig struct {
	Width, Height int
	Title         string
	Resizable     bool
}

type Window struct {
	Width, Height int
	Resizable     bool
	Title         string

	// -- private

	glfwWindow *glfw.Window
	surface    vk.SurfaceKHR
}

func (w *Window) SetupWindow() error {
	if err := glfw.Init(); err != nil {
		return err
	}

	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	if w.Resizable {
		glfw.WindowHint(glfw.Resizable, glfw.True)
	} else {
		glfw.WindowHint(glfw.Resizable, glfw.False)
	}

	window, err := glfw.CreateWindow(w.Width, w.Height, w.Title, nil, nil)
	if err != nil {
		return err
	}

	w.glfwWindow = window
	w.surface = vk.SurfaceKHR(0)

	return nil
}

func (w *Window) IsOpen() bool {
	return !w.glfwWindow.ShouldClose()
}

func (w *Window) PollEvents() {
	glfw.PollEvents()
}

func (w *Window) Destroy() {
	w.glfwWindow.Destroy()
	glfw.Terminate()
}

// -- internal

func (w *Window) createSurface(vkInstance vk.Instance) error {
	instance_GLFW := (*struct{})(unsafe.Pointer(uintptr(vkInstance)))

	surfaceHandlePtr, err := w.glfwWindow.CreateWindowSurface(instance_GLFW, nil)
	if err != nil {
		return err
	}

	w.surface = *(*vk.SurfaceKHR)(unsafe.Pointer(surfaceHandlePtr))
	return nil
}
