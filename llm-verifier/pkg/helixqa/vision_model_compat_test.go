package helixqa

import "testing"

// TestVisionModelAliasKeepsStubFieldNames fails to COMPILE if the VisionModel
// alias is removed or loses a field the earlier stub type exposed.
func TestVisionModelAliasKeepsStubFieldNames(t *testing.T) {
	m := VisionModel{
		Provider:         "p",
		Model:            "m",
		QualityScore:     0.5,
		ReliabilityScore: 0.6,
		InputCostPer1k:   0.1,
		OutputCostPer1k:  0.2,
		AvgLatencyMs:     7,
	}
	if m.Provider != "p" || m.Model != "m" || m.QualityScore != 0.5 ||
		m.ReliabilityScore != 0.6 || m.InputCostPer1k != 0.1 ||
		m.OutputCostPer1k != 0.2 || m.AvgLatencyMs != 7 {
		t.Fatalf("field round-trip failed: %+v", m)
	}
	var _ []VisionModel = VisionModelRegistry()
	if len(VisionModelRegistry()) == 0 {
		t.Fatal("empty registry")
	}
}
