package mitm

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go-9router/internal/db"
)

type CertManager struct {
	mu        sync.RWMutex
	mitmDir   string
	caCert    *x509.Certificate
	caPrivKey *rsa.PrivateKey
	caPEM     []byte
}

func NewCertManager(mitmDir string) (*CertManager, error) {
	if mitmDir == "" {
		mitmDir = filepath.Join(db.GetDataDir(), "mitm")
	}

	if err := os.MkdirAll(mitmDir, 0755); err != nil {
		return nil, err
	}

	cm := &CertManager{
		mitmDir: mitmDir,
	}

	if err := cm.loadOrCreateRootCA(); err != nil {
		return nil, err
	}

	return cm, nil
}

func (cm *CertManager) GetCAPath() string {
	return filepath.Join(cm.mitmDir, "rootCA.crt")
}

func (cm *CertManager) loadOrCreateRootCA() error {
	caCertPath := filepath.Join(cm.mitmDir, "rootCA.crt")
	caKeyPath := filepath.Join(cm.mitmDir, "rootCA.key")

	// 1. Check if files exist and are valid (not expiring within 30 days)
	if _, errCert := os.Stat(caCertPath); errCert == nil {
		if _, errKey := os.Stat(caKeyPath); errKey == nil {
			certPEM, err := os.ReadFile(caCertPath)
			if err == nil {
				keyPEM, err := os.ReadFile(caKeyPath)
				if err == nil {
					block, _ := pem.Decode(certPEM)
					if block != nil {
						cert, err := x509.ParseCertificate(block.Bytes)
						if err == nil && time.Now().Add(30*24*time.Hour).Before(cert.NotAfter) {
							// CA is valid! Load it.
							keyBlock, _ := pem.Decode(keyPEM)
							if keyBlock != nil {
								privKey, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
								if err == nil {
									cm.caCert = cert
									cm.caPrivKey = privKey
									cm.caPEM = certPEM
									return nil
								}
							}
						}
					}
				}
			}
		}
	}

	// 2. Generate new Root CA
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate CA private key: %w", err)
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("failed to generate serial number: %w", err)
	}

	caTemplate := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   "9Router MITM Root CA",
			Organization: []string{"9Router"},
			Country:      []string{"US"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0), // 10 years validity
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caBytes, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &privKey.PublicKey, privKey)
	if err != nil {
		return fmt.Errorf("failed to create CA certificate: %w", err)
	}

	caPEMBlock := &pem.Block{Type: "CERTIFICATE", Bytes: caBytes}
	caPEM := pem.EncodeToMemory(caPEMBlock)

	keyPEMBlock := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privKey)}
	keyPEM := pem.EncodeToMemory(keyPEMBlock)

	if err := os.WriteFile(caCertPath, caPEM, 0644); err != nil {
		return err
	}
	if err := os.WriteFile(caKeyPath, keyPEM, 0600); err != nil {
		return err
	}

	cm.caCert, _ = x509.ParseCertificate(caBytes)
	cm.caPrivKey = privKey
	cm.caPEM = caPEM

	return nil
}

// Generate Leaf Certificate signed by the Root CA for SNI servernames
func (cm *CertManager) GenerateLeafCert(domain string) (*tls.Certificate, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, err
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: domain,
		},
		DNSNames:    []string{domain, "*." + domain},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(1, 0, 0), // 1 year validity
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, template, cm.caCert, &privKey.PublicKey, cm.caPrivKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign leaf cert: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privKey)})

	// We append CA cert to the leaf cert so clients can verify the trust chain fully
	fullChainPEM := append(certPEM, cm.caPEM...)

	tlsCert, err := tls.X509KeyPair(fullChainPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to load dynamic TLS key pair: %w", err)
	}

	return &tlsCert, nil
}
