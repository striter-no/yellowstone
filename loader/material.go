package assets

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Material for MTL file
type Material struct {
	Name    string
	Ns      float32    // Shininess (specular exponent)
	Ka      [3]float32 // Ambient color
	Kd      [3]float32 // Diffuse color
	Ks      [3]float32 // Specular color
	Ke      [3]float32 // Emissive color
	Ni      float32    // Optical density (index of refraction)
	D       float32    // Dissolve (1 = 0 alpha, 0 = 255 alpha)
	Illum   int        // Illumination model
	BumpMul float32

	// --- PBR-exts ---
	Pr  float32 // Roughness (0..1)
	Pm  float32 // Metallic (0..1)
	Pc  float32 // Clearcoat (0..1)
	Pcr float32 // Clearcoat roughness

	// --- Textures ---
	MapKd     string // Diffuse
	MapKs     string // Specular
	MapKe     string // Emissive
	MapNs     string // Shininess/roughness (legacy)
	MapD      string // Alpha/dissolve
	MapRefl   string // Reflection
	MapBump   string // Bump/normal map
	MapNormal string // Normal map

	// --- PBR-карты (exts) ---
	MapPr        string // Roughness map  ("map_Pr", "map_roughness")
	MapPm        string // Metallic map   ("map_Pm", "map_metallic")
	MapPc        string // Clearcoat map  ("map_Pc")
	MapKEmissive string
}

func LoadMTL(filepath string) (map[string]Material, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть MTL-файл %s: %w", filepath, err)
	}
	defer file.Close()

	materials := make(map[string]Material)

	var curName string
	var curMat Material

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
		case "newmtl":

			if curName != "" {
				materials[curName] = curMat
			}

			if len(parts) >= 2 {
				curName = strings.Join(parts[1:], " ")

				if len(parts) >= 2 {
					curName = strings.Join(parts[1:], " ")
					curMat = Material{
						Name:  curName,
						Ns:    250,
						Ka:    [3]float32{-1, -1, -1},
						Kd:    [3]float32{-1, -1, -1},
						Ks:    [3]float32{-1, -1, -1},
						Ke:    [3]float32{-1, -1, -1},
						Ni:    1.5,
						D:     1,
						Illum: 2,
						Pr:    0.5,
						Pm:    0.0,
						Pc:    0.0,
					}
				}
			}

		case "Ns":
			if curName != "" && len(parts) >= 2 {
				curMat.Ns, _ = parseFloat32(parts[1])
			}
		case "Ka":
			if curName != "" && len(parts) >= 4 {
				curMat.Ka[0], _ = parseFloat32(parts[1])
				curMat.Ka[1], _ = parseFloat32(parts[2])
				curMat.Ka[2], _ = parseFloat32(parts[3])
			}
		case "Kd":
			if curName != "" && len(parts) >= 4 {
				curMat.Kd[0], _ = parseFloat32(parts[1])
				curMat.Kd[1], _ = parseFloat32(parts[2])
				curMat.Kd[2], _ = parseFloat32(parts[3])
			}
		case "Ks":
			if curName != "" && len(parts) >= 4 {
				curMat.Ks[0], _ = parseFloat32(parts[1])
				curMat.Ks[1], _ = parseFloat32(parts[2])
				curMat.Ks[2], _ = parseFloat32(parts[3])
			}
		case "Ke":
			if curName != "" && len(parts) >= 4 {
				curMat.Ke[0], _ = parseFloat32(parts[1])
				curMat.Ke[1], _ = parseFloat32(parts[2])
				curMat.Ke[2], _ = parseFloat32(parts[3])
			}
		case "Ni":
			if curName != "" && len(parts) >= 2 {
				curMat.Ni, _ = parseFloat32(parts[1])
			}
		case "d":
			if curName != "" && len(parts) >= 2 {
				curMat.D, _ = parseFloat32(parts[1])
			}
		case "Tr":
			if curName != "" && len(parts) >= 2 {
				tr, _ := parseFloat32(parts[1])
				curMat.D = 1.0 - tr
			}
		case "illum":
			if curName != "" && len(parts) >= 2 {
				curMat.Illum, _ = strconv.Atoi(parts[1])
			}
		case "map_Kd":
			if curName != "" && len(parts) >= 2 {
				curMat.MapKd = mapTexPath(parts[1:])
			}
		case "map_Ks":
			if curName != "" && len(parts) >= 2 {
				curMat.MapKs = mapTexPath(parts[1:])
			}
		case "map_Ke":
			if curName != "" && len(parts) >= 2 {
				curMat.MapKe = mapTexPath(parts[1:])
			}
		case "map_Ns":
			if curName != "" && len(parts) >= 2 {
				curMat.MapNs = mapTexPath(parts[1:])
			}
		case "map_d":
			if curName != "" && len(parts) >= 2 {
				curMat.MapD = mapTexPath(parts[1:])
			}
		case "map_refl":
			if curName != "" && len(parts) >= 2 {
				curMat.MapRefl = mapTexPath(parts[1:])
			}
		case "map_Bump", "bump":
			if curName != "" && len(parts) >= 2 {
				bumpMul := float32(1.0)
				texParts := parts[1:]
				for i := 0; i < len(texParts); i++ {
					if texParts[i] == "-bm" && i+1 < len(texParts) {
						bumpMul, _ = parseFloat32(texParts[i+1])
						texParts = append(texParts[:i], texParts[i+2:]...)
						break
					}
				}
				curMat.BumpMul = bumpMul
				if len(texParts) > 0 {
					curMat.MapBump = mapTexPath(texParts)
				}
			}
		case "map_normal":
			if curName != "" && len(parts) >= 2 {
				texParts := parts[1:]
				for i := 0; i < len(texParts); i++ {
					if texParts[i] == "-bm" && i+1 < len(texParts) {
						mul, _ := parseFloat32(texParts[i+1])
						curMat.BumpMul = mul
						texParts = append(texParts[:i], texParts[i+2:]...)
						break
					}
				}
				if len(texParts) > 0 {
					curMat.MapNormal = mapTexPath(texParts)
				}
			}
		case "Pr", "pr":
			if curName != "" && len(parts) >= 2 {
				curMat.Pr, _ = parseFloat32(parts[1])
			}
		case "Pm", "pm":
			if curName != "" && len(parts) >= 2 {
				curMat.Pm, _ = parseFloat32(parts[1])
			}
		case "Pc", "pc":
			if curName != "" && len(parts) >= 2 {
				curMat.Pc, _ = parseFloat32(parts[1])
			}
		case "Pcr", "pcr":
			if curName != "" && len(parts) >= 2 {
				curMat.Pcr, _ = parseFloat32(parts[1])
			}
		case "map_Pr", "map_roughness":
			if curName != "" && len(parts) >= 2 {
				curMat.MapPr = mapTexPath(parts[1:])
			}
		case "map_Pm", "map_metallic":
			if curName != "" && len(parts) >= 2 {
				curMat.MapPm = mapTexPath(parts[1:])
			}
		case "map_Pc":
			if curName != "" && len(parts) >= 2 {
				curMat.MapPc = mapTexPath(parts[1:])
			}
		}
	}

	if curName != "" {
		materials[curName] = curMat
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read MTL file %s: %w", filepath, err)
	}

	return materials, nil
}
func mapTexPath(tokens []string) string {
	if len(tokens) == 0 {
		return ""
	}

	flags := map[string]bool{
		"-bm": true, "-clamp": true, "-blendu": true, "-blendv": true,
		"-cc": true, "-imfchan": true, "-type": true, "-o": true,
		"-s": true, "-t": true, "-mm": true, "-texres": true,
	}

	var pathTokens []string
	for i := 0; i < len(tokens); i++ {
		if flags[tokens[i]] {
			if i+1 < len(tokens) && !strings.HasPrefix(tokens[i+1], "-") {
				i++
			}
			continue
		}

		if strings.HasPrefix(tokens[i], "-") {
			if i+1 < len(tokens) && !strings.HasPrefix(tokens[i+1], "-") {
				i++
			}
			continue
		}

		pathTokens = append(pathTokens, tokens[i])
	}

	return strings.Join(pathTokens, " ")
}

func parseFloat32(s string) (float32, error) {
	v, err := strconv.ParseFloat(s, 32)
	if err != nil {
		return 0, err
	}
	return float32(v), nil
}
