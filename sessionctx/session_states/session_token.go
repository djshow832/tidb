package session_states

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pingcap/errors"
	"github.com/pingcap/tidb/util/logutil"
	"go.uber.org/zap"
)

const (
	tokenLifetime = time.Minute
)

type SessionToken struct {
	Username   string    `json:"username"`
	SignTime   time.Time `json:"sign-time"`
	ExpireTime time.Time `json:"expire-time"`
	Signature  []byte    `json:"signature,omitempty"`
}

func CreateSessionToken(username string) (*SessionToken, error) {
	signingCert := GetSigningCert()
	if signingCert == nil {
		return nil, errors.New("creating session token needs a signing certificate")
	}
	now := time.Now()
	token := &SessionToken{
		Username:   username,
		SignTime:   now,
		ExpireTime: now.Add(tokenLifetime),
	}
	tokenBytes, err := json.Marshal(token)
	if err != nil {
		return nil, errors.Trace(err)
	}
	token.Signature, err = signingCert.Sign(tokenBytes)
	return token, err
}

func ValidateSessionToken(tokenBytes []byte, username string) (err error) {
	signingCert := GetSigningCert()
	if signingCert == nil {
		return errors.New("verifying session token needs a signing certificate")
	}
	var token SessionToken
	if err = json.Unmarshal(tokenBytes, &token); err != nil {
		return errors.Trace(err)
	}
	signature := token.Signature
	token.Signature = nil
	if tokenBytes, err = json.Marshal(token); err != nil {
		return errors.Trace(err)
	}
	if err = signingCert.CheckSignature(tokenBytes, signature); err != nil {
		return errors.Wrap(err, "invalid signature")
	}
	now := time.Now()
	if now.After(token.ExpireTime) {
		return errors.New("token is expired")
	}
	if token.SignTime.Add(tokenLifetime).Before(now) {
		return errors.New("token lifetime is too long")
	}
	if !strings.EqualFold(username, token.Username) {
		return errors.New("username does not match")
	}
	return nil
}

var (
	globalSigningCert atomic.Value
)

func GetSigningCert() *SigningCert {
	return globalSigningCert.Load().(*SigningCert)
}

type SigningCert struct {
	sync.RWMutex
	certPath string
	keyPath  string
	cert     *x509.Certificate
	privKey  crypto.PrivateKey
}

func InitSigningCert(certPath, keyPath string) error {
	if len(certPath) == 0 || len(keyPath) == 0 {
		return nil
	}
	signingCert := &SigningCert{
		certPath: certPath,
		keyPath:  keyPath,
	}
	if err := signingCert.loadCert(); err != nil {
		return err
	}
	go func() {
		for range time.Tick(time.Hour * 24 * 30) { // 30 days
			err := signingCert.loadCert()
			if err != nil {
				logutil.BgLogger().Error("automatically loading signing certificates failed", zap.Error(err))
			} else {
				logutil.BgLogger().Info("automatically loading signing certificates succeeded")
			}
		}
	}()
	globalSigningCert.Store(signingCert)
	return nil
}

// TODO: the cert file may happen to be replaced between signing and checking, so we need to keep the old cert for a while.
func (sc *SigningCert) loadCert() error {
	tlsCert, err := tls.LoadX509KeyPair(sc.certPath, sc.keyPath)
	if err != nil {
		return errors.Wrap(err, "load x509 failed")
	}
	var cert *x509.Certificate
	if tlsCert.Leaf != nil {
		cert = tlsCert.Leaf
	} else {
		if cert, err = x509.ParseCertificate(tlsCert.Certificate[0]); err != nil {
			return errors.Wrap(err, "parse x509 cert failed")
		}
	}
	sc.Lock()
	sc.cert = cert
	sc.privKey = tlsCert.PrivateKey
	sc.Unlock()
	return nil
}

func (sc *SigningCert) Sign(content []byte) ([]byte, error) {
	var (
		signer crypto.Signer
		opts   crypto.SignerOpts
	)
	sc.RLock()
	defer sc.RUnlock()
	switch sc.privKey.(type) {
	case ed25519.PrivateKey:
		signer = sc.privKey.(ed25519.PrivateKey)
	case *rsa.PrivateKey:
		signer = sc.privKey.(*rsa.PrivateKey)
		var pssHash crypto.Hash
		switch sc.cert.SignatureAlgorithm {
		case x509.SHA256WithRSAPSS:
			pssHash = crypto.SHA256
		case x509.SHA384WithRSAPSS:
			pssHash = crypto.SHA384
		case x509.SHA512WithRSAPSS:
			pssHash = crypto.SHA512
		}
		if pssHash != 0 {
			h := pssHash.New()
			h.Write(content)
			content = h.Sum(nil)
			opts = &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash, Hash: pssHash}
		} else {
			hashed := sha256.Sum256(content)
			content = hashed[:]
			opts = crypto.SHA256
		}
	case *ecdsa.PrivateKey:
		signer = sc.privKey.(*ecdsa.PrivateKey)
	default:
		return nil, errors.Errorf("not supported private key type '%s' for signing", sc.cert.SignatureAlgorithm.String())
	}
	return signer.Sign(rand.Reader, content, opts)
}

// CheckSignature checks the signature and the content.
func (sc *SigningCert) CheckSignature(content, signature []byte) error {
	sc.RLock()
	defer sc.RUnlock()
	return sc.cert.CheckSignature(sc.cert.SignatureAlgorithm, content, signature)
}
