package bitmap

import (
	"bytes"
	"testing"

	"gotest.tools/assert"
)

func test(t *testing.T, api API) {
	for i := uint64(0); i < 4099; i++ {
		api.Add(i)
	}
	for i := uint64(0xFFFFFFFF); i < 0xFFFFFFFF+1000; i++ {
		api.Add(i)
	}

	for i := 0; i < 4099; i++ {
		assert.Equal(t, api.Contains(uint64(i)), true)
	}
	for i := uint64(0xFFFFFFFF); i < 0xFFFFFFFF+1000; i++ {
		assert.Equal(t, api.Contains(uint64(i)), true)
	}
	assert.Equal(t, api.Len(), uint64(5099))
	assert.Equal(t, api.IsEmpty(), false)

	for i := uint64(0); i < 4099; i += 2 {
		api.Remove(i)
	}
	for i := uint64(0xFFFFFFFF); i < 0xFFFFFFFF+1000; i += 2 {
		api.Remove(i)
	}

	for i := uint64(0); i < 4099; i++ {
		if i%2 == 0 {
			assert.Equal(t, api.Contains(i), false)
		} else {
			assert.Equal(t, api.Contains(i), true)
		}
	}
	for i := uint64(0xFFFFFFFF); i < 0xFFFFFFFF+1000; i += 2 {
		if i%2 != 0 {
			assert.Equal(t, api.Contains(i), false)
		} else {
			assert.Equal(t, api.Contains(i), true)
		}
	}

	buf := &bytes.Buffer{}
	len, err := api.WriteTo(buf)
	assert.Equal(t, err, nil)
	assert.Equal(t, len > 0, true)

	for i := uint64(0); i < 4099; i++ {
		api.Remove(i)
	}

	_, err = api.ReadFrom(buf)
	assert.Equal(t, err, nil)

	for i := uint64(0); i < 4099; i++ {
		if i%2 == 0 {
			assert.Equal(t, api.Contains(i), false)
		} else {
			assert.Equal(t, api.Contains(i), true)
		}
	}
}

func Test_Bitset(t *testing.T) {
	api, _ := New(EngineBitset)
	test(t, api)
}

func Test_Roaring(t *testing.T) {
	api, _ := New(EngineRoaring)
	test(t, api)
}

func Test_NoEngine(t *testing.T) {
	_, err := New("xxx")
	assert.Error(t, err, "no such engine:xxx")
}
