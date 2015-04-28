package httpmux

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	//"log"
	"math/big"
	"net"
	"time"
)

func NewSelfSignedRSAConfig(rsaBits int) (*tls.Config, error) {
	now := time.Now()

	priv, err := rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		return nil, err
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"SmartWeb"},
		},
		NotBefore: now.Add(-1 * time.Minute),
		NotAfter:  now.Add(5 * time.Minute), // FIXME allow for more than 5 minutes of uptime

		IsCA:                  true,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,

		DNSNames: []string{"*"},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, err
	}

	certBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyBytes := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	cert, err := tls.X509KeyPair(certBytes, keyBytes)
	if err != nil {
		return nil, err
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	return config, nil
}

type Listener struct {
	net.Listener
	config *tls.Config
}

func NewListener(listener net.Listener, certFile, keyFile string) (*Listener, error) {
	config := &tls.Config{}

	var err error
	config.Certificates = make([]tls.Certificate, 1)
	config.Certificates[0], err = tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	return NewListenerConfig(listener, config), nil
}

func NewListenerConfig(listener net.Listener, config *tls.Config) *Listener {
	if config.NextProtos == nil {
		config.NextProtos = []string{"http/1.1"}
	}

	return &Listener{
		listener,
		config,
	}
}

func (self *Listener) Accept() (net.Conn, error) {
	c, err := self.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return DispatchConnection(c, self.config)
}

type conn struct {
	io.Reader
	net.Conn
	config *tls.Config
	br     *bytes.Reader
}

func (c *conn) Read(b []byte) (int, error) {
	n, e := c.Reader.Read(b)
	//log.Printf("%v, %v = Read(%v)\n", n, e, b)
	return n, e
}

func (c *conn) WriteTo(w io.Writer) (int64, error) {
	//log.Println("WriteTo")
	n, err := c.br.WriteTo(w)
	if err != nil {
		return n, err
	}

	m, err := io.Copy(w, c.Conn)
	return n + m, err
}

func DispatchConnection(c net.Conn, config *tls.Config) (net.Conn, error) {
	var buf [256]byte
	var zeroTime time.Time

	c.SetReadDeadline(zeroTime)
	m, err := io.ReadFull(c, buf[:1])
	if err != nil {
		return nil, err
	}

	c.SetReadDeadline(time.Now())
	n, err := c.Read(buf[1:])
	if err != nil {
		if e, ok := err.(net.Error); !ok || !e.Timeout() {
			return nil, err
		}
	}

	//log.Printf("%v + %v = Read(%v)\n", m, n, buf[:m+n])

	c.SetReadDeadline(zeroTime)

	br := bytes.NewReader(buf[:m+n])
	c = &conn{
		io.MultiReader(br, c),
		c,
		config,
		br,
	}

	if isHttp(buf[:m+n]) {
		return c, nil
	} else {
		return tls.Server(c, config), nil
	}
}

func isHttp(buf []byte) bool {
	for _, c := range buf {
		if c != '\t' && (c < 32 || c >= 127) {
			return false
		}
	}
	return true
}
