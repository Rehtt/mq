package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"os"
	"time"
)

func InitTlsConfig(certFile, keyFile string) (*tls.Config, error) {
	_, certErr := os.Stat(certFile)
	_, keyErr := os.Stat(keyFile)
	if os.IsNotExist(certErr) || os.IsNotExist(keyErr) {
		cert, key := GenerateTLSConfig()
		if err := os.WriteFile(certFile, cert, 0o644); err != nil {
			return nil, err
		}
		if err := os.WriteFile(keyFile, key, 0o644); err != nil {
			return nil, err
		}
	}
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"mq"},
		MinVersion:   tls.VersionTLS13,
	}, nil
}

// 自签证书
func GenerateTLSConfig() ([]byte, []byte) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	return certPEM, keyPEM
	// tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	// if err != nil {
	// 	panic(err)
	// }
	// return &tls.Config{
	// 	Certificates: []tls.Certificate{tlsCert},
	// 	NextProtos:   []string{"rehtt-mq"},
	// }
}
