// devcert generates a localhost-only certificate signed by a dedicated development CA.
package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

func main() {
	out := flag.String("out", "../.certs", "directorio de certificados locales")
	flag.Parse()
	if err := generate(*out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("Certificado TLS para localhost generado. Importa ca.crt como raíz de confianza para usarlo en el navegador.")
}
func generate(out string) error {
	for _, name := range []string{"ca.crt", "localhost.crt", "localhost.key"} {
		if _, err := os.Stat(filepath.Join(out, name)); err == nil {
			return fmt.Errorf("%s ya existe; no se sobrescribieron los certificados", name)
		}
	}
	if err := os.MkdirAll(out, 0700); err != nil {
		return err
	}
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	serial := func() *big.Int {
		n, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
		if err != nil {
			panic(err)
		}
		return n
	}
	now := time.Now()
	ca := &x509.Certificate{SerialNumber: serial(), Subject: pkix.Name{CommonName: "Realtime Chat localhost development CA"}, NotBefore: now.Add(-time.Hour), NotAfter: now.AddDate(1, 0, 0), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign, MaxPathLenZero: true}
	caDER, err := x509.CreateCertificate(rand.Reader, ca, ca, &caKey.PublicKey, caKey)
	if err != nil {
		return err
	}
	leaf := &x509.Certificate{SerialNumber: serial(), Subject: pkix.Name{CommonName: "localhost"}, NotBefore: ca.NotBefore, NotAfter: ca.NotAfter, DNSNames: []string{"localhost"}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")}, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	leafDER, err := x509.CreateCertificate(rand.Reader, leaf, ca, &serverKey.PublicKey, caKey)
	if err != nil {
		return err
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(serverKey)
	if err != nil {
		return err
	}
	for _, file := range []struct {
		name, kind string
		data       []byte
	}{{"ca.crt", "CERTIFICATE", caDER}, {"localhost.crt", "CERTIFICATE", leafDER}, {"localhost.key", "PRIVATE KEY", keyDER}} {
		if err := os.WriteFile(filepath.Join(out, file.name), pem.EncodeToMemory(&pem.Block{Type: file.kind, Bytes: file.data}), 0600); err != nil {
			return err
		}
	}
	return nil
}
