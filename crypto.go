package walg

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

// GetKeyRingId extracts name of a key to use from env variable
func GetKeyRingId() string {
	return os.Getenv("WALE_GPG_KEY_ID")
}

const gpgBin = "gpg"

// CachedKey is the data transfer object describing format of key ring cache
type CachedKey struct {
	KeyId string `json:"keyId"`
	Body  []byte `json:"body"`
}

// Here we read armoured version of Key by calling GPG process
func getPubRingArmour(keyId string) ([]byte, error) {
	var cache CachedKey
	var cacheFilename string

	usr, err := user.Current()
	if err == nil {
		cacheFilename = filepath.Join(usr.HomeDir, ".walg_key_cache")
		file, err := ioutil.ReadFile(cacheFilename)
		// here we ignore whatever error can occur
		if err == nil {
			json.Unmarshal(file, &cache)
			if cache.KeyId == keyId && len(cache.Body) > 0 { // don't return an empty cached value
				return cache.Body, nil
			}
		}
	}

	cmd := exec.Command(gpgBin, "-a", "--export", keyId)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	if stderr.Len() > 0 { // gpg -a --export <key-id> reports error on stderr and exits == 0 if the key isn't found
		return nil, errors.New(strings.TrimSpace(stderr.String()))
	}

	cache.KeyId = keyId
	cache.Body = out
	marshal, err := json.Marshal(&cache)
	if err == nil && len(cacheFilename) > 0 {
		ioutil.WriteFile(cacheFilename, marshal, 0644)
	}

	return out, nil
}

func getSecretRingArmour(keyId string) ([]byte, error) {
	out, err := exec.Command(gpgBin, "-a", "--export-secret-key", keyId).Output()
	if err != nil {
		return nil, err
	}
	return out, nil
}
