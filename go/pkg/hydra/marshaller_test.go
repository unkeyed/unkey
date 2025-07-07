package hydra

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONMarshaller(t *testing.T) {
	marshaller := NewJSONMarshaller()

	t.Run("MarshalUnmarshalStruct", func(t *testing.T) {
		type TestStruct struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}

		original := TestStruct{Name: "test", Value: 42}

		// Marshal
		data, err := marshaller.Marshal(original)
		require.NoError(t, err)
		assert.Contains(t, string(data), "test")
		assert.Contains(t, string(data), "42")

		// Unmarshal
		var result TestStruct
		err = marshaller.Unmarshal(data, &result)
		require.NoError(t, err)
		assert.Equal(t, original, result)
	})

	t.Run("MarshalUnmarshalPrimitive", func(t *testing.T) {
		original := "hello world"

		data, err := marshaller.Marshal(original)
		require.NoError(t, err)

		var result string
		err = marshaller.Unmarshal(data, &result)
		require.NoError(t, err)
		assert.Equal(t, original, result)
	})

	t.Run("MarshalUnmarshalMap", func(t *testing.T) {
		original := map[string]interface{}{
			"key1": "value1",
			"key2": 123,
			"key3": true,
		}

		data, err := marshaller.Marshal(original)
		require.NoError(t, err)

		var result map[string]interface{}
		err = marshaller.Unmarshal(data, &result)
		require.NoError(t, err)
		assert.Equal(t, "value1", result["key1"])
		// Note: JSON unmarshaling converts numbers to float64
		assert.Equal(t, float64(123), result["key2"])
		assert.Equal(t, true, result["key3"])
	})
}
