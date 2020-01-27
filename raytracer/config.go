package raytracer

// Config keeps Raytracer Configuration
type Config struct {
	SamplerLimit             int     `json:"sampler_limit"`
	LightHardLimit           float64 `json:"light_distance"`
	MaxReflectionDepth       int     `json:"max_reflection_depth"`
	RayCorrection            float64 `json:"ray_correction"`
	OcclusionRate            float64 `json:"occlusion_rate"`
	AmbientRadius            float64 `json:"ambient_occlusion_radius"`
	RenderOcclusion          bool    `json:"render_occlusion"`
	RenderLights             bool    `json:"render_lights"`
	RenderColors             bool    `json:"render_colors"`
	RenderAmbientColors      bool    `json:"render_ambient_color"`
	AmbientColorSharingRatio float64 `json:"ambient_color_ratio"`
	RenderReflections        bool    `json:"render_reflections"`
	Width                    int     `json:"width"`
	Height                   int     `json:"height"`
	EdgeDetechThreshold      float64 `json:"edge_detect_threshold"`
	Profiling                bool    `json:"profiling"`
}
