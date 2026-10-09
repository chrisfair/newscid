package version

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetVersionInfo(t *testing.T) {
	info := Get()
	assert.NotEmpty(t, info.Version, "Version should not be empty")
	assert.NotEmpty(t, info.Channel, "Channel should not be empty")
	assert.NotEmpty(t, info.GoVersion, "GoVersion should not be empty")
	assert.NotEmpty(t, info.Platform, "Platform should not be empty")
	assert.NotEmpty(t, info.Compiler, "Compiler should not be empty")

	// Semantic version regex
	semverRegex := regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.-]+)?$`)
	assert.True(t, semverRegex.MatchString(info.Version), "Version %s must match semver format", info.Version)
}

func TestVersionStringFormat(t *testing.T) {
	str := String()
	assert.Contains(t, str, "NewScid v")
	assert.Contains(t, str, "commit:")
	assert.Contains(t, str, "built:")

	short := Short()
	assert.Equal(t, "v"+Version, short)
}
