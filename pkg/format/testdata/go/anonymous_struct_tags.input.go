package alignment

type envelope struct {
Short string `json:"short"`
Payload struct {
Name string `json:"name"`
} `json:"payload"` // payload
Geometry struct {
Type string `json:"type"`
Coordinates [][2]float64 `json:"coordinates"`
} `json:"geometry"`
LongField int `json:"long"`
}

type report struct {
Tests []struct {
Name string `json:"name"`
Expected string `json:"expected"`
} `json:"tests"`
Count int `json:"count"`
}
