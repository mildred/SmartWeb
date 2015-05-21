package server2

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"log"
	"math/big"
	"net/http"
	"strconv"
	"time"
)

var keygen_form = `
<!DOCTYPE html>
<html>
	<body>
		<form action="?keygen" method="post">
			<input type="input" name="name" placeholder="Common Name" />
			<keygen name="key" challenge="c h a ll e n g e/strin g">
			valid
			<input type="number" name="days" value="365" />
			days
			<input type="submit">
		</form>
	</body>
</html>`

type SubjectPublicKeyInfo struct {
	Algorithm        pkix.AlgorithmIdentifier
	SubjectPublicKey asn1.BitString
}

type PublicKeyAndChallenge struct {
	Spki      SubjectPublicKeyInfo
	Challenge string
}

type SignedPublicKeyAndChallenge struct {
	PublicKeyAndChallenge PublicKeyAndChallenge
	SignitureAlgorithm    pkix.AlgorithmIdentifier
	Signiture             asn1.BitString
}

func (server SmartServer) handleGETKeygen(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "text/html; charset=utf-8")
	res.Write([]byte(keygen_form))
}

func (server SmartServer) handlePOSTKeygen(res http.ResponseWriter, req *http.Request) {
	//b, _ := ioutil.ReadAll(req.Body)
	//log.Println(string(b))
	req.ParseForm()
	b64key := req.Form["key"][0]
	key, err := base64.StdEncoding.DecodeString(b64key)
	if err != nil {
		handleError(res, 400, err.Error())
		return
	}

	var keyAndChallenge SignedPublicKeyAndChallenge
	rest, err := asn1.Unmarshal(key, &keyAndChallenge)

	log.Printf("key: %#v\nkeyAndChallenge: %#v\nrest: %#v\nerr: %#v\n", key, keyAndChallenge, rest, err)

	der_pubkey, err := asn1.Marshal(keyAndChallenge.PublicKeyAndChallenge.Spki)
	if err != nil {
		handleError(res, 500, err.Error())
		return
	}

	pubkey, err := x509.ParsePKIXPublicKey(der_pubkey)
	if err != nil {
		handleError(res, 500, err.Error())
		return
	}

	log.Printf("pubkey: %#v\n", pubkey)

	rsapubkey, ok := pubkey.(*rsa.PublicKey)
	if !ok {
		handleError(res, 400, "Expect RSA Public Key")
		return
	}

	log.Printf("rsapubkey: %#v\n", rsapubkey)

	now := time.Now()

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		handleError(res, 400, err.Error())
	}

	days, e := strconv.ParseInt(req.Form.Get("days"), 10, 0)
	if e != nil {
		days = 365
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: req.Form.Get("name"),
		},
		NotBefore: now.Add(-1 * 24 * time.Hour),
		NotAfter:  now.Add(time.Duration(days) * 24 * time.Hour),

		IsCA:                  true,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	log.Printf("cert: %#v\npubkey: %#v\nprivkey: %#v\n", server.Certificate, pubkey, server.PrivateKey)

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, server.Certificate, pubkey, server.PrivateKey)
	if err != nil {
		handleError(res, 400, err.Error())
	}

	//rsa.VerifyPKCS1v15()

	/*
		The public key and challenge string are DER encoded as PublicKeyAndChallenge,
		and then digitally signed with the private key to produce a
		SignedPublicKeyAndChallenge. The SignedPublicKeyAndChallenge is Base64
		encoded, and the ASCII data is finally submitted to the server as the value
		of a form name/value pair, where the name is name as specified by the name
		attribute of the keygen element. If no challenge string is provided, then
		it will be encoded as an IA5STRING of length zero.
	*/

	log.Printf("Keygen From: %#v\n", b64key)
	res.Header().Set("Content-Type", "application/x-x509-user-cert")
	//res.WriteHeader(http.StatusMethodNotAllowed)
	res.Write(derBytes)
}

func (server SmartServer) handleKeygen(res http.ResponseWriter, req *http.Request) {
	if req.Method == "GET" {
		server.handleGETKeygen(res, req)
	} else if req.Method == "POST" {
		server.handlePOSTKeygen(res, req)
	} else {
		res.WriteHeader(http.StatusMethodNotAllowed)
	}
}
