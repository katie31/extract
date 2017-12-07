package walg

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"

	"golang.org/x/crypto/openpgp"
)

type Crypter interface {
	IsUsed() bool
	Encrypt(writer io.WriteCloser) (io.WriteCloser, error)
	Decrypt(reader io.ReadCloser) (io.Reader, error)
}

// Crypter incapsulates specific of cypher method
// Includes keys, infrastructutre information etc
// If many encryption methods will be used it worth
// to extract interface
type OpenPGPCrypter struct {
	configured, armed bool
	keyRingId         string

	pubKey    openpgp.EntityList
	secretKey openpgp.EntityList
}

// Function to check necessity of Crypter use
// Must be called prior to any other crypter call
func (crypter *OpenPGPCrypter) IsUsed() bool {
	if !crypter.configured {
		crypter.ConfigureGPGCrypter()
	}
	return crypter.armed
}

// Internal OpenPGPCrypter initialization
func (crypter *OpenPGPCrypter) ConfigureGPGCrypter() {
	crypter.configured = true
	crypter.keyRingId = GetKeyRingId()
	crypter.armed = len(crypter.keyRingId) != 0
}

var CrypterUseMischief = errors.New("Crypter is not checked before use")

// Creates encryption writer from ordinary writer
func (crypter *OpenPGPCrypter) Encrypt(writer io.WriteCloser) (io.WriteCloser, error) {
	if !crypter.configured {
		return nil, CrypterUseMischief
	}
	if crypter.pubKey == nil {
		armour, err := GetPubRingArmour(crypter.keyRingId)
		if err != nil {
			return nil, err
		}

		entitylist, err := openpgp.ReadArmoredKeyRing(bytes.NewReader(armour))
		if err != nil {
			return nil, err
		}
		crypter.pubKey = entitylist
	}

	return &DelayWriteCloser{writer, crypter.pubKey, nil}, nil
}

// Encryption starts writing header immediately.
// But there is a lot of places where writer is instantiated long before pipe
// is ready. This is why here is used special writer, which delays encryption
// initialization before actual write. If no write occurs, initialization
// still is performed, to handle zero-byte files correctly
type DelayWriteCloser struct {
	inner io.WriteCloser
	el    openpgp.EntityList
	outer *io.WriteCloser
}

func (d *DelayWriteCloser) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	if d.outer == nil {
		wc, err0 := openpgp.Encrypt(d.inner, d.el, nil, nil, nil)
		if err0 != nil {
			return 0, err
		}
		d.outer = &wc
	}
	n, err = (*d.outer).Write(p)
	return
}

func (d *DelayWriteCloser) Close() error {
	if d.outer == nil {
		wc, err0 := openpgp.Encrypt(d.inner, d.el, nil, nil, nil)
		if err0 != nil {
			return err0
		}
		d.outer = &wc
	}

	return (*d.outer).Close()
}

// Created decripted reader from ordinary reader
func (crypter *OpenPGPCrypter) Decrypt(reader io.ReadCloser) (io.Reader, error) {
	if !crypter.configured {
		return nil, CrypterUseMischief
	}
	if crypter.secretKey == nil {
		armour, err := GetSecretRingArmour(crypter.keyRingId)
		if err != nil {
			return nil, err
		}

		entitylist, err := openpgp.ReadArmoredKeyRing(bytes.NewReader(armour))
		if err != nil {
			return nil, err
		}
		crypter.secretKey = entitylist
	}

	var md, err0 = openpgp.ReadMessage(reader, crypter.secretKey, nil, nil)
	if err0 != nil {
		return nil, err0
	}

	return md.UnverifiedBody, nil
}

func GetKeyRingId() string {
	return os.Getenv("WALE_GPG_KEY_ID")
}

const gpgBin = "gpg"

type CachedKey struct {
	KeyId string `json:"keyId"`
	Body  []byte `json:"body"`
}

// Here we read armoured version of Key by calling GPG process
func GetPubRingArmour(keyId string) ([]byte, error) {
	var cache CachedKey
	var cacheFilename string

	usr, err := user.Current()
	if err == nil {

		cacheFilename = usr.HomeDir + "/.walg_key_cache"

		file, err := ioutil.ReadFile(cacheFilename)
		// here we ignore whatever error can occur
		if err == nil {
			json.Unmarshal(file, &cache)
			if cache.KeyId == keyId {
				return cache.Body, nil
			}
		}
	}

	out, err := exec.Command(gpgBin, "-a", "--export", keyId).Output()
	if err != nil {
		return nil, err
	}

	cache.KeyId = keyId
	cache.Body = out
	marshal, err := json.Marshal(&cache)
	if err == nil && len(cacheFilename) > 0 {
		ioutil.WriteFile(cacheFilename, marshal, 0644)
	}

	return out, nil
}

func GetSecretRingArmour(keyId string) ([]byte, error) {
	out, err := exec.Command(gpgBin, "-a", "--export-secret-key", keyId).Output()
	if err != nil {
		return nil, err
	}
	return out, nil
}
