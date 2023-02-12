package helpers_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yangl1996/nesthub/internal/helpers"
)

func TestErrorListToErr(t *testing.T) {
	t.Parallel()

	t.Run("No errors in list", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, nil, helpers.ErrListToErr("", []error{}))
	})

	t.Run("Errors present", func(t *testing.T) {
		t.Parallel()
		prefix := "poke"
		errs := []error{
			errors.New("me"),
			nil,
			errors.New("you"),
		}
		assert.Equal(t, "poke: me, you", helpers.ErrListToErr(prefix, errs).Error())
	})
}
