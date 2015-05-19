#!/bin/bash

while true; do
    case "$1" in
        action=*)   action="${1#*=}" ; shift ;;
        user=*)     user="${1#*=}"   ; shift ;;
        page=*)     page="${1#*=}"   ; shift ;;
        *=*)        eval "$1" ; shift ;;
        --)         shift ; break ;;
        *)          break ;;
    esac
done

( set -x
sparql-update 'http://127.0.0.1:8080/openrdf-sesame/repositories/smartweb/statements' "
PREFIX sw: <tag:mildred.fr,2015-05:SmartWeb#>

DELETE {
	?oldacl ?pred ?obj .
}
WHERE {
	?oldacl a sw:ACL ;
		sw:about $page ;
		sw:user $user ;
		?pred ?obj .
};

INSERT DATA {
	[]	a sw:ACL ;
		sw:about $page ;
		sw:user  $user ;
		sw:allow $action .
}
"
)
sparql-query 'http://127.0.0.1:8080/openrdf-sesame/repositories/smartweb' "   
PREFIX sw: <tag:mildred.fr,2015-05:SmartWeb#>

SELECT ?acl ?page ?user
WHERE {
    ?acl a sw:ACL ; sw:about ?page ; sw:user ?user .
}
"