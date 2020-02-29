package raytracer

/*
Light related methods
*/

import (
	"math"
)

func calculateDirectionalLight(scene *Scene, intersection *IntersectionTriangle, light *Light, depth int) (result Vector) {
	var shortestIntersection IntersectionTriangle
	if !intersection.Hit {
		return
	}

	lightD := scaleVector(light.Direction, -1)

	dotP := dot(intersection.IntersectionNormal, lightD)
	if dotP < 0 {
		return
	}

	samples := sampleSphere(0.2, GlobalConfig.LightSampleCount)
	totalHits := 0.0
	totalLight := Vector{}

	for i := range samples {
		rayStart := scaleVector(lightD, 999999999999)
		rayStart = addVector(rayStart, intersection.Intersection)
		rayStart = addVector(rayStart, samples[i])
		dir := normalizeVector(subVector(intersection.Intersection, rayStart))

		shortestIntersection = raycastSceneIntersect(scene, rayStart, dir)
		if (shortestIntersection.Triangle != nil && shortestIntersection.Triangle.id == intersection.Triangle.id) || shortestIntersection.Dist < DIFF {
			if !sameSideTest(intersection.IntersectionNormal, shortestIntersection.IntersectionNormal, 0) {
				return
			}

			intensity := dotP * light.LightStrength
			intensity *= GlobalConfig.Exposure

			totalLight = addVector(totalLight, Vector{
				light.Color[0] * intensity,
				light.Color[1] * intensity,
				light.Color[2] * intensity,
				intensity,
			})
			totalHits += 1.0
		}
	}
	if totalHits > 0 {
		return scaleVector(totalLight, totalHits/float64(GlobalConfig.LightSampleCount))
	}

	return
}

// Calculate light for given light source.
// Result will be used to calculate "avarage" of the pixel color
func calculateLight(scene *Scene, intersection *IntersectionTriangle, light *Light, depth int) (result Vector) {
	var shortestIntersection IntersectionTriangle

	if !intersection.Hit {
		return
	}

	l1 := normalizeVector(subVector(light.Position, intersection.Intersection))
	l2 := intersection.IntersectionNormal
	dotP := dot(l2, l1)
	if dotP < 0 {
		return
	}

	if intersection.Triangle.Material.Light {
		if intersection.Triangle.Material.LightStrength == 0 {
			intersection.Triangle.Material.LightStrength = light.LightStrength
		}
		return Vector{
			GlobalConfig.Exposure * light.Color[0] * intersection.Triangle.Material.LightStrength,
			GlobalConfig.Exposure * light.Color[1] * intersection.Triangle.Material.LightStrength,
			GlobalConfig.Exposure * light.Color[2] * intersection.Triangle.Material.LightStrength,
			1,
		}
	}

	rayDir := normalizeVector(subVector(intersection.Intersection, light.Position))
	rayStart := light.Position
	rayLength := vectorDistance(intersection.Intersection, light.Position)

	shortestIntersection = raycastSceneIntersect(scene, rayStart, rayDir)
	s := math.Abs(rayLength - shortestIntersection.Dist)

	if (shortestIntersection.Triangle != nil && shortestIntersection.Triangle.id == intersection.Triangle.id) || s < DIFF {
		if !sameSideTest(intersection.IntersectionNormal, shortestIntersection.IntersectionNormal, 0) {
			return
		}

		intensity := (1 / (rayLength * rayLength)) * GlobalConfig.Exposure
		intensity *= dotP * light.LightStrength

		if intersection.Triangle.Material.LightStrength > 0 {
			intensity = intersection.Triangle.Material.LightStrength * GlobalConfig.Exposure
		}

		return Vector{
			light.Color[0] * intensity,
			light.Color[1] * intensity,
			light.Color[2] * intensity,
			intensity,
		}
	}
	return
}

func calculateTotalLight(scene *Scene, intersection *IntersectionTriangle, depth int) (result Vector) {
	if (!intersection.Hit) || (depth >= GlobalConfig.MaxReflectionDepth) {
		return
	}

	if intersection.Triangle.Material.Light {
		c := scaleVector(intersection.Triangle.Material.Color, intersection.Triangle.Material.LightStrength)
		return c
	}

	lightChan := make(chan Vector, len(scene.Lights))

	for i := range scene.Lights {
		go func(scene *Scene, intersection *IntersectionTriangle, light *Light, depth int, lightChan chan Vector) {
			if light.Directional {
				lightChan <- calculateDirectionalLight(scene, intersection, light, depth)
			} else {
				lightChan <- calculateLight(scene, intersection, light, depth)
			}
		}(scene, intersection, &scene.Lights[i], depth, lightChan)
	}

	result = Vector{}
	for i := 0; i < len(scene.Lights); i++ {
		light := <-lightChan
		if light[3] > 0 {
			result = addVector(result, light)
		}
	}

	if GlobalConfig.PhotonSpacing > 0 && GlobalConfig.RenderCaustics {
		if intersection.Triangle.Photons != nil && len(intersection.Triangle.Photons) > 0 {
			for i := range intersection.Triangle.Photons {
				if vectorDistance(intersection.Triangle.Photons[i].Location, intersection.Intersection) < GlobalConfig.PhotonSpacing {
					c := scaleVector(intersection.Triangle.Photons[i].Color, GlobalConfig.Exposure)
					result = addVector(result, c)
				}
			}
		}
	}

	return result
}
