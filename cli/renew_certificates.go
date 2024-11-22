package cli

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"io"
	"math/big"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

const (
	defaultIssueYears  = 10
	defaultRenewMonths = 12

	certFilename    = "cert.pem"
	privKeyFilename = "privkey.pem"
)

func RenewCertificatesHandler(u *url.URL) error {
	return RenewCertificates(u.Query().Has("force"))
}

func RenewCertificates(force bool) error {

	rca := nod.Begin("renewing localhost certificates...")
	defer rca.EndWithResult("done")

	// https://medium.com/@shaneutt/create-sign-x509-certificates-in-golang-8ac4ae49f903

	certificatesDir, err := pathways.GetAbsDir(data.Certificates)
	if err != nil {
		return rca.EndWithError(err)
	}

	if cert, err := loadCertificate(certificatesDir); err == nil && cert != nil {
		// check if certificate expires less than a defaultRenewMonths from now
		// and if not - do nothing
		if cert.NotAfter.Before(time.Now().AddDate(0, defaultRenewMonths, 0)) {
			rca.EndWithResult("certificate does not need to be renewed yet")
			return nil
		}
	} else if err != nil {
		return rca.EndWithError(err)
	}

	ca, caPrivKey, err := generateCertAuthorityPrivateKey()
	if err != nil {
		return rca.EndWithError(err)
	}

	cert, certPrivKey, err := generateCertificatePrivateKey()
	if err != nil {
		return rca.EndWithError(err)
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, &certPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return rca.EndWithError(err)
	}

	if err := storeCertPrivKey(certBytes, certPrivKey, certificatesDir); err != nil {
		return rca.EndWithError(err)
	}

	return nil
}

func generateCertAuthorityPrivateKey() (*x509.Certificate, *rsa.PrivateKey, error) {
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(1857),
		Subject: pkix.Name{
			Organization:  []string{"Arelate"},
			Country:       []string{"FR"},
			Province:      []string{"Provence"},
			Locality:      []string{"Arles"},
			StreetAddress: []string{"2 Place Lamartine"},
			PostalCode:    []string{"13004"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(defaultIssueYears, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
	}

	caPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}

	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, nil, err
	}

	caPem := new(bytes.Buffer)
	if err := pem.Encode(caPem, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	}); err != nil {
		return nil, nil, err
	}

	caPrivKeyPem := new(bytes.Buffer)
	if err := pem.Encode(caPrivKeyPem, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey),
	}); err != nil {
		return nil, nil, err
	}

	return ca, caPrivKey, nil
}

func generateCertificatePrivateKey() (*x509.Certificate, *rsa.PrivateKey, error) {
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(1658),
		Subject: pkix.Name{
			Organization:  []string{"Arelate"},
			Country:       []string{"FR"},
			Province:      []string{"Provence"},
			Locality:      []string{"Arles"},
			StreetAddress: []string{"2 Place Lamartine"},
			PostalCode:    []string{"13004"},
		},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(defaultIssueYears, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	certPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}

	return cert, certPrivKey, nil
}

func storeCertPrivKey(certBytes []byte, certPrivKey *rsa.PrivateKey, certificatesDir string) error {

	certPem := new(bytes.Buffer)
	if err := pem.Encode(certPem, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	}); err != nil {
		return err
	}

	certPrivKeyPem := new(bytes.Buffer)
	if err := pem.Encode(certPrivKeyPem, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(certPrivKey),
	}); err != nil {
		return err
	}

	certPath := filepath.Join(certificatesDir, certFilename)
	privKeyPath := filepath.Join(certificatesDir, privKeyFilename)

	certFile, err := os.Create(certPath)
	if err != nil {
		return nil
	}
	defer certFile.Close()

	if _, err := io.Copy(certFile, certPem); err != nil {
		return err
	}

	privKeyFile, err := os.Create(privKeyPath)
	if err != nil {
		return nil
	}
	defer privKeyFile.Close()

	if _, err := io.Copy(privKeyFile, certPrivKeyPem); err != nil {
		return err
	}

	return nil
}

func loadCertificate(certificatesDir string) (*x509.Certificate, error) {

	certPath := filepath.Join(certificatesDir, certFilename)
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		return nil, nil
	}

	var certBytes []byte
	buf := bytes.NewBuffer(certBytes)

	certFile, err := os.Open(certPath)
	if err != nil {
		return nil, err
	}
	defer certFile.Close()

	if _, err := io.Copy(buf, certFile); err != nil {
		return nil, err
	}

	if dec, _ := pem.Decode(buf.Bytes()); dec != nil {
		cert, err := x509.ParseCertificate(dec.Bytes)
		if err != nil {
			return nil, err
		}
		return cert, nil
	}

	return nil, errors.New("no suitable x509 certificate found")
}
