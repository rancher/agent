package utils

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"fmt"
	"github.com/Sirupsen/logrus"
	"hash"
	"io"
	"os"
)

func downloadFile(url string, dest string, reporthook interface{}, checksum string) (string, error) {
	return downloadFileUtil(url, dest, reporthook, checksum)
}

func downloadFileUtil(url string, dest string, reporthook interface{}, checksum string) (string, error) {
	temp_name := tempFileInWorkDir(dest)
	logrus.Info(fmt.Sprintf("Downloading %s to %s", url, temp_name))
	err := downloadFromUrl(url, temp_name)
	if err == nil {
		if checksum != "" {
			err1 := validateChecksum(temp_name, checksum)
			if err != nil {
				return "", err1
			}
		}
		return temp_name, nil
	}
	return temp_name, err
}

func checksum(file_path string, digest hash.Hash) (string, error) {
	file, err := os.Open(file_path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	_, err1 := io.Copy(digest, file)
	if err1 != nil {
		return "", err1
	}
	return fmt.Sprintf("%x", digest.Sum([]byte{})), nil
}

func validateChecksum(file_name string, checksum_value string) error {
	digest_type := len(checksum_value)
	var digest hash.Hash
	switch digest_type {
	case 32:
		digest = md5.New()
	case 40:
		digest = sha1.New()
	case 64:
		digest = sha256.New()
	case 128:
		digest = sha512.New()
	default:
		return errors.New("invalid digest_type!")
	}
	new_value, err := checksum(file_name, digest)
	if err != nil {
		return err
	}
	if new_value != checksum_value {
		return errors.New(fmt.Sprintf("Invalid checksum [%s]", new_value))
	}
	return nil
}
