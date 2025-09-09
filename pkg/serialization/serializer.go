// Package serialization provides flexible serialization for FlowGraph checkpoints
// PRINCIPLES:
// - KISS: Simple interface with multiple codec implementations
// - DRY: Reusable across all checkpoint implementations
// - SOLID: Interface segregation for different serializers
package serialization

import (
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"

	"github.com/klauspost/compress/zstd"
	"github.com/vmihailenco/msgpack/v5"
)

// Codec interface for serialization
// PRINCIPLES:
// - ISP: Simple interface with ≤5 methods
// - SRP: Single responsibility for serialization
type Codec interface {
	Encode(v interface{}) ([]byte, error)
	Decode(data []byte, v interface{}) error
	Name() string
}

// CompressionType represents compression algorithms
type CompressionType string

const (
	CompressionNone CompressionType = "none"
	CompressionGzip CompressionType = "gzip"
	CompressionZstd CompressionType = "zstd"
)

// SerializationConfig holds serialization settings
type SerializationConfig struct {
	Codec       Codec
	Compression CompressionType
	EncryptKey  []byte // AES-256 key (32 bytes)
}

// Serializer provides complete serialization with compression and encryption
// PRINCIPLES:
// - KISS: Simple interface hiding complex operations
// - SRP: Single responsibility for complete serialization pipeline
type Serializer struct {
	config SerializationConfig
}

// NewSerializer creates a new serializer with configuration
func NewSerializer(config SerializationConfig) *Serializer {
	return &Serializer{config: config}
}

// Serialize encodes, compresses, and encrypts data
// PRINCIPLES:
// - KISS: Simple method signature, complex work hidden
// - Error handling follows Go idioms
func (s *Serializer) Serialize(v interface{}) ([]byte, error) {
	// 1. Encode with codec
	data, err := s.config.Codec.Encode(v)
	if err != nil {
		return nil, fmt.Errorf("codec encoding failed: %w", err)
	}

	// 2. Compress if enabled
	data, err = s.compress(data)
	if err != nil {
		return nil, fmt.Errorf("compression failed: %w", err)
	}

	// 3. Encrypt if key provided
	if len(s.config.EncryptKey) > 0 {
		data, err = s.encrypt(data)
		if err != nil {
			return nil, fmt.Errorf("encryption failed: %w", err)
		}
	}

	return data, nil
}

// Deserialize decrypts, decompresses, and decodes data
func (s *Serializer) Deserialize(data []byte, v interface{}) error {
	var err error

	// 1. Decrypt if key provided
	if len(s.config.EncryptKey) > 0 {
		data, err = s.decrypt(data)
		if err != nil {
			return fmt.Errorf("decryption failed: %w", err)
		}
	}

	// 2. Decompress if enabled
	data, err = s.decompress(data)
	if err != nil {
		return fmt.Errorf("decompression failed: %w", err)
	}

	// 3. Decode with codec
	err = s.config.Codec.Decode(data, v)
	if err != nil {
		return fmt.Errorf("codec decoding failed: %w", err)
	}

	return nil
}

// compress applies compression based on configuration
func (s *Serializer) compress(data []byte) ([]byte, error) {
	switch s.config.Compression {
	case CompressionNone:
		return data, nil
	case CompressionGzip:
		return s.compressGzip(data)
	case CompressionZstd:
		return s.compressZstd(data)
	default:
		return data, nil
	}
}

// decompress removes compression based on configuration
func (s *Serializer) decompress(data []byte) ([]byte, error) {
	switch s.config.Compression {
	case CompressionNone:
		return data, nil
	case CompressionGzip:
		return s.decompressGzip(data)
	case CompressionZstd:
		return s.decompressZstd(data)
	default:
		return data, nil
	}
}

// compressGzip compresses data using gzip
func (s *Serializer) compressGzip(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)

	if _, err := writer.Write(data); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// decompressGzip decompresses gzip data
func (s *Serializer) decompressGzip(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

// compressZstd compresses data using zstd
func (s *Serializer) compressZstd(data []byte) ([]byte, error) {
	encoder, err := zstd.NewWriter(nil)
	if err != nil {
		return nil, err
	}
	defer encoder.Close()

	return encoder.EncodeAll(data, nil), nil
}

// decompressZstd decompresses zstd data
func (s *Serializer) decompressZstd(data []byte) ([]byte, error) {
	decoder, err := zstd.NewReader(nil)
	if err != nil {
		return nil, err
	}
	defer decoder.Close()

	return decoder.DecodeAll(data, nil)
}

// encrypt encrypts data using AES-GCM
func (s *Serializer) encrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.config.EncryptKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// decrypt decrypts data using AES-GCM
func (s *Serializer) decrypt(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.config.EncryptKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("invalid ciphertext size")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// JSONCodec implements JSON serialization
type JSONCodec struct{}

func (c *JSONCodec) Encode(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (c *JSONCodec) Decode(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func (c *JSONCodec) Name() string {
	return "json"
}

// MsgPackCodec implements MessagePack serialization
type MsgPackCodec struct{}

func (c *MsgPackCodec) Encode(v interface{}) ([]byte, error) {
	return msgpack.Marshal(v)
}

func (c *MsgPackCodec) Decode(data []byte, v interface{}) error {
	return msgpack.Unmarshal(data, v)
}

func (c *MsgPackCodec) Name() string {
	return "msgpack"
}

// NewJSONCodec creates a new JSON codec
func NewJSONCodec() Codec {
	return &JSONCodec{}
}

// NewMsgPackCodec creates a new MessagePack codec
func NewMsgPackCodec() Codec {
	return &MsgPackCodec{}
}

// DefaultSerializer creates a serializer with sensible defaults
func DefaultSerializer() *Serializer {
	return NewSerializer(SerializationConfig{
		Codec:       NewMsgPackCodec(),
		Compression: CompressionZstd,
	})
}
