#!/bin/sh
set -x
sparql-query 'http://127.0.0.1:8080/repositories/smartweb' "$(cat <<EOF
PREFIX sw: <tag:mildred.fr,2015-05:SmartWeb#>
SELECT *
WHERE {
	GRAPH ?g {
		?s ?p ?o 
	}
	<http://127.0.0.1:8000/edit/ckeditor/skins/moono/> sw:child+ ?page.
	FILTER (str(?g) = concat(str(?page), "?rdf")) 
}
EOF
)" | less -S
