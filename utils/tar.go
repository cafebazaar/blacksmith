package utils

import (
	"archive/tar"
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
)

// https://gist.github.com/svett/dc27b7fb04c2549e3ada#file-untarball-go
func Untar(tarball, target string) error {
	reader, err := os.Open(tarball)
	if err != nil {
		return err
	}
	defer reader.Close()
	tarReader := tar.NewReader(reader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		path := filepath.Join(target, header.Name)
		info := header.FileInfo()
		if info.IsDir() {
			if err = os.MkdirAll(path, info.Mode()); err != nil {
				return err
			}
			continue
		}

		file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(file, tarReader)
		if err != nil {
			return err
		}
	}
	return nil
}

// http://www.mrwaggel.be/post/generate-md5-hash-of-a-file/
func HashFileMD5(filePath string) (string, error) {
	// Initialize variable returnMD5String now in case an error has to be returned
	var returnMD5String string

	// Open the passed argument and check for any error
	file, err := os.Open(filePath)
	if err != nil {
		return returnMD5String, err
	}

	// Tell the program to call the following function when the current function returns
	defer file.Close()

	// Open a new hash interface to write to
	hash := md5.New()

	// Copy the file in the hash interface and check for any error
	if _, err := io.Copy(hash, file); err != nil {
		return returnMD5String, err
	}

	// Get the 16 bytes hash
	hashInBytes := hash.Sum(nil)[:16]

	// Convert the bytes to a string
	returnMD5String = hex.EncodeToString(hashInBytes)

	return returnMD5String, nil
}
