package server2

import (
	"crypto/sha256"
	"crypto/x509"
	"github.com/mildred/SmartWeb/sparql"
	"net/url"
)

func checkAuth(dataSet *sparql.Client, u *url.URL, method, user string) (bool, error) {
	var parentChain string
	for _, url := range urlParents(u) {
		parentChain = parentChain + sparql.MakeQuery(" %1u", url.String())
	}

	res, err := dataSet.Select(sparql.MakeQuery(`
		PREFIX sw: <tag:mildred.fr,2015-05:SmartWeb#>

		SELECT ?page ?acl ?user ?auth ?act
		WHERE {
			VALUES ?src { %1q }
			?page
				sw:child* ?src .
			?acl
				a        sw:ACL ;
				sw:about ?page ;
				sw:user+ ?user ;
				?auth    ?act .
			VALUES ?user { %3u sw:Anonymous }
			VALUES ?auth { sw:allow sw:deny }
			VALUES ?act { %2s sw:Default }
		}
	`, parentChain, method, user))
	if err != nil {
		return false, err
	}

	var firstPage string = ""
	var defaultAuth, actionAuth, auth string
	for _, binding := range res.Results.Bindings {
		if firstPage != "" && firstPage != binding["page"].Value {
			break
		}
		firstPage = binding["page"].Value
		auth := binding["auth"].Value
		act := binding["act"].Value
		if act == "tag:mildred.fr,2015-05:SmartWeb#Default" {
			if defaultAuth != "" && defaultAuth != auth {
				defaultAuth = "deny"
			} else {
				defaultAuth = auth
			}
		} else {
			if actionAuth != "" && actionAuth != auth {
				actionAuth = "deny"
			} else {
				actionAuth = auth
			}
		}
	}
	if actionAuth != "" {
		auth = actionAuth
	} else {
		auth = defaultAuth
	}
	return auth == "tag:mildred.fr,2015-05:SmartWeb#allow", nil
}

func SHA256Fingerprint(cert x509.Certificate) []byte {
	h := sha256.New()
	h.Write(cert.RawSubjectPublicKeyInfo)
	return h.Sum(nil)
}
