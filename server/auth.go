package server

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

type Authenticator struct {
	nonces map[string]Nonce
	mutex  sync.Mutex
}

type AuthDomain struct {
	Realm string
	Entry Entry
}

type Nonce struct {
	Time     time.Time
	Validity time.Duration
	Opaque   string
	Domain   string
	Realm    string
}

func CreateAuthenticator() Authenticator {
	return Authenticator{
		nonces: make(map[string]Nonce),
	}
}

var nonceValidity = time.Minute * 5
var AnonymousAuthDomain string = "anonymous"

func RandomKey() (string, error) {
	var data [12]byte
	_, err := io.ReadFull(rand.Reader, data[:])
	return base64.StdEncoding.EncodeToString(data[:]), err
}

func escape(s string) string {
	return strings.Replace(strings.Replace(s, "\\", "\\\\", -1), "\"", "\\\"", -1)
}

/*
 h function for MD5 algorithm (returns a lower-case hex MD5 digest)
*/
func h(data string) string {
	digest := md5.New()
	digest.Write([]byte(data))
	return fmt.Sprintf("%x", digest.Sum(nil))
}

func getAuthParams(req *http.Request) []AuthMethod {
	var methods []AuthMethod

	for _, h := range req.Header["Authorization"] {
		parser := HeaderParser{Buffer: h}
		parser.Init()
		if err := parser.Parse(); err != nil {
			log.Println(err)
		} else {
			parser.Execute()
		}
		for _, m := range parser.Methods {
			methods = append(methods, m)
		}
	}
	return methods
}

func (auth *Authenticator) purge(now time.Time) {
	for k, nonce := range auth.nonces {
		if nonce.Time.Add(nonce.Validity).Before(now) {
			//log.Printf("Purge nonce %s\n", k)
			delete(auth.nonces, k)
		}
	}
}

func (auth *Authenticator) generateNonce(domain string, realm string, time time.Time, valid time.Duration) (string, string, error) {
	key, err := RandomKey()
	if err != nil {
		return "", "", err
	}

	opaque, err := RandomKey()
	if err != nil {
		return "", "", err
	}

	//log.Printf("Generate nonce %s\n", key)

	auth.nonces[key] = Nonce{
		Time:     time,
		Validity: valid,
		Opaque:   opaque,
		Domain:   domain,
		Realm:    realm,
	}
	return key, opaque, nil
}

func (auth *Authenticator) checkNonce(noncekey string, nonce_count string, now time.Time) (bool, string, string, string) {
	// FIXME do something with nonce_count
	nonce, ok := auth.nonces[noncekey]
	if !ok {
		return false, "", "", ""
	}
	opaque := nonce.Opaque
	auth_domain := nonce.Domain
	auth_realm := nonce.Realm
	valid := nonce.Time.Add(nonce.Validity).After(now)
	if !valid {
		delete(auth.nonces, noncekey)
		//log.Printf("Purge stale nonce %s\n", noncekey)
	}
	return valid, opaque, auth_domain, auth_realm
}

func (Auth *Authenticator) checkRequest(entry Entry, req *http.Request, now time.Time) (bool, bool, bool, []error) {
	var errors []error
	var stale = false
	var found_auth_all = false

	auth_params := getAuthParams(req)
	for _, auth := range auth_params {
		if auth.Name == "Digest" {
			algorithm := auth.Params["algorithm"]
			qop := auth.Params["qop"]
			opaque := auth.Params["opaque"]
			nonce := auth.Params["nonce"]
			response := auth.Params["response"]
			username := auth.Params["username"]
			digest_uri := auth.Params["uri"]
			cnonce := auth.Params["cnonce"]
			nonce_count := auth.Params["nc"]

			if algorithm != "MD5" || qop != "auth" {
				//log.Println("Invalid algorithm or qop")
				continue
			}

			ok, nonce_opaque, auth_domain, auth_realm := Auth.checkNonce(nonce, nonce_count, now)
			if !ok {
				stale = true
				//log.Println("Stale nonce " + nonce + " count: " + nonce_count)
				continue
			} else if opaque != nonce_opaque {
				//log.Printf("Opaque data invalid %s %s\n", opaque, nonce_opaque)
				continue
			}

			var password string
			if found, pass, err := getAuthCreds(entry, auth_domain, auth.Name, username); err != nil {
				errors = append(errors, err)
				continue
			} else if !found {
				log.Printf("Digest Authentication failure, %s no username %s\n", req.RequestURI, username)
				continue
			} else {
				password = string(pass)
			}

			HA1 := h(username + ":" + auth_realm + ":" + password)
			HA2 := h(req.Method + ":" + digest_uri)
			KD := h(strings.Join([]string{
				HA1, nonce, nonce_count, cnonce, qop, HA2}, ":"))

			if KD != response {
				log.Printf("Digest Authentication failure %s %s != %s\n", req.RequestURI, KD, response)
				continue
			}

			allow, found_auth, err := getAuthPerms(entry, auth_domain, req.Method)
			found_auth_all = found_auth_all || found_auth
			if err != nil {
				errors = append(errors, err)
			}

			if allow {
				return true, found_auth_all, false, errors
			}
		}
	}

	allow, found_auth, err := getAuthPerms(entry, AnonymousAuthDomain, req.Method)
	found_auth_all = found_auth_all || found_auth
	if err != nil {
		errors = append(errors, err)
	}

	return allow, found_auth_all, stale, errors
}

func (Auth *Authenticator) Authenticate(entry Entry, res http.ResponseWriter, req *http.Request) bool {
	Auth.mutex.Lock()
	defer Auth.mutex.Unlock()

	now := time.Now()
	Auth.purge(now)

	authorized, found_auth, stale, errors := Auth.checkRequest(entry, req, now)
	if len(errors) > 0 {
		for err := range errors {
			fmt.Println(err)
		}
	}

	auths, auth_errs := authList(entry)
	if len(auth_errs) > 0 {
		for err := range auth_errs {
			log.Println(err)
		}
	}

	for _, auth := range auths {
		domain, err := getAuthDomain(entry, auth)
		if err != nil {
			log.Println(err)
		}
		staleStr := ""
		if stale {
			staleStr = `, stale="true"`
		}

		if nonce, opaque, nonce_err := Auth.generateNonce(auth, domain.Realm, now, nonceValidity); nonce_err != nil {
			log.Println(nonce_err)
		} else {
			res.Header().Add("WWW-Authenticate",
				fmt.Sprintf(`Digest realm="%s", domain="%s", nonce="%s", opaque="%s"%s, algorithm="MD5", qop="auth"`,
					escape(domain.Realm), escape(getPath(domain.Entry)), nonce, opaque, staleStr))
		}
	}

	if found_auth {
		return authorized
	} else {
		return len(auths) == 0
	}
}

func authList(entry Entry) ([]string, []error) {
	var errors []error
	var res_auths []string

	for ent := entry; ent != nil; ent = ent.Parent(true) {
		auths := entry.Parameters().Child("auth")
		auth_list, err := auths.Children()
		if err != nil && os.IsExist(err) {
			errors = append(errors, err)
			continue
		}
		for _, auth := range auth_list {
			found := false
			if auth.Name() == AnonymousAuthDomain {
				continue
			}
			for _, v := range res_auths {
				if v == auth.Name() {
					found = true
					break
				}
			}
			if !found {
				res_auths = append(res_auths, auth.Name())
			}
		}
		if !auths.Child("inherit").Exists() && auths.Exists() {
			break
		}
	}

	return res_auths, errors
}

func getAuthDomain(entry Entry, authname string) (AuthDomain, error) {
	var domain AuthDomain
	var ent Entry
	var err error
	var haveRealm = false

	for ent = entry; ent != nil; ent = ent.Parent(true) {
		auth := entry.Parameters().Child("auth").Child(authname)
		if !auth.Exists() {
			continue
		}

		if !haveRealm {
			data, e := auth.Child("realm").Read()
			if e == nil {
				domain.Realm = string(data)
				haveRealm = true
			} else if e != nil && os.IsExist(err) {
				domain.Realm = authname
				err = e
				haveRealm = true
			}
		}

		if !auth.Child("inherit").Exists() {
			break
		}
	}

	domain.Entry = ent
	return domain, nil
}

func getAuthCreds(entry Entry, auth string, method string, username string) (bool, []byte, error) {
	for ent := entry; ent != nil; ent = ent.Parent(true) {
		auth := entry.Parameters().Child("auth").Child(auth)
		if !auth.Exists() {
			continue
		}

		users := auth.Child(method + ".users")
		user := users.Child(username)

		data, e := user.Read()
		if e == nil {
			return true, data, nil
		} else if e != nil && os.IsExist(e) {
			return false, []byte{}, e
		}

		if !auth.Child("inherit").Exists() {
			break
		}
	}
	return false, []byte{}, nil
}

func getAuthPerms(entry Entry, auth string, method string) (bool, bool, error) {
	for ent := entry; ent != nil; ent = ent.Parent(true) {
		auth := ent.Parameters().Child("auth").Child(auth)

		if data, err := auth.Child(method + ".perm").Read(); err == nil {
			//log.Printf("%v %v %v\n", auth, string(data), strings.TrimSpace(string(data)) == "allow")
			return strings.TrimSpace(string(data)) == "allow", true, nil
		} else if err != nil && os.IsExist(err) {
			return false, true, err
		}

		if data, err := auth.Child("default.perm").Read(); err == nil {
			return strings.TrimSpace(string(data)) == "allow", true, nil
		} else if err != nil && os.IsExist(err) {
			return false, true, err
		}

		if auth.DirExists() && !auth.Child("inherit").Exists() {
			break
		}
	}
	return false, false, nil
}

func getPath(entry Entry) string {
	p := ""
	e := entry
	for e != nil {
		p = e.Name() + "/" + p
		e = e.Parent(false)
		parent := e.Parent(true)
		if e == nil && parent != nil {
			p = ""
			e = parent
		}
	}
	return path.Clean("/" + p)
}
