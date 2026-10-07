package sets_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"newscid/pkg/sets"
)

func TestPieceSetsAndThemesCatalog(t *testing.T) {
	pieces := sets.GetAvailablePieceSets()
	assert.NotEmpty(t, pieces)

	// Verify Merida and Alpha are listed as native
	foundMerida := false
	foundAlpha := false
	for _, p := range pieces {
		if p.ID == "merida" {
			foundMerida = true
			assert.True(t, p.IsScidNative)
		}
		if p.ID == "alpha" {
			foundAlpha = true
			assert.True(t, p.IsScidNative)
		}
	}
	assert.True(t, foundMerida)
	assert.True(t, foundAlpha)

	themes := sets.GetAvailableBoardThemes()
	assert.NotEmpty(t, themes)

	// Verify Classic Wood and Blue-White
	foundWood := false
	for _, th := range themes {
		if th.ID == "classic-wood" {
			foundWood = true
			assert.NotEmpty(t, th.Light)
			assert.NotEmpty(t, th.Dark)
		}
	}
	assert.True(t, foundWood)
}
