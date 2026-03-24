package media

import (
	"bytes"
	"testing"
)

func TestEncryptDecryptAESECB(t *testing.T) {
	tests := []struct {
		name      string
		plaintext []byte
	}{
		{"empty", []byte{}},
		{"small", []byte("hello")},
		{"exact_block", make([]byte, 16)},
		{"multiple_blocks", make([]byte, 48)},
		{"random_100", randomBytes(100)},
		{"random_1000", randomBytes(1000)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := GenerateAESKey()
			if err != nil {
				t.Fatalf("GenerateAESKey: %v", err)
			}

			ciphertext, err := EncryptAESECB(tt.plaintext, key)
			if err != nil {
				t.Fatalf("EncryptAESECB: %v", err)
			}

			decrypted, err := DecryptAESECB(ciphertext, key)
			if err != nil {
				t.Fatalf("DecryptAESECB: %v", err)
			}

			if !bytes.Equal(tt.plaintext, decrypted) {
				t.Errorf("decrypted != plaintext\ngot: %v\nwant: %v", decrypted, tt.plaintext)
			}
		})
	}
}

func TestAESECBPaddedSize(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{0, 16},
		{1, 16},
		{15, 16},
		{16, 32},
		{17, 32},
		{100, 112},
		{128, 144},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := AESECBPaddedSize(tt.input)
			if got != tt.expected {
				t.Errorf("AESECBPaddedSize(%d) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

func TestEncryptAESECBInvalidKey(t *testing.T) {
	tests := []struct {
		name string
		key  []byte
	}{
		{"too_short", []byte("short")},
		{"too_long", make([]byte, 32)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := EncryptAESECB([]byte("test"), tt.key)
			if err == nil {
				t.Error("expected error for invalid key length")
			}
		})
	}
}

func TestDecryptAESECBInvalidCiphertext(t *testing.T) {
	key, _ := GenerateAESKey()

	tests := []struct {
		name       string
		ciphertext []byte
	}{
		{"not_multiple_of_16", []byte("not16bytes!")},
		{"empty", []byte{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecryptAESECB(tt.ciphertext, key)
			if err == nil {
				t.Error("expected error for invalid ciphertext")
			}
		})
	}
}

func TestGenerateAESKey(t *testing.T) {
	key1, err := GenerateAESKey()
	if err != nil {
		t.Fatalf("GenerateAESKey: %v", err)
	}

	if len(key1) != 16 {
		t.Errorf("key length = %d, want 16", len(key1))
	}

	key2, err := GenerateAESKey()
	if err != nil {
		t.Fatalf("GenerateAESKey: %v", err)
	}

	if bytes.Equal(key1, key2) {
		t.Error("keys should be random and different")
	}
}

func TestGenerateFileKey(t *testing.T) {
	key1, err := GenerateFileKey()
	if err != nil {
		t.Fatalf("GenerateFileKey: %v", err)
	}

	// Should be 32 hex characters (16 bytes)
	if len(key1) != 32 {
		t.Errorf("file key length = %d, want 32", len(key1))
	}

	key2, err := GenerateFileKey()
	if err != nil {
		t.Fatalf("GenerateFileKey: %v", err)
	}

	if key1 == key2 {
		t.Error("file keys should be random and different")
	}
}

func randomBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i % 256)
	}
	return b
}