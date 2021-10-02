package server

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/pkg/errors"
	"io/ioutil"
	"tunnel-transporter/constants"
)

var (
	TlsConfig *tls.Config
)

type Config struct {
	Port           uint16
	Authentication struct {
		Type constants.AuthenticationType

		StaticToken struct {
			Token string
		} `yaml:"static-token"`

		Certificate struct {
			caCertificatePath        string `yaml:"ca-certificate-path"`
			serverCertificatePath    string `yaml:"server-certificate-path"`
			serverCertificateKeyPath string `yaml:"server-certificate-key-path"`
		}
	}
}

func CreateServer(serverConfig *Config) error {
	if serverConfig == nil {
		return errors.New("missing server configuration")
	}

	if serverConfig.Authentication.Type == constants.StaticToken {
		if serverConfig.Authentication.StaticToken.Token == "" {
			return errors.New("static-token authentication requires not blank token value")
		}
	}

	if serverConfig.Authentication.Type == constants.Certificate {
		cert, err := tls.LoadX509KeyPair(
			serverConfig.Authentication.Certificate.serverCertificatePath,
			serverConfig.Authentication.Certificate.serverCertificateKeyPath)
		if err != nil {
			return err
		}

		caCertBytes, err := ioutil.ReadFile(serverConfig.Authentication.Certificate.caCertificatePath)
		if err != nil {
			return err
		}

		certPool := x509.NewCertPool()
		ok := certPool.AppendCertsFromPEM(caCertBytes)
		if !ok {
			return errors.New("error while appending CA certificate to pool")
		}

		TlsConfig = &tls.Config{
			Certificates:       []tls.Certificate{cert},
			ClientCAs:          certPool,
			InsecureSkipVerify: true,
			ClientAuth:         tls.RequireAndVerifyClientCert,
		}
	}

	return nil
}
