package server2

import (
	"net/url"
	"crypto/sha256"
	"crypto/x509"
	"github.com/mildred/SmartWeb/sparql"
)

var query = `
PREFIX sw: <tag:mildred.fr,2015-05:SmartWeb#>

SELECT ?acl ?auth
WHERE {
	?acl
		a sw:ACL ;
		sw:about/(^sw:parent)* %page ;
		sw:user+ ?user
		?auth ?method .
	VALUES ?user { %user sw:Anonymous }
	FILTER ( ?user == %user || ?user == sw:Anonymous )
	FILTER ( ?auth == sw:allow || ?auth == sw:deny )
	FILTER ( ?method == %method || ?method == sw:Default )
}


SELECT DISTINCT ?acl ?auth ?defAuth
WHERE {
	?acl
		?auth %method ;
		?defAuth sw:Default ;
		sw:user+ ?user .
	VALUES ?user { %user sw:Anonymous }
	VALUES ?auth { sw:allow sw:deny }
	VALUES ?defAauth { sw:allow sw:deny }
	{
		SELECT ?page ?acl
		WHERE {
			%page sw:parent ?page .
			?acl a sw:ACL ; sw:about ?page .
		}
		LIMIT 1
	}
}

	?acl
		a sw:ACL
		sw:about ?page
	
	%page sw:parent* ?page


DELETE {
	?oldacl ?pred ?obj .
}
INSERT {
	[	a sw:ACL ;
		sw:about %page ;
		sw:user  %user ;
		sw:allow %action .
	]
}
WHERE {
	?oldacl a sw:ACL ;
		sw:about %page ;
		sw:user %user .
}
`

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
			?src
				sw:parent* ?page .
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
			break;
		}
		firstPage = binding["page"].Value
		auth := binding["auth"].Value
		act  := binding["act"].Value
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