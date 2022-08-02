package departures

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeepDirection(t *testing.T) {
	assert := assert.New(t)
	assert.True(keepDirection(DirectionTypeForward, DirectionTypeForward))
	assert.False(keepDirection(DirectionTypeForward, DirectionTypeBackward))
	assert.True(keepDirection(DirectionTypeForward, DirectionTypeBoth))

	assert.False(keepDirection(DirectionTypeBackward, DirectionTypeForward))
	assert.True(keepDirection(DirectionTypeBackward, DirectionTypeBackward))
	assert.True(keepDirection(DirectionTypeBackward, DirectionTypeBoth))

	assert.True(keepDirection(DirectionTypeUnknown, DirectionTypeBackward))
	assert.True(keepDirection(DirectionTypeUnknown, DirectionTypeForward))
	assert.True(keepDirection(DirectionTypeUnknown, DirectionTypeBoth))
}
