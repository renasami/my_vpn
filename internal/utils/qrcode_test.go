// Package utils provides utility functions for the VPN server including
// QR code generation, configuration formatting, and other helper functions
// commonly used across different components of the system.
package utils

import (
	"strings"
	"testing"

	"github.com/skip2/go-qrcode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleWireGuardConfig = `[Interface]
PrivateKey = cG9zdCBleGFtcGxlIGNvZGU=
Address = 10.0.0.2/32
DNS = 8.8.8.8

[Peer]
PublicKey = c2hhcmluZyBpcyBjYXJpbmc=
Endpoint = 203.0.113.1:51820
AllowedIPs = 0.0.0.0/0`

func TestNewQRCodeGenerator(t *testing.T) {
	t.Run("should create QR code generator with default settings", func(t *testing.T) {
		generator := NewQRCodeGenerator()
		
		assert.NotNil(t, generator)
		assert.Equal(t, 256, generator.Size)
		assert.Equal(t, qrcode.Medium, generator.RecoveryLevel)
	})
}

func TestNewQRCodeGeneratorWithOptions(t *testing.T) {
	t.Run("should create QR code generator with custom options", func(t *testing.T) {
		options := QRCodeOptions{
			Size:          512,
			RecoveryLevel: qrcode.High,
		}
		
		generator := NewQRCodeGeneratorWithOptions(options)
		
		assert.NotNil(t, generator)
		assert.Equal(t, 512, generator.Size)
		assert.Equal(t, qrcode.High, generator.RecoveryLevel)
	})
	
	t.Run("should use defaults for invalid options", func(t *testing.T) {
		options := QRCodeOptions{
			Size:          0, // Invalid size
			RecoveryLevel: qrcode.Low,
		}
		
		generator := NewQRCodeGeneratorWithOptions(options)
		
		assert.NotNil(t, generator)
		assert.Equal(t, 256, generator.Size) // Should use default
		assert.Equal(t, qrcode.Low, generator.RecoveryLevel)
	})
}

func TestQRCodeGenerator_GeneratePNG(t *testing.T) {
	generator := NewQRCodeGenerator()
	
	t.Run("should generate PNG QR code successfully", func(t *testing.T) {
		testContent := "Test QR Code Content"
		
		pngData, err := generator.GeneratePNG(testContent)
		
		require.NoError(t, err)
		assert.NotEmpty(t, pngData)
		assert.Greater(t, len(pngData), 100) // PNG should have reasonable size
		
		// Check PNG header
		assert.Equal(t, []byte{0x89, 0x50, 0x4E, 0x47}, pngData[:4])
	})
	
	t.Run("should generate different PNG for different content", func(t *testing.T) {
		content1 := "Content 1"
		content2 := "Content 2"
		
		png1, err := generator.GeneratePNG(content1)
		require.NoError(t, err)
		
		png2, err := generator.GeneratePNG(content2)
		require.NoError(t, err)
		
		assert.NotEqual(t, png1, png2)
	})
	
	t.Run("should handle empty content", func(t *testing.T) {
		pngData, err := generator.GeneratePNG("")
		
		// Empty content should result in an error
		assert.Error(t, err)
		assert.Empty(t, pngData)
		assert.Contains(t, err.Error(), "no data to encode")
	})
}

func TestQRCodeGenerator_GenerateBase64(t *testing.T) {
	generator := NewQRCodeGenerator()
	
	t.Run("should generate base64 QR code successfully", func(t *testing.T) {
		testContent := "Test QR Code Content"
		
		base64Data, err := generator.GenerateBase64(testContent)
		
		require.NoError(t, err)
		assert.NotEmpty(t, base64Data)
		assert.True(t, strings.HasPrefix(base64Data, "data:image/png;base64,"))
		assert.Greater(t, len(base64Data), 100) // Base64 should have reasonable length
	})
	
	t.Run("should generate different base64 for different content", func(t *testing.T) {
		content1 := "Content 1"
		content2 := "Content 2"
		
		base64_1, err := generator.GenerateBase64(content1)
		require.NoError(t, err)
		
		base64_2, err := generator.GenerateBase64(content2)
		require.NoError(t, err)
		
		assert.NotEqual(t, base64_1, base64_2)
	})
}

func TestQRCodeGenerator_GenerateTerminal(t *testing.T) {
	generator := NewQRCodeGenerator()
	
	t.Run("should generate terminal QR code successfully", func(t *testing.T) {
		testContent := "Test QR Code Content"
		
		terminalQR, err := generator.GenerateTerminal(testContent)
		
		require.NoError(t, err)
		assert.NotEmpty(t, terminalQR)
		
		// Terminal QR should contain block characters or spaces
		assert.True(t, len(terminalQR) > 50) // Should be reasonably sized
	})
	
	t.Run("should generate different terminal QR for different content", func(t *testing.T) {
		content1 := "Content 1"
		content2 := "Content 2"
		
		terminal1, err := generator.GenerateTerminal(content1)
		require.NoError(t, err)
		
		terminal2, err := generator.GenerateTerminal(content2)
		require.NoError(t, err)
		
		assert.NotEqual(t, terminal1, terminal2)
	})
}

func TestQRCodeGenerator_Generate(t *testing.T) {
	generator := NewQRCodeGenerator()
	testContent := "Test QR Code Content"
	
	t.Run("should generate PNG format", func(t *testing.T) {
		result, err := generator.Generate(testContent, "png")
		
		require.NoError(t, err)
		assert.IsType(t, []byte{}, result)
		
		pngData := result.([]byte)
		assert.Greater(t, len(pngData), 100)
	})
	
	t.Run("should generate base64 format", func(t *testing.T) {
		result, err := generator.Generate(testContent, "base64")
		
		require.NoError(t, err)
		assert.IsType(t, "", result)
		
		base64Data := result.(string)
		assert.True(t, strings.HasPrefix(base64Data, "data:image/png;base64,"))
	})
	
	t.Run("should generate terminal format", func(t *testing.T) {
		result, err := generator.Generate(testContent, "terminal")
		
		require.NoError(t, err)
		assert.IsType(t, "", result)
		
		terminalData := result.(string)
		assert.Greater(t, len(terminalData), 50)
	})
	
	t.Run("should reject unsupported format", func(t *testing.T) {
		result, err := generator.Generate(testContent, "unsupported")
		
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "unsupported format")
	})
}

func TestGenerateWireGuardConfigQR(t *testing.T) {
	t.Run("should generate QR code for valid WireGuard config", func(t *testing.T) {
		options := QRCodeOptions{
			Size:          256,
			RecoveryLevel: qrcode.Medium,
			Format:        "base64",
		}
		
		result, err := GenerateWireGuardConfigQR(sampleWireGuardConfig, options)
		
		require.NoError(t, err)
		assert.IsType(t, "", result)
		
		base64Data := result.(string)
		assert.True(t, strings.HasPrefix(base64Data, "data:image/png;base64,"))
	})
	
	t.Run("should reject empty configuration", func(t *testing.T) {
		options := GetDefaultQRCodeOptions()
		
		result, err := GenerateWireGuardConfigQR("", options)
		
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "configuration cannot be empty")
	})
	
	t.Run("should reject invalid WireGuard configuration", func(t *testing.T) {
		invalidConfig := "This is not a WireGuard config"
		options := GetDefaultQRCodeOptions()
		
		result, err := GenerateWireGuardConfigQR(invalidConfig, options)
		
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid WireGuard configuration format")
	})
	
	t.Run("should generate PNG format for WireGuard config", func(t *testing.T) {
		options := QRCodeOptions{
			Size:          256,
			RecoveryLevel: qrcode.Medium,
			Format:        "png",
		}
		
		result, err := GenerateWireGuardConfigQR(sampleWireGuardConfig, options)
		
		require.NoError(t, err)
		assert.IsType(t, []byte{}, result)
		
		pngData := result.([]byte)
		assert.Greater(t, len(pngData), 100)
	})
	
	t.Run("should generate terminal format for WireGuard config", func(t *testing.T) {
		options := GetTerminalQRCodeOptions()
		
		result, err := GenerateWireGuardConfigQR(sampleWireGuardConfig, options)
		
		require.NoError(t, err)
		assert.IsType(t, "", result)
		
		terminalData := result.(string)
		assert.Greater(t, len(terminalData), 50)
	})
}

func TestValidateWireGuardConfig(t *testing.T) {
	t.Run("should validate correct WireGuard config", func(t *testing.T) {
		isValid := validateWireGuardConfig(sampleWireGuardConfig)
		assert.True(t, isValid)
	})
	
	t.Run("should reject config without Interface section", func(t *testing.T) {
		configWithoutInterface := `[Peer]
PublicKey = c2hhcmluZyBpcyBjYXJpbmc=
Endpoint = 203.0.113.1:51820
AllowedIPs = 0.0.0.0/0`
		
		isValid := validateWireGuardConfig(configWithoutInterface)
		assert.False(t, isValid)
	})
	
	t.Run("should reject config without Peer section", func(t *testing.T) {
		configWithoutPeer := `[Interface]
PrivateKey = cG9zdCBleGFtcGxlIGNvZGU=
Address = 10.0.0.2/32
DNS = 8.8.8.8`
		
		isValid := validateWireGuardConfig(configWithoutPeer)
		assert.False(t, isValid)
	})
	
	t.Run("should reject empty config", func(t *testing.T) {
		isValid := validateWireGuardConfig("")
		assert.False(t, isValid)
	})
}

func TestContainsSection(t *testing.T) {
	config := "[Interface]\nSome content\n[Peer]\nMore content"
	
	t.Run("should find existing section", func(t *testing.T) {
		assert.True(t, containsSection(config, "[Interface]"))
		assert.True(t, containsSection(config, "[Peer]"))
	})
	
	t.Run("should not find non-existing section", func(t *testing.T) {
		assert.False(t, containsSection(config, "[NonExistent]"))
	})
	
	t.Run("should handle empty config", func(t *testing.T) {
		assert.False(t, containsSection("", "[Interface]"))
	})
}

func TestGetDefaultQRCodeOptions(t *testing.T) {
	t.Run("should return default options", func(t *testing.T) {
		options := GetDefaultQRCodeOptions()
		
		assert.Equal(t, 256, options.Size)
		assert.Equal(t, qrcode.Medium, options.RecoveryLevel)
		assert.Equal(t, "base64", options.Format)
	})
}

func TestGetTerminalQRCodeOptions(t *testing.T) {
	t.Run("should return terminal options", func(t *testing.T) {
		options := GetTerminalQRCodeOptions()
		
		assert.Equal(t, 256, options.Size)
		assert.Equal(t, qrcode.Medium, options.RecoveryLevel)
		assert.Equal(t, "terminal", options.Format)
	})
}