package agent

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
	Id             string
	Authentication struct {
		Type constants.AuthenticationType

		StaticToken struct {
			Token string
		} `yaml:"static-token"`

		Certificate struct {
			caCertificatePath       string `yaml:"ca-certificate-path"`
			agentCertificatePath    string `yaml:"agent-certificate-path"`
			agentCertificateKeyPath string `yaml:"agent-certificate-key-path"`
		}
	}
	ServerEndpoint string `yaml:"server-endpoint"`
	LocalEndpoint  string `yaml:"local-endpoint"`
}

func CreateAgent(agentConfig *Config) error {
	if agentConfig == nil {
		return errors.New("missing agent configuration")
	}

	if agentConfig.Authentication.Type == constants.StaticToken {
		if agentConfig.Authentication.StaticToken.Token == "" {
			return errors.New("static-token authentication requires not blank token value")
		}
	}

	if agentConfig.Authentication.Type == constants.Certificate {
		cert, err := tls.LoadX509KeyPair(
			agentConfig.Authentication.Certificate.agentCertificatePath,
			agentConfig.Authentication.Certificate.agentCertificateKeyPath,
		)
		if err != nil {
			return err
		}

		caCertBytes, err := ioutil.ReadFile(agentConfig.Authentication.Certificate.caCertificatePath)
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
			RootCAs:            certPool,
			InsecureSkipVerify: true,
		}
	}

	return nil
}
