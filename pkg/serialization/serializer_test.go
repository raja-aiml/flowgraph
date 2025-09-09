package serialization

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestData represents test data structure
type TestData struct {
	ID    string            `json:"id" msgpack:"id"`
	Name  string            `json:"name" msgpack:"name"`
	Data  map[string]string `json:"data" msgpack:"data"`
	Count int               `json:"count" msgpack:"count"`
}

func TestJSONCodec(t *testing.T) {
	codec := NewJSONCodec()

	testData := TestData{
		ID:    "test-1",
		Name:  "Test Data",
		Data:  map[string]string{"key": "value"},
		Count: 42,
	}

	// Test encoding
	encoded, err := codec.Encode(testData)
	require.NoError(t, err)
	assert.NotEmpty(t, encoded)

	// Test decoding
	var decoded TestData
	err = codec.Decode(encoded, &decoded)
	require.NoError(t, err)
	assert.Equal(t, testData, decoded)

	// Test codec name
	assert.Equal(t, "json", codec.Name())
}

func TestMsgPackCodec(t *testing.T) {
	codec := NewMsgPackCodec()

	testData := TestData{
		ID:    "test-1",
		Name:  "Test Data",
		Data:  map[string]string{"key": "value"},
		Count: 42,
	}

	// Test encoding
	encoded, err := codec.Encode(testData)
	require.NoError(t, err)
	assert.NotEmpty(t, encoded)

	// Test decoding
	var decoded TestData
	err = codec.Decode(encoded, &decoded)
	require.NoError(t, err)
	assert.Equal(t, testData, decoded)

	// Test codec name
	assert.Equal(t, "msgpack", codec.Name())
}

func TestSerializer_BasicSerialization(t *testing.T) {
	config := SerializationConfig{
		Codec:       NewJSONCodec(),
		Compression: CompressionNone,
	}
	serializer := NewSerializer(config)

	testData := TestData{
		ID:    "test-1",
		Name:  "Test Data",
		Data:  map[string]string{"key": "value"},
		Count: 42,
	}

	// Test serialization
	serialized, err := serializer.Serialize(testData)
	require.NoError(t, err)
	assert.NotEmpty(t, serialized)

	// Test deserialization
	var deserialized TestData
	err = serializer.Deserialize(serialized, &deserialized)
	require.NoError(t, err)
	assert.Equal(t, testData, deserialized)
}

func TestSerializer_WithCompression(t *testing.T) {
	tests := []struct {
		name        string
		compression CompressionType
	}{
		{"gzip compression", CompressionGzip},
		{"zstd compression", CompressionZstd},
		{"no compression", CompressionNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := SerializationConfig{
				Codec:       NewMsgPackCodec(),
				Compression: tt.compression,
			}
			serializer := NewSerializer(config)

			testData := TestData{
				ID:   "test-1",
				Name: "Large Test Data with lots of repetitive content to test compression efficiency",
				Data: map[string]string{
					"key1": "value1 repeated content repeated content repeated content",
					"key2": "value2 repeated content repeated content repeated content",
					"key3": "value3 repeated content repeated content repeated content",
				},
				Count: 1000,
			}

			// Test serialization with compression
			serialized, err := serializer.Serialize(testData)
			require.NoError(t, err)
			assert.NotEmpty(t, serialized)

			// Test deserialization
			var deserialized TestData
			err = serializer.Deserialize(serialized, &deserialized)
			require.NoError(t, err)
			assert.Equal(t, testData, deserialized)
		})
	}
}

func TestSerializer_WithEncryption(t *testing.T) {
	// Generate AES-256 key
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)

	config := SerializationConfig{
		Codec:       NewJSONCodec(),
		Compression: CompressionNone,
		EncryptKey:  key,
	}
	serializer := NewSerializer(config)

	testData := TestData{
		ID:    "secret-data",
		Name:  "Confidential Information",
		Data:  map[string]string{"secret": "very confidential"},
		Count: 42,
	}

	// Test serialization with encryption
	serialized, err := serializer.Serialize(testData)
	require.NoError(t, err)
	assert.NotEmpty(t, serialized)

	// Verify data is encrypted (shouldn't contain plaintext)
	assert.NotContains(t, string(serialized), "secret-data")
	assert.NotContains(t, string(serialized), "Confidential Information")

	// Test deserialization
	var deserialized TestData
	err = serializer.Deserialize(serialized, &deserialized)
	require.NoError(t, err)
	assert.Equal(t, testData, deserialized)
}

func TestSerializer_CompressionAndEncryption(t *testing.T) {
	// Generate AES-256 key
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)

	config := SerializationConfig{
		Codec:       NewMsgPackCodec(),
		Compression: CompressionZstd,
		EncryptKey:  key,
	}
	serializer := NewSerializer(config)

	testData := TestData{
		ID:   "compressed-encrypted",
		Name: "Data with both compression and encryption enabled for maximum security",
		Data: map[string]string{
			"key1": "large repetitive content " + string(make([]byte, 1000)),
			"key2": "more repetitive content " + string(make([]byte, 1000)),
		},
		Count: 9999,
	}

	// Test full pipeline
	serialized, err := serializer.Serialize(testData)
	require.NoError(t, err)
	assert.NotEmpty(t, serialized)

	// Test deserialization
	var deserialized TestData
	err = serializer.Deserialize(serialized, &deserialized)
	require.NoError(t, err)
	assert.Equal(t, testData, deserialized)
}

func TestDefaultSerializer(t *testing.T) {
	serializer := DefaultSerializer()

	testData := TestData{
		ID:    "default-test",
		Name:  "Default Serializer Test",
		Data:  map[string]string{"default": "config"},
		Count: 123,
	}

	// Test with default configuration
	serialized, err := serializer.Serialize(testData)
	require.NoError(t, err)
	assert.NotEmpty(t, serialized)

	var deserialized TestData
	err = serializer.Deserialize(serialized, &deserialized)
	require.NoError(t, err)
	assert.Equal(t, testData, deserialized)
}

func TestSerializer_ErrorHandling(t *testing.T) {
	t.Run("invalid encryption key size", func(t *testing.T) {
		config := SerializationConfig{
			Codec:      NewJSONCodec(),
			EncryptKey: []byte("short"), // Invalid key size
		}
		invalidSerializer := NewSerializer(config)

		_, err := invalidSerializer.Serialize("test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "encryption failed")
	})

	t.Run("corrupted encrypted data", func(t *testing.T) {
		key := make([]byte, 32)
		rand.Read(key)

		config := SerializationConfig{
			Codec:      NewJSONCodec(),
			EncryptKey: key,
		}
		encryptedSerializer := NewSerializer(config)

		// Create corrupted data
		corruptedData := []byte("corrupted encrypted data")

		var result interface{}
		err := encryptedSerializer.Deserialize(corruptedData, &result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "decryption failed")
	})
}

func BenchmarkSerializer_JSON(b *testing.B) {
	serializer := NewSerializer(SerializationConfig{
		Codec:       NewJSONCodec(),
		Compression: CompressionNone,
	})

	testData := TestData{
		ID:    "benchmark-test",
		Name:  "Benchmark Data",
		Data:  map[string]string{"key": "value"},
		Count: 1000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		serialized, _ := serializer.Serialize(testData)
		var deserialized TestData
		_ = serializer.Deserialize(serialized, &deserialized)
	}
}

func BenchmarkSerializer_MsgPack(b *testing.B) {
	serializer := NewSerializer(SerializationConfig{
		Codec:       NewMsgPackCodec(),
		Compression: CompressionNone,
	})

	testData := TestData{
		ID:    "benchmark-test",
		Name:  "Benchmark Data",
		Data:  map[string]string{"key": "value"},
		Count: 1000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		serialized, _ := serializer.Serialize(testData)
		var deserialized TestData
		_ = serializer.Deserialize(serialized, &deserialized)
	}
}

func BenchmarkSerializer_WithCompression(b *testing.B) {
	serializer := NewSerializer(SerializationConfig{
		Codec:       NewMsgPackCodec(),
		Compression: CompressionZstd,
	})

	// Create larger test data for compression
	largeData := make(map[string]string)
	for i := 0; i < 100; i++ {
		largeData[fmt.Sprintf("key%d", i)] = "repetitive content " + string(make([]byte, 100))
	}

	testData := TestData{
		ID:    "benchmark-compression",
		Name:  "Large Benchmark Data for Compression",
		Data:  largeData,
		Count: 10000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		serialized, _ := serializer.Serialize(testData)
		var deserialized TestData
		_ = serializer.Deserialize(serialized, &deserialized)
	}
}
