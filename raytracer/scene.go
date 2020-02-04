package raytracer

import (
	"encoding/json"
	"image"
	_ "image/jpeg" // fuck you go-linter
	_ "image/png"  // fuck you go-linter
	"io/ioutil"
	"log"
	"os"
	"time"
)

// Light -
type Light struct {
	Position      Vector  `json:"position"`
	Color         Vector  `json:"color"`
	Active        bool    `json:"active"`
	LightStrength float64 `json:"light_strength"`
}

// Observer -
type Observer struct {
	Position    Vector  `json:"position"`
	Target      Vector  `json:"target"`
	Up          Vector  `json:"up"`
	Fov         float64 `json:"fov"`
	AspectRatio float64 `json:"aspect_ratio"`
	Zoom        float64 `json:"zoom"`
	Near        float64 `json:"near"`
	Far         float64 `json:"far"`
	Perspective bool    `json:"perspective"`
	Projection  *Matrix `json:"projection"`
	view        Matrix
	width       int
	height      int
}

// Stats of the scene
type Stats struct {
	NumberOfVertices  int64
	NumberOfMaterials int64
	NumberOfIndices   int64
	NumberOfTriangles int64
}

// PixelStorage to Store pixel information before turning it into a png
// we need to do this for post-processing.
type PixelStorage struct {
	DirectLightEnergy    Vector
	AmbientOcclusionRate float64
	Color                Vector
	AmbientColor         Vector
	TotalLight           Vector
	Depth                float64
	X                    int
	Y                    int
}

// Scene -
type Scene struct {
	Objects        map[string]*Object `json:"objects"`
	Lights         []Light            `json:"lights"`
	Observers      []Observer         `json:"observers"`
	ImageMap       map[string]image.Image
	Stats          Stats
	Pixels         [][]PixelStorage
	Width          int
	Height         int
	ShortRadius    float64
	OpenScene      bool
	Config         Config
	OutputFilename string
}

// LoadConfig file for the render
func (s *Scene) LoadConfig(jsonFile string) error {
	var config Config
	log.Printf("Loading configuration from %s", jsonFile)
	file, err := ioutil.ReadFile(jsonFile)
	if err != nil {
		log.Fatalf("Error while reading file: %s", err.Error())
		return err
	}
	log.Printf("Unmarshal JSON\n")
	err = json.Unmarshal(file, &config)
	if err != nil {
		log.Fatalf("Error unmarshalling %s", err.Error())
		return err
	}
	s.Config = config
	return nil
}

// LoadJSON -
func (s *Scene) LoadJSON(jsonFile string) error {
	start := time.Now()
	log.Printf("Loading file: %s\n", jsonFile)
	file, err := ioutil.ReadFile(jsonFile)
	if err != nil {
		log.Fatalf("Error while reading file: %s", err.Error())
		return err
	}
	log.Printf("Unmarshal JSON\n")
	err = json.Unmarshal(file, &s)
	if err != nil {
		log.Fatalf("Error unmarshalling %s", err.Error())
		return err
	}
	log.Printf("Fixing object Ws\n")
	for name, obj := range s.Objects {
		fixObjectVectorW(obj)
		obj.calcRadius()
		s.Objects[name] = obj
	}

	// Order of below calls is important!
	s.flatten()

	s.processObjects()
	s.parseMaterials()
	s.fixLightPos()
	s.ambientOcclusion()
	s.calcStats()
	s.loadLights()
	log.Printf("Loaded scene in %f seconds\n", time.Since(start).Seconds())
	return nil
}

func (s *Scene) loadLights() {
	for k := range s.Objects {
		for i := range s.Objects[k].Triangles {
			if !s.Objects[k].Triangles[i].Material.Light {
				continue
			}
			mat := s.Objects[k].Triangles[i].Material
			lights := sampleTriangle(s.Objects[k].Triangles[i], s.Config.SamplerLimit)
			for li := range lights {
				light := Light{
					Position:      lights[li],
					Color:         mat.Color,
					Active:        true,
					LightStrength: s.Objects[k].Triangles[i].Material.LightStrength * 100,
				}
				s.Lights = append(s.Lights, light)
			}
		}
	}
	log.Print("Building photon map")
	if s.Config.CausticsThreshold > 0 {
		buildPhotonMap(s)
	}
}

func (s *Scene) ambientOcclusion() {
	log.Printf("Calculating ambient parameters")
	bb := BoundingBox{}
	for k := range s.Objects {
		obj := s.Objects[k]
		bb.extend(obj.Root.getBoundingBox())
	}
	dia := bb.MaxExtend[0] - bb.MinExtend[0]
	if bb.MaxExtend[1]-bb.MinExtend[1] < dia {
		dia = bb.MaxExtend[1] - bb.MinExtend[1]
	}
	if bb.MaxExtend[2]-bb.MinExtend[2] < dia {
		dia = bb.MaxExtend[2] - bb.MinExtend[2]
	}
	s.ShortRadius = dia / 2.0
	cast := raycastSceneIntersect(s, s.Observers[0].Position, s.Observers[0].Up)
	s.OpenScene = !cast.Hit
	log.Printf("Ambient max radius: %f", s.ShortRadius)
	if s.OpenScene {
		log.Print("Exterior Scene")
	} else {
		log.Print("Interior Scene")
	}
}

// Lights have 0 as w but they are not vectors, they are positions;
// so we need to set them to 1.0
func (s *Scene) fixLightPos() {
	for i := range s.Lights {
		s.Lights[i].Position[3] = 1.0
	}
}

// Parse all material images and store them in scene object
// so we won't have to open and read for each pixel.
// TODO: Free material image if it is not being used.
// TODO: This method is complex and has more than one responsibility
// NOTE: This function assumes that objects are already flattened!
func (s *Scene) parseMaterials() {
	log.Printf("Parse material textures\n")
	s.ImageMap = make(map[string]image.Image)
	for k := range s.Objects {
		for m := range s.Objects[k].Materials {
			mat := s.Objects[k].Materials[m]
			if _, ok := s.ImageMap[mat.Texture]; ok {
				continue
			}
			if mat.Texture != "" {
				inFile, err := os.Open(mat.Texture)
				if err != nil {
					log.Printf("Material texture [%s] can't be opened for material [%s]\n", mat.Texture, m)
					inFile.Close()
					continue
				}
				src, _, err := image.Decode(inFile)
				if err != nil {
					log.Printf("Error reading image file [%s]: [%s]\n", mat.Texture, err.Error())
					inFile.Close()
					continue
				}
				s.ImageMap[mat.Texture] = src
				inFile.Close()
			}
		}
	}
}

func (s *Scene) flatten() {
	log.Printf("Flatten Scene Objects\n")
	s.Objects = flattenSceneObjects(s.Objects)
}

func (s *Scene) calcStats() {
	nov := int(0)
	noi := int(0)
	nom := int(0)
	not := int(0)
	for k := range s.Objects {
		nov += len(s.Objects[k].Vertices)
		for m := range s.Objects[k].Materials {
			noi += len(s.Objects[k].Materials[m].Indices)
			nom++
		}
		not += len(s.Objects[k].Triangles)
	}
	s.Stats.NumberOfVertices = int64(nov)
	s.Stats.NumberOfIndices = int64(noi)
	s.Stats.NumberOfMaterials = int64(nom)
	s.Stats.NumberOfTriangles = int64(not)
	log.Printf("Number of vertices: %d\n", nov)
	log.Printf("Number of indices: %d\n", noi)
	log.Printf("Number of materials: %d\n", nom)
	log.Printf("Number of triangles: %d\n", not)
}

// Flatten Scene Objects and move them to root
// So, we won't have to multiply matrices each time
func flattenSceneObjects(objects map[string]*Object) map[string]*Object {
	result := make(map[string]*Object)
	for k := range objects {
		result[k] = objects[k]
		if len(objects[k].Children) > 0 {
			flatList := flattenSceneObjects(objects[k].Children)
			for subKey := range flatList {
				subObj := flatList[subKey]
				subObj.Matrix = multiplyMatrix(subObj.Matrix, objects[k].Matrix)
				result[k+subKey] = subObj
			}
		}
	}
	return result
}

// TODO: This is a bit heavy, refactor
func (s *Scene) processObjects() {
	log.Printf("Transform object vertices to absolute and build KDTrees")

	for k := range s.Objects {
		log.Printf("Prepare object %s", k)
		obj := s.Objects[k]
		log.Printf("Local to absolute")
		absoluteVertices := localToAbsoluteList(obj.Vertices, obj.Matrix)
		for i := 0; i < len(absoluteVertices); i++ {
			obj.Vertices[i] = absoluteVertices[i]
		}
		log.Printf("Unify triangles")
		obj.UnifyTriangles()
		log.Printf("Build KDTree")
		obj.KDTree()
		log.Printf("Built %d nodes with %d max depth, object ready", totalNodes, maxDepth)
		totalNodes = 0
		maxDepth = 0
		s.Objects[k] = obj
	}
}
