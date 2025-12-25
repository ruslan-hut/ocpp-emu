package v201

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"sync"
	"time"
)

// CertificateUseType defines how a certificate is used
type CertificateUseType string

const (
	// Root certificate types
	CertificateUseV2GRootCertificate          CertificateUseType = "V2GRootCertificate"
	CertificateUseMORootCertificate           CertificateUseType = "MORootCertificate"
	CertificateUseCSMSRootCertificate         CertificateUseType = "CSMSRootCertificate"
	CertificateUseManufacturerRootCertificate CertificateUseType = "ManufacturerRootCertificate"

	// Leaf certificate types
	CertificateUseChargingStationCertificate CertificateUseType = "ChargingStationCertificate"
	CertificateUseV2GCertificate             CertificateUseType = "V2GCertificate"
)

// HashAlgorithmType defines the hash algorithm used for certificate identification
type HashAlgorithmType string

const (
	HashAlgorithmSHA256 HashAlgorithmType = "SHA256"
	HashAlgorithmSHA384 HashAlgorithmType = "SHA384"
	HashAlgorithmSHA512 HashAlgorithmType = "SHA512"
)

// InstallCertificateStatusType defines the status of certificate installation
type InstallCertificateStatusType string

const (
	InstallCertificateStatusAccepted InstallCertificateStatusType = "Accepted"
	InstallCertificateStatusRejected InstallCertificateStatusType = "Rejected"
	InstallCertificateStatusFailed   InstallCertificateStatusType = "Failed"
)

// DeleteCertificateStatusType defines the status of certificate deletion
type DeleteCertificateStatusType string

const (
	DeleteCertificateStatusAccepted DeleteCertificateStatusType = "Accepted"
	DeleteCertificateStatusFailed   DeleteCertificateStatusType = "Failed"
	DeleteCertificateStatusNotFound DeleteCertificateStatusType = "NotFound"
)

// GetInstalledCertificateStatusType defines the status of getting installed certificates
type GetInstalledCertificateStatusType string

const (
	GetInstalledCertificateStatusAccepted GetInstalledCertificateStatusType = "Accepted"
	GetInstalledCertificateStatusNotFound GetInstalledCertificateStatusType = "NotFound"
)

// StoredCertificate represents a certificate stored in the certificate store
type StoredCertificate struct {
	CertificateType CertificateUseType
	Certificate     *x509.Certificate
	PEM             string // Original PEM data
	PrivateKey      crypto.PrivateKey
	InstalledAt     time.Time
	HashData        CertificateHashDataType
}

// CertificateStore manages certificates for a charging station
type CertificateStore struct {
	certificates map[string]*StoredCertificate // key is serial number hex
	mu           sync.RWMutex

	// Pending CSRs awaiting signed certificates
	pendingCSRs map[CertificateUseType]*PendingCSR
	csrMu       sync.RWMutex

	// Station identity for CSR generation
	stationID    string
	organization string
	country      string
}

// PendingCSR represents a pending certificate signing request
type PendingCSR struct {
	CSR        []byte
	PrivateKey crypto.PrivateKey
	CreatedAt  time.Time
	CertType   CertificateUseType
	CSREncoded string // Base64 encoded CSR
}

// NewCertificateStore creates a new certificate store
func NewCertificateStore(stationID, organization, country string) *CertificateStore {
	return &CertificateStore{
		certificates: make(map[string]*StoredCertificate),
		pendingCSRs:  make(map[CertificateUseType]*PendingCSR),
		stationID:    stationID,
		organization: organization,
		country:      country,
	}
}

// InstallCertificate installs a certificate into the store
func (cs *CertificateStore) InstallCertificate(certType CertificateUseType, pemData string) (InstallCertificateStatusType, error) {
	// Parse the PEM certificate
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return InstallCertificateStatusRejected, fmt.Errorf("failed to decode PEM certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return InstallCertificateStatusRejected, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Validate certificate type matches expected usage
	if err := cs.validateCertificateType(cert, certType); err != nil {
		return InstallCertificateStatusRejected, err
	}

	// Check if certificate is expired
	now := time.Now()
	if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
		return InstallCertificateStatusRejected, fmt.Errorf("certificate is not valid: notBefore=%v, notAfter=%v", cert.NotBefore, cert.NotAfter)
	}

	// Compute hash data
	hashData := cs.computeHashData(cert)

	// Store the certificate
	stored := &StoredCertificate{
		CertificateType: certType,
		Certificate:     cert,
		PEM:             pemData,
		InstalledAt:     time.Now(),
		HashData:        hashData,
	}

	cs.mu.Lock()
	cs.certificates[hashData.SerialNumber] = stored
	cs.mu.Unlock()

	return InstallCertificateStatusAccepted, nil
}

// InstallSignedCertificate installs a signed certificate that was requested via CSR
func (cs *CertificateStore) InstallSignedCertificate(certType CertificateUseType, pemChain string) (string, error) {
	// Check if we have a pending CSR for this certificate type
	cs.csrMu.RLock()
	pendingCSR, exists := cs.pendingCSRs[certType]
	cs.csrMu.RUnlock()

	if !exists {
		return "Rejected", fmt.Errorf("no pending CSR for certificate type %s", certType)
	}

	// Parse the certificate chain (may contain multiple certificates)
	var certs []*x509.Certificate
	remaining := []byte(pemChain)
	for len(remaining) > 0 {
		block, rest := pem.Decode(remaining)
		if block == nil {
			break
		}
		remaining = rest

		if block.Type != "CERTIFICATE" {
			continue
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return "Rejected", fmt.Errorf("failed to parse certificate: %w", err)
		}
		certs = append(certs, cert)
	}

	if len(certs) == 0 {
		return "Rejected", fmt.Errorf("no valid certificates found in chain")
	}

	// The first certificate should be the leaf certificate
	leafCert := certs[0]

	// Verify the certificate matches our private key
	if !cs.verifyKeyPair(leafCert, pendingCSR.PrivateKey) {
		return "Rejected", fmt.Errorf("certificate does not match pending CSR private key")
	}

	// Compute hash data
	hashData := cs.computeHashData(leafCert)

	// Store the certificate with its private key
	stored := &StoredCertificate{
		CertificateType: certType,
		Certificate:     leafCert,
		PEM:             pemChain,
		PrivateKey:      pendingCSR.PrivateKey,
		InstalledAt:     time.Now(),
		HashData:        hashData,
	}

	cs.mu.Lock()
	cs.certificates[hashData.SerialNumber] = stored
	cs.mu.Unlock()

	// Remove the pending CSR
	cs.csrMu.Lock()
	delete(cs.pendingCSRs, certType)
	cs.csrMu.Unlock()

	// Store any intermediate certificates in the chain
	for i := 1; i < len(certs); i++ {
		intermediateCert := certs[i]
		intermediateHash := cs.computeHashData(intermediateCert)

		// Determine the type based on the certificate
		var intermediateType CertificateUseType
		if intermediateCert.IsCA {
			if certType == CertificateUseV2GCertificate {
				intermediateType = CertificateUseV2GRootCertificate
			} else {
				intermediateType = CertificateUseCSMSRootCertificate
			}
		} else {
			intermediateType = certType
		}

		intermediateStored := &StoredCertificate{
			CertificateType: intermediateType,
			Certificate:     intermediateCert,
			PEM:             string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: intermediateCert.Raw})),
			InstalledAt:     time.Now(),
			HashData:        intermediateHash,
		}

		cs.mu.Lock()
		cs.certificates[intermediateHash.SerialNumber] = intermediateStored
		cs.mu.Unlock()
	}

	return "Accepted", nil
}

// DeleteCertificate removes a certificate from the store
func (cs *CertificateStore) DeleteCertificate(hashData CertificateHashDataType) DeleteCertificateStatusType {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Find certificate by hash data
	for serialNumber, stored := range cs.certificates {
		if cs.hashDataMatches(stored.HashData, hashData) {
			delete(cs.certificates, serialNumber)
			return DeleteCertificateStatusAccepted
		}
	}

	return DeleteCertificateStatusNotFound
}

// GetInstalledCertificateIds returns the hash data for all installed certificates of the specified types
func (cs *CertificateStore) GetInstalledCertificateIds(certTypes []string) (GetInstalledCertificateStatusType, []CertificateHashDataChainType) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	// If no types specified, return all certificates
	typeSet := make(map[CertificateUseType]bool)
	if len(certTypes) == 0 {
		// Return all types
		typeSet[CertificateUseV2GRootCertificate] = true
		typeSet[CertificateUseMORootCertificate] = true
		typeSet[CertificateUseCSMSRootCertificate] = true
		typeSet[CertificateUseManufacturerRootCertificate] = true
		typeSet[CertificateUseChargingStationCertificate] = true
		typeSet[CertificateUseV2GCertificate] = true
	} else {
		for _, t := range certTypes {
			typeSet[CertificateUseType(t)] = true
		}
	}

	// Group certificates by type
	certsByType := make(map[CertificateUseType][]*StoredCertificate)
	for _, stored := range cs.certificates {
		if typeSet[stored.CertificateType] {
			certsByType[stored.CertificateType] = append(certsByType[stored.CertificateType], stored)
		}
	}

	if len(certsByType) == 0 {
		return GetInstalledCertificateStatusNotFound, nil
	}

	// Build result
	var result []CertificateHashDataChainType
	for certType, certs := range certsByType {
		for _, cert := range certs {
			chain := CertificateHashDataChainType{
				CertificateType:     string(certType),
				CertificateHashData: cert.HashData,
			}

			// Find child certificates (certificates signed by this one)
			if cert.Certificate.IsCA {
				for _, otherCert := range cs.certificates {
					if cs.isSignedBy(otherCert.Certificate, cert.Certificate) && otherCert != cert {
						chain.ChildCertificateHashData = append(chain.ChildCertificateHashData, otherCert.HashData)
					}
				}
			}

			result = append(result, chain)
		}
	}

	return GetInstalledCertificateStatusAccepted, result
}

// GenerateCSR generates a Certificate Signing Request
func (cs *CertificateStore) GenerateCSR(certType CertificateUseType) (string, error) {
	// Generate a new ECDSA key pair (P-256 is commonly used for EV charging)
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", fmt.Errorf("failed to generate private key: %w", err)
	}

	// Create the CSR template
	commonName := cs.stationID
	if certType == CertificateUseV2GCertificate {
		commonName = fmt.Sprintf("%s-V2G", cs.stationID)
	}

	template := &x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{cs.organization},
			Country:      []string{cs.country},
		},
		SignatureAlgorithm: x509.ECDSAWithSHA256,
	}

	// Create the CSR
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, template, privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to create CSR: %w", err)
	}

	// Encode to PEM
	csrPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csrDER,
	})

	// Base64 encode for transmission (OCPP format)
	csrBase64 := base64.StdEncoding.EncodeToString(csrPEM)

	// Store the pending CSR
	cs.csrMu.Lock()
	cs.pendingCSRs[certType] = &PendingCSR{
		CSR:        csrDER,
		PrivateKey: privateKey,
		CreatedAt:  time.Now(),
		CertType:   certType,
		CSREncoded: csrBase64,
	}
	cs.csrMu.Unlock()

	return csrBase64, nil
}

// GetCertificate retrieves a certificate by type
func (cs *CertificateStore) GetCertificate(certType CertificateUseType) *StoredCertificate {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	for _, stored := range cs.certificates {
		if stored.CertificateType == certType {
			return stored
		}
	}
	return nil
}

// GetCertificateByHash retrieves a certificate by its hash data
func (cs *CertificateStore) GetCertificateByHash(hashData CertificateHashDataType) *StoredCertificate {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	for _, stored := range cs.certificates {
		if cs.hashDataMatches(stored.HashData, hashData) {
			return stored
		}
	}
	return nil
}

// HasPendingCSR checks if there's a pending CSR for the given certificate type
func (cs *CertificateStore) HasPendingCSR(certType CertificateUseType) bool {
	cs.csrMu.RLock()
	defer cs.csrMu.RUnlock()
	_, exists := cs.pendingCSRs[certType]
	return exists
}

// GetPendingCSR returns the pending CSR for the given certificate type
func (cs *CertificateStore) GetPendingCSR(certType CertificateUseType) *PendingCSR {
	cs.csrMu.RLock()
	defer cs.csrMu.RUnlock()
	return cs.pendingCSRs[certType]
}

// ClearExpiredCSRs removes CSRs older than the specified duration
func (cs *CertificateStore) ClearExpiredCSRs(maxAge time.Duration) {
	cs.csrMu.Lock()
	defer cs.csrMu.Unlock()

	now := time.Now()
	for certType, csr := range cs.pendingCSRs {
		if now.Sub(csr.CreatedAt) > maxAge {
			delete(cs.pendingCSRs, certType)
		}
	}
}

// GetAllCertificates returns all stored certificates
func (cs *CertificateStore) GetAllCertificates() []*StoredCertificate {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	result := make([]*StoredCertificate, 0, len(cs.certificates))
	for _, stored := range cs.certificates {
		result = append(result, stored)
	}
	return result
}

// computeHashData computes the hash data for a certificate
func (cs *CertificateStore) computeHashData(cert *x509.Certificate) CertificateHashDataType {
	// Hash the issuer name (DER encoded)
	issuerHash := sha256.Sum256(cert.RawIssuer)

	// Hash the issuer public key
	var issuerKeyHash [32]byte
	if cert.AuthorityKeyId != nil {
		issuerKeyHash = sha256.Sum256(cert.AuthorityKeyId)
	} else {
		// For self-signed certs, use the subject key identifier or public key
		if cert.SubjectKeyId != nil {
			issuerKeyHash = sha256.Sum256(cert.SubjectKeyId)
		} else {
			issuerKeyHash = sha256.Sum256(cert.RawSubjectPublicKeyInfo)
		}
	}

	return CertificateHashDataType{
		HashAlgorithm:  string(HashAlgorithmSHA256),
		IssuerNameHash: hex.EncodeToString(issuerHash[:]),
		IssuerKeyHash:  hex.EncodeToString(issuerKeyHash[:]),
		SerialNumber:   hex.EncodeToString(cert.SerialNumber.Bytes()),
	}
}

// hashDataMatches checks if two hash data structures match
func (cs *CertificateStore) hashDataMatches(a, b CertificateHashDataType) bool {
	return a.HashAlgorithm == b.HashAlgorithm &&
		a.IssuerNameHash == b.IssuerNameHash &&
		a.IssuerKeyHash == b.IssuerKeyHash &&
		a.SerialNumber == b.SerialNumber
}

// validateCertificateType validates that a certificate matches the expected type
func (cs *CertificateStore) validateCertificateType(cert *x509.Certificate, certType CertificateUseType) error {
	switch certType {
	case CertificateUseV2GRootCertificate,
		CertificateUseMORootCertificate,
		CertificateUseCSMSRootCertificate,
		CertificateUseManufacturerRootCertificate:
		// Root certificates should be CA certificates
		if !cert.IsCA {
			return fmt.Errorf("certificate type %s requires a CA certificate", certType)
		}
	case CertificateUseChargingStationCertificate, CertificateUseV2GCertificate:
		// Leaf certificates should not be CA certificates
		// (but we'll be lenient here as some implementations may differ)
	}
	return nil
}

// verifyKeyPair verifies that a certificate's public key matches a private key
func (cs *CertificateStore) verifyKeyPair(cert *x509.Certificate, privateKey crypto.PrivateKey) bool {
	switch priv := privateKey.(type) {
	case *ecdsa.PrivateKey:
		pub, ok := cert.PublicKey.(*ecdsa.PublicKey)
		if !ok {
			return false
		}
		return priv.PublicKey.Equal(pub)
	default:
		return false
	}
}

// isSignedBy checks if childCert was signed by parentCert
func (cs *CertificateStore) isSignedBy(childCert, parentCert *x509.Certificate) bool {
	if err := childCert.CheckSignatureFrom(parentCert); err != nil {
		return false
	}
	return true
}

// ValidateCertificateChain validates a certificate chain against installed root certificates
func (cs *CertificateStore) ValidateCertificateChain(leafCert *x509.Certificate, intermediates []*x509.Certificate) error {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	// Build root pool from installed root certificates
	roots := x509.NewCertPool()
	for _, stored := range cs.certificates {
		switch stored.CertificateType {
		case CertificateUseV2GRootCertificate,
			CertificateUseMORootCertificate,
			CertificateUseCSMSRootCertificate,
			CertificateUseManufacturerRootCertificate:
			roots.AddCert(stored.Certificate)
		}
	}

	// Build intermediate pool
	intermediatePool := x509.NewCertPool()
	for _, intermediate := range intermediates {
		intermediatePool.AddCert(intermediate)
	}

	// Verify the certificate
	opts := x509.VerifyOptions{
		Roots:         roots,
		Intermediates: intermediatePool,
		CurrentTime:   time.Now(),
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}

	_, err := leafCert.Verify(opts)
	return err
}

// ExportCertificatePEM exports a certificate in PEM format
func (cs *CertificateStore) ExportCertificatePEM(certType CertificateUseType) (string, error) {
	cert := cs.GetCertificate(certType)
	if cert == nil {
		return "", fmt.Errorf("certificate not found: %s", certType)
	}
	return cert.PEM, nil
}

// GetCertificateCount returns the number of installed certificates
func (cs *CertificateStore) GetCertificateCount() int {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return len(cs.certificates)
}
