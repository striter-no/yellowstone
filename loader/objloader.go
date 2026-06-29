package assets

import (
	"bufio"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-gl/mathgl/mgl32"
)

type Vertex struct {
	Pos   mgl32.Vec3
	UV    mgl32.Vec2
	Color mgl32.Vec3
}

type OBJResult struct {
	Meshes        map[string][]Vertex
	MeshMaterials map[string]string
	Materials     map[string]Material
}

func LoadOBJ(path string) (*OBJResult, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть файл %s: %w", path, err)
	}
	defer file.Close()

	var vertices []mgl32.Vec3
	var uvs []mgl32.Vec2
	var normals []mgl32.Vec3

	result := &OBJResult{
		Meshes:        make(map[string][]Vertex),
		MeshMaterials: make(map[string]string),
		Materials:     make(map[string]Material),
	}

	currentObject := "default"
	currentMaterial := ""
	objDir := filepath.Dir(path)

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		switch parts[0] {
		case "mtllib":
			if len(parts) >= 2 {
				mtlFile := strings.Join(parts[1:], " ")
				mtlPath := filepath.Join(objDir, mtlFile)
				mats, err := LoadMTL(mtlPath)
				if err != nil {
					return nil, fmt.Errorf("ошибка загрузки MTL %s: %w", mtlPath, err)
				}
				maps.Copy(result.Materials, mats)
			}

		case "o", "g":
			if len(parts) >= 2 {
				currentObject = strings.Join(parts[1:], "_")
				if _, exists := result.MeshMaterials[currentObject]; !exists {
					result.MeshMaterials[currentObject] = currentMaterial
				}
			}

		case "usemtl":
			if len(parts) >= 2 {
				currentMaterial = strings.Join(parts[1:], " ")
				result.MeshMaterials[currentObject] = currentMaterial
			}

		case "v":
			if len(parts) >= 4 {
				x, _ := strconv.ParseFloat(parts[1], 32)
				y, _ := strconv.ParseFloat(parts[2], 32)
				z, _ := strconv.ParseFloat(parts[3], 32)
				vertices = append(vertices, mgl32.Vec3{float32(x), float32(y), float32(z)})
			}

		case "vt":
			if len(parts) >= 3 {
				u, _ := strconv.ParseFloat(parts[1], 32)
				v, _ := strconv.ParseFloat(parts[2], 32)
				uvs = append(uvs, mgl32.Vec2{float32(u), float32(1.0 - v)})
			}

		case "vn":
			if len(parts) >= 4 {
				nx, _ := strconv.ParseFloat(parts[1], 32)
				ny, _ := strconv.ParseFloat(parts[2], 32)
				nz, _ := strconv.ParseFloat(parts[3], 32)
				normals = append(normals, mgl32.Vec3{float32(nx), float32(ny), float32(nz)})
			}

		case "f":
			if len(parts) < 4 {
				continue
			}

			type faceVert struct {
				v, vt, vn int
			}
			verts := make([]faceVert, 0, len(parts)-1)
			for i := 1; i < len(parts); i++ {
				v, vt, vn := parseFaceVertex(parts[i])
				if v < 0 {
					v = len(vertices) + v + 1
				}
				if vt < 0 {
					vt = len(uvs) + vt + 1
				}
				if vn < 0 {
					vn = len(normals) + vn + 1
				}
				if v < 1 || v > len(vertices) {
					continue
				}
				verts = append(verts, faceVert{v: v, vt: vt, vn: vn})
			}
			if len(verts) < 3 {
				continue
			}

			var baseColor mgl32.Vec4
			if currentMaterial != "" {
				if mat, ok := result.Materials[currentMaterial]; ok {
					r := uint8(mat.Kd[0])
					g := uint8(mat.Kd[1])
					b := uint8(mat.Kd[2])
					a := uint8(mat.D)
					baseColor = mgl32.Vec4{float32(r), float32(g), float32(b), float32(a)}
				} else {
					baseColor = mgl32.Vec4{1, 1, 1, 1}
				}
			} else {
				baseColor = mgl32.Vec4{1, 1, 1, 1}
			}

			resolve := func(fv faceVert) (mgl32.Vec3, mgl32.Vec2, mgl32.Vec3) {
				pos := vertices[fv.v-1]
				var uv mgl32.Vec2
				if fv.vt > 0 && fv.vt <= len(uvs) {
					uv = uvs[fv.vt-1]
				}
				var n mgl32.Vec3
				if fv.vn > 0 && fv.vn <= len(normals) {
					n = normals[fv.vn-1]
				}
				return pos, uv, n
			}

			v0p, v0uv, v0n := resolve(verts[0])
			for i := 1; i+1 < len(verts); i++ {
				vip, viuv, vin := resolve(verts[i])
				vi1p, vi1uv, vi1n := resolve(verts[i+1])

				_ = vin
				_ = vi1n
				_ = v0n

				// tri := render.TBO{
				// 	V0: v0p, V1: vip, V2: vi1p,
				// 	UV0: v0uv, UV1: viuv, UV2: vi1uv,
				// 	N0: v0n, N1: vin, N2: vi1n,
				// 	C0: baseColor, C1: baseColor, C2: baseColor,
				// 	OmniDir: false,
				// }

				V0 := Vertex{
					Pos:   v0p,
					UV:    v0uv,
					Color: baseColor.Vec3(),
				}

				V1 := Vertex{
					Pos:   vip,
					UV:    viuv,
					Color: baseColor.Vec3(),
				}

				V2 := Vertex{
					Pos:   vi1p,
					UV:    vi1uv,
					Color: baseColor.Vec3(),
				}

				result.Meshes[currentObject] = append(result.Meshes[currentObject], V0)
				result.Meshes[currentObject] = append(result.Meshes[currentObject], V1)
				result.Meshes[currentObject] = append(result.Meshes[currentObject], V2)
			}
		}
	}

	if err := scanner.Err(); err != nil {

		return nil, fmt.Errorf("failed to read from file %s: %w", path, err)
	}

	return result, nil
}

func parseFaceVertex(fv string) (vIdx, vtIdx, vnIdx int) {
	parts := strings.Split(fv, "/")
	vIdx, _ = strconv.Atoi(parts[0])

	if len(parts) > 1 && parts[1] != "" {
		vtIdx, _ = strconv.Atoi(parts[1])
	} else {
		vtIdx = 0
	}

	if len(parts) > 2 && parts[2] != "" {
		vnIdx, _ = strconv.Atoi(parts[2])
	} else {
		vnIdx = 0
	}

	return vIdx, vtIdx, vnIdx
}
