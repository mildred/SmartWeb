#!/bin/bash

update_url="http://localhost:8080/repositories/SYSTEM/statements"
repo_name=smartweb
delete_repo=n
run_request=y

echo -n "OpenRDF Sesame update URL? [$update_url] "
read ans
[[ -n "$ans" ]] && update_url="$ans"

echo -n "Repository name? [$repo_name] "
read ans
[[ -n "$ans" ]] && repo_name="$ans"

repo_url="${update_url/SYSTEM/$repo_name}"
repo_url="${repo_url%/statements}"

echo -n "Delete repository $repo_url? [y/N] "
read ans
[[ -n "$ans" ]] && delete_repo="$ans"
delete_repo="$(tr A-Z a-z <<<"$delete_repo")"

if [[ $delete_repo = y ]]; then
    ( set -x
      curl -v -X DELETE "$repo_url"
    )
fi

request="$(cat <<EOF
PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
PREFIX rep: <http://www.openrdf.org/config/repository#>
PREFIX sr: <http://www.openrdf.org/config/repository/sail#>
PREFIX sail: <http://www.openrdf.org/config/sail#>
PREFIX sys: <http://www.openrdf.org/config/repository#>
PREFIX ns: <http://www.openrdf.org/config/sail/native#>

DELETE {
  GRAPH ?ctx { ?s ?p ?o }
  ?ctx a sys:RepositoryContext .
  ?repo ?pred ?obj .
}
WHERE {
  GRAPH ?ctx {
    ?s ?p ?o .
    ?any rep:repositoryID "$repo_name" .
  }
  ?ctx a sys:RepositoryContext .
  ?repo rep:repositoryID "$repo_name" ; ?pred ?obj .
};

INSERT DATA {
  _:node001 a sys:RepositoryContext .
  GRAPH _:node001 {
    [] a rep:Repository ;
      rep:repositoryID "$repo_name" ;
      rdfs:label "SmartWeb Store" ;
      rep:repositoryImpl [
        rep:repositoryType "openrdf:SailRepository" ;
        sr:sailImpl [
          sail:sailType "openrdf:NativeStore" ;
          ns:tripleIndexes "spoc,posc"
        ]
      ].
  }
}
EOF
)"

echo "SPARQL Request:"
echo
echo "$request"
echo

echo -n "Run request? [Y/n] "
read ans
[[ -n "$ans" ]] && run_request="$ans"
run_request="$(tr A-Z a-z <<<"$run_request")"

if [[ $run_request != n ]]; then
  ( set -x
    sparql-update -v "$update_url" "$request"
  )
fi

query_url="${update_url%/statements}"
( set -x
  sparql-query "$query_url" "
PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
PREFIX rep: <http://www.openrdf.org/config/repository#>
PREFIX sys: <http://www.openrdf.org/config/repository#>

SELECT ?graph ?repo ?name ?label
WHERE {
  ?graph a sys:RepositoryContext .
  GRAPH ?graph {
    ?repo a rep:Repository ;
      rep:repositoryID ?name ;
      rdfs:label ?label .
  }
}
"
)

( exec 2>&1
  set -x
  sparql-query "$query_url" "SELECT ?g ?s ?p ?o WHERE { ?s ?p ?o OPTIONAL { GRAPH ?g { ?s ?p ?o } } }"
) | less -S