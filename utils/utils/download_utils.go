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
	"net/http"
	"os"
)

func DownloadFile(url string, dest string, reporthook interface{}, checksum string) (string, error) {
	return downloadFileUtil(url, dest, reporthook, checksum)
}

func downloadFileUtil(url string, dest string, reporthook interface{}, checksum string) (string, error) {
	tempName := TempFileInWorkDir(dest)
	logrus.Info(tempName)
	logrus.Info(fmt.Sprintf("Downloading %s to %s", url, tempName))
	err := downloadFromURL(url, tempName)
	if err == nil {
		if checksum != "" {
			err1 := validateChecksum(tempName, checksum)
			if err != nil {
				return "", err1
			}
		}
		return tempName, nil
	}
	return tempName, err
}

func checksum(filePath string, digest hash.Hash) (string, error) {
	file, err := os.Open(filePath)
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

func validateChecksum(fileName string, checksumValue string) error {
	digestType := len(checksumValue)
	var digest hash.Hash
	switch digestType {
	case 32:
		digest = md5.New()
	case 40:
		digest = sha1.New()
	case 64:
		digest = sha256.New()
	case 128:
		digest = sha512.New()
	default:
		return errors.New("invalid digestType")
	}
	newValue, err := checksum(fileName, digest)
	if err != nil {
		return err
	}
	if newValue != checksumValue {
		return fmt.Errorf("Invalid checksum [%s]", newValue)
	}
	return nil
}

func downloadFromURL(rawurl string, filepath string) error {
	file, err := os.OpenFile(filepath, os.O_WRONLY, 0666)
	if err == nil {
		defer file.Close()
		response, err1 := http.Get(rawurl)
		if err1 != nil {
			logrus.Error(fmt.Sprintf("Error while downloading error: %s", err1))
			return err1
		}
		defer response.Body.Close()
		n, err := io.Copy(file, response.Body)
		if err != nil {
			logrus.Error(fmt.Sprintf("Error while copy file: %s", err))
			return err
		}
		logrus.Infof("%v bytes downloaded successfully", n)
		return nil
	}
	return err
}
