package config

import (
	"os"
	"io/ioutil"
)

// CloudSQL environment variable names
const (
	applicationCredentials = "VENDASTA_APPLICATION_CREDENTIALS"
	serviceAccountEmail    = "VENDASTA_SERVICE_ACCOUNT"
	publicKeyID            = "VENDASTA_PUBLIC_KEY_ID"
)

// VendastaApplicationCredentials returns the vendasta application credentials
func VendastaApplicationCredentials() (string, error) {
	bytes, err := ioutil.ReadFile(os.Getenv(applicationCredentials))
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// ServiceAccountEmail returns service account email for the current microservice
func ServiceAccountEmail() string {
	return os.Getenv(serviceAccountEmail)
}

// PublicKeyID returns the public key for the current microservice
func PublicKeyID() (string, error) {
	bytes, err := ioutil.ReadFile(os.Getenv(publicKeyID))
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
