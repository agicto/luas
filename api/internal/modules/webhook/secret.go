package webhook

import (
	"encoding/base64"
	"fmt"
	"slices"
	"strings"

	"github.com/zgiai/luas/api/internal/capabilities/crypto"
	"github.com/zgiai/luas/api/internal/domain"
	"github.com/zgiai/luas/api/internal/infra/config"
)

const webhookSecretPrefix = "whsec_"

type secretMaterial struct {
	Plaintext  string
	Ciphertext string
	Hint       string
}

type secretProtector struct {
	encryptor *crypto.AESEncryptor
}

// NewSecretProtector creates the endpoint signing-key custody adapter.
func NewSecretProtector(cfg *config.Config) (*secretProtector, error) {
	if cfg == nil {
		return &secretProtector{}, nil
	}
	selected := slices.Contains(cfg.Starters.Optional, "webhook")
	if !selected && strings.TrimSpace(cfg.Webhook.EncryptionKey) == "" {
		return &secretProtector{}, nil
	}
	if len(strings.TrimSpace(cfg.Webhook.EncryptionKey)) < 32 {
		return nil, fmt.Errorf("webhook encryption key is unavailable: %w", domain.ErrServiceUnavailable)
	}
	return &secretProtector{encryptor: crypto.NewAESEncryptorFromString(cfg.Webhook.EncryptionKey)}, nil
}

func (p *secretProtector) Generate() (*secretMaterial, error) {
	if p == nil || p.encryptor == nil {
		return nil, domain.ErrServiceUnavailable
	}
	key, err := crypto.GenerateKey(32)
	if err != nil {
		return nil, fmt.Errorf("generate webhook secret: %w", err)
	}
	defer clear(key)
	plaintext := webhookSecretPrefix + base64.StdEncoding.EncodeToString(key)
	ciphertext, err := p.encryptor.EncryptString(plaintext)
	if err != nil {
		return nil, fmt.Errorf("encrypt webhook secret: %w", err)
	}
	return &secretMaterial{
		Plaintext:  plaintext,
		Ciphertext: ciphertext,
		Hint:       plaintext[len(plaintext)-8:],
	}, nil
}

func (p *secretProtector) Decrypt(ciphertext string) ([]byte, error) {
	if p == nil || p.encryptor == nil || ciphertext == "" {
		return nil, domain.ErrServiceUnavailable
	}
	plaintext, err := p.encryptor.DecryptString(ciphertext)
	if err != nil || !strings.HasPrefix(plaintext, webhookSecretPrefix) {
		return nil, fmt.Errorf("decrypt webhook secret: %w", domain.ErrServiceUnavailable)
	}
	key, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(plaintext, webhookSecretPrefix))
	if err != nil || len(key) != 32 {
		return nil, fmt.Errorf("decode webhook secret: %w", domain.ErrServiceUnavailable)
	}
	return key, nil
}
