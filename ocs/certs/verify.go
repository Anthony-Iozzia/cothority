package certs

import (
	"bytes"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"errors"

	"go.dedis.ch/cothority/v3"
	"go.dedis.ch/kyber/v3"
)

var (
	// selection of OID numbers is not random See documents
	// https://tools.ietf.org/html/rfc5280#page-49
	// https://tools.ietf.org/html/rfc7229
	WriteIdOID      = asn1.ObjectIdentifier{1, 3, 6, 1, 5, 5, 7, 13, 1}
	EphemeralKeyOID = asn1.ObjectIdentifier{1, 3, 6, 1, 5, 5, 7, 13, 2}
)

// Verify takes a root certificate and the certificate to verify. It then verifies
// the certificates with regard to the signature of the root-certificate to the
// authCert.
// ocsID is the ID of the LTS cothority, while U is the commitment to the secret.
func Verify(vo x509.VerifyOptions, cert *x509.Certificate, X kyber.Point, U kyber.Point) (err error) {
	wid, err := GetExtensionFromCert(cert, WriteIdOID)
	if err != nil {
		return Erret(err)
	}
	err = WriteID(wid).Verify(X, U)
	if err != nil {
		return Erret(err)
	}

	unmarkUnhandledCriticalExtension(cert, WriteIdOID)
	unmarkUnhandledCriticalExtension(cert, EphemeralKeyOID)

	_, err = cert.Verify(vo)
	return Erret(err)
}

// WriteID is the ID that will be revealed to the X509 verification method.
type WriteID []byte

func NewWriteID(X, U kyber.Point) (WriteID, error) {
	if X == nil || U == nil {
		return nil, errors.New("X or U is missing")
	}
	wid := sha256.New()
	_, err := X.MarshalTo(wid)
	if err != nil {
		return nil, Erret(err)
	}
	_, err = U.MarshalTo(wid)
	if err != nil {
		return nil, Erret(err)
	}
	return wid.Sum(nil), nil
}

func (wid WriteID) Verify(X, U kyber.Point) error {
	other, err := NewWriteID(X, U)
	if err != nil {
		return Erret(err)
	}
	if bytes.Compare(wid, other) != 0 {
		return errors.New("not the same writeID")
	}
	return nil
}

func GetPointFromCert(certBuf []byte, extID asn1.ObjectIdentifier) (kyber.Point, error) {
	cert, err := x509.ParseCertificate(certBuf)
	if err != nil {
		return nil, Erret(err)
	}
	secret := cothority.Suite.Point()
	secretBuf, err := GetExtensionFromCert(cert, extID)
	if err != nil {
		return nil, Erret(err)
	}
	err = secret.UnmarshalBinary(secretBuf)
	return secret, Erret(err)
}

func GetExtensionFromCert(cert *x509.Certificate, extID asn1.ObjectIdentifier) ([]byte, error) {
	var buf []byte
	for _, ext := range cert.Extensions {
		if ext.Id.Equal(extID) {
			buf = ext.Value
			break
		}
	}
	if buf == nil {
		return nil, errors.New("didn't find extension in certificate")
	}
	return buf, nil
}

func unmarkUnhandledCriticalExtension(cert *x509.Certificate, id asn1.ObjectIdentifier) {
	for i, extension := range cert.UnhandledCriticalExtensions {
		if id.Equal(extension) {
			cert.UnhandledCriticalExtensions = append(cert.UnhandledCriticalExtensions[0:i],
				cert.UnhandledCriticalExtensions[i+1:]...)
			return
		}
	}
}

func getExtension(certificate *x509.Certificate, id asn1.ObjectIdentifier) *pkix.Extension {

	for _, ext := range certificate.Extensions {
		if ext.Id.Equal(id) {
			return &ext
		}
	}

	return nil
}
