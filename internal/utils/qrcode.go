// Package utils provides utility functions for the VPN server including
// QR code generation, configuration formatting, and other helper functions
// commonly used across different components of the system.
package utils

import (
	"bytes"
	"encoding/base64"
	"fmt"

	"github.com/skip2/go-qrcode"
)

// QRCodeGenerator provides functionality to generate QR codes for VPN client configurations.
// It supports generating QR codes in different formats and sizes for easy sharing
// of WireGuard client configurations via mobile devices and QR code scanners.
type QRCodeGenerator struct {
	// Size determines the pixel dimensions of the generated QR code
	Size int
	// RecoveryLevel determines the error correction level for the QR code
	RecoveryLevel qrcode.RecoveryLevel
}

// QRCodeOptions represents configuration options for QR code generation.
type QRCodeOptions struct {
	Size          int                    `json:"size"`           // QR code size in pixels (default: 256)
	RecoveryLevel qrcode.RecoveryLevel   `json:"recovery_level"` // Error correction level (default: Medium)
	Format        string                 `json:"format"`         // Output format: "png", "base64", "terminal"
}

// NewQRCodeGenerator creates a new QR code generator with default settings.
// The default settings provide a good balance between size and readability
// for most use cases including mobile device scanning.
// Returns a pointer to the newly created QRCodeGenerator.
func NewQRCodeGenerator() *QRCodeGenerator {
	return &QRCodeGenerator{
		Size:          256,
		RecoveryLevel: qrcode.Medium,
	}
}

// NewQRCodeGeneratorWithOptions creates a new QR code generator with custom options.
// This allows fine-tuning the QR code generation for specific requirements
// such as different sizes or error correction levels.
// Returns a pointer to the newly created QRCodeGenerator.
func NewQRCodeGeneratorWithOptions(options QRCodeOptions) *QRCodeGenerator {
	generator := &QRCodeGenerator{
		Size:          options.Size,
		RecoveryLevel: options.RecoveryLevel,
	}
	
	// Set defaults if not specified
	if generator.Size <= 0 {
		generator.Size = 256
	}
	
	return generator
}

// GeneratePNG generates a QR code as PNG image data.
// It takes the content string (typically a WireGuard configuration) and returns
// the PNG image data as a byte slice that can be saved to file or served over HTTP.
// Returns the PNG data or an error if generation fails.
func (qr *QRCodeGenerator) GeneratePNG(content string) ([]byte, error) {
	pngData, err := qrcode.Encode(content, qr.RecoveryLevel, qr.Size)
	if err != nil {
		return nil, fmt.Errorf("failed to generate QR code PNG: %w", err)
	}
	return pngData, nil
}

// GenerateBase64 generates a QR code as base64-encoded PNG image.
// This is useful for embedding QR codes directly in HTML pages or JSON responses
// without requiring separate image files or endpoints.
// Returns the base64-encoded PNG data or an error if generation fails.
func (qr *QRCodeGenerator) GenerateBase64(content string) (string, error) {
	pngData, err := qr.GeneratePNG(content)
	if err != nil {
		return "", fmt.Errorf("failed to generate PNG for base64 encoding: %w", err)
	}
	
	encoded := base64.StdEncoding.EncodeToString(pngData)
	return fmt.Sprintf("data:image/png;base64,%s", encoded), nil
}

// GenerateTerminal generates a QR code for display in terminal/console.
// This creates an ASCII representation of the QR code that can be displayed
// in command-line interfaces, making it useful for server administration.
// Returns the terminal-formatted QR code string or an error if generation fails.
func (qr *QRCodeGenerator) GenerateTerminal(content string) (string, error) {
	qrCode, err := qrcode.New(content, qr.RecoveryLevel)
	if err != nil {
		return "", fmt.Errorf("failed to create QR code: %w", err)
	}
	
	// Generate a simple ASCII representation using the bitmap
	bitmap := qrCode.Bitmap()
	return qr.convertBitmapToASCII(bitmap), nil
}

// convertBitmapToASCII converts a QR code bitmap to ASCII representation.
// This creates a simple text-based visualization using block characters.
func (qr *QRCodeGenerator) convertBitmapToASCII(bitmap [][]bool) string {
	var buf bytes.Buffer
	
	// Add top border
	buf.WriteString("  ")
	for range bitmap[0] {
		buf.WriteString("██")
	}
	buf.WriteString("\n")
	
	// Convert bitmap to ASCII using block characters
	for _, row := range bitmap {
		buf.WriteString("██") // Left border
		for _, module := range row {
			if module {
				buf.WriteString("  ") // Black module (inverted for better visibility)
			} else {
				buf.WriteString("██") // White module
			}
		}
		buf.WriteString("██\n") // Right border
	}
	
	// Add bottom border
	buf.WriteString("  ")
	for range bitmap[0] {
		buf.WriteString("██")
	}
	buf.WriteString("\n")
	
	return buf.String()
}

// Generate creates a QR code in the specified format.
// This is a convenience method that automatically chooses the appropriate
// generation method based on the format parameter.
// Supported formats: "png", "base64", "terminal"
// Returns the generated QR code data and format-specific type information.
func (qr *QRCodeGenerator) Generate(content string, format string) (interface{}, error) {
	switch format {
	case "png":
		return qr.GeneratePNG(content)
	case "base64":
		return qr.GenerateBase64(content)
	case "terminal":
		return qr.GenerateTerminal(content)
	default:
		return nil, fmt.Errorf("unsupported format: %s (supported: png, base64, terminal)", format)
	}
}

// GenerateWireGuardConfigQR generates a QR code for a WireGuard client configuration.
// This is a specialized method that formats the configuration appropriately
// for QR code scanning by WireGuard mobile applications.
// Returns the QR code in the specified format or an error if generation fails.
func GenerateWireGuardConfigQR(config string, options QRCodeOptions) (interface{}, error) {
	if config == "" {
		return nil, fmt.Errorf("configuration cannot be empty")
	}
	
	// Validate that this looks like a WireGuard config
	if !validateWireGuardConfig(config) {
		return nil, fmt.Errorf("invalid WireGuard configuration format")
	}
	
	generator := NewQRCodeGeneratorWithOptions(options)
	return generator.Generate(config, options.Format)
}

// validateWireGuardConfig performs basic validation on WireGuard configuration content.
// It checks for the presence of required sections and basic formatting.
// Returns true if the configuration appears to be valid WireGuard format.
func validateWireGuardConfig(config string) bool {
	// Basic validation - check for required sections
	return containsSection(config, "[Interface]") && containsSection(config, "[Peer]")
}

// containsSection checks if the configuration contains a specific section header.
// This is a helper function for configuration validation.
func containsSection(config, section string) bool {
	return len(config) > 0 && contains(config, section)
}

// contains is a simple string contains check.
// This helper function provides compatibility across different Go versions.
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// GetDefaultQRCodeOptions returns the default options for QR code generation.
// These defaults are optimized for WireGuard mobile app compatibility
// and provide good readability across different devices and lighting conditions.
func GetDefaultQRCodeOptions() QRCodeOptions {
	return QRCodeOptions{
		Size:          256,
		RecoveryLevel: qrcode.Medium,
		Format:        "base64",
	}
}

// GetTerminalQRCodeOptions returns options optimized for terminal display.
// These settings ensure the QR code displays well in command-line environments.
func GetTerminalQRCodeOptions() QRCodeOptions {
	return QRCodeOptions{
		Size:          256, // Size doesn't affect terminal output
		RecoveryLevel: qrcode.Medium,
		Format:        "terminal",
	}
}