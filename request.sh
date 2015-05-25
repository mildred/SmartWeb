#!/bin/sh
( set -x
  sparql-update 'http://127.0.0.1:8080/repositories/smartweb/statements' "$(cat <<EOF
DROP SILENT GRAPH <http://localhost:8000/toto/graphs/_rdf.ttl> ;
INSERT DATA {
 GRAPH <http://localhost:8000/toto/graphs/_rdf.ttl> {
   <graphs/_rdf.ttl> <tag:mildred.fr,2015-05:SmartWeb#hash> <sha1:5c335f24ec1d63faacb3ad6a7edee1add2fbede6> } } ;
   DROP SILENT GRAPH <http://localhost:8000/toto/graphs/page.html_rdf.ttl> ;
   INSERT DATA {
    GRAPH <http://localhost:8000/toto/graphs/page.html_rdf.ttl> {
      <graphs/page.html_rdf.ttl> <tag:mildred.fr,2015-05:SmartWeb#hash> <sha1:1f9a836ee03f83e333129035b4c248c577e3a5f7> } } ;
      DROP SILENT GRAPH <http://localhost:8000/toto/graphs.sparql> ;
      INSERT DATA {
       GRAPH <http://localhost:8000/toto/graphs.sparql> {
         <graphs.sparql> <tag:mildred.fr,2015-05:SmartWeb#hash> <sha1:e24515d561c693f73bcc0facf7072a6771f7e624> } } ;
	 DROP SILENT GRAPH <http://localhost:8000/toto/hashs/sha1:0> ;
	 INSERT DATA {
	  GRAPH <http://localhost:8000/toto/hashs/sha1:0> {
	    <hashs/sha1:0> <tag:mildred.fr,2015-05:SmartWeb#hash> <sha1:da39a3ee5e6b4b0d3255bfef95601890afd80709> } } ;
	    DROP SILENT GRAPH <http://localhost:8000/toto/.graphs.nq.swp> ;
	    INSERT DATA {
	     GRAPH <http://localhost:8000/toto/.graphs.nq.swp> {
	       <.graphs.nq.swp> <tag:mildred.fr,2015-05:SmartWeb#hash> <sha1:9a33b2734326c83052c9d8983864220ed9f3bb45> } } ;
	       DROP SILENT GRAPH <http://localhost:8000/toto/graphs.nq> ;
	       INSERT DATA {
	        GRAPH <http://localhost:8000/toto/graphs.nq> {
		  <graphs.nq> <tag:mildred.fr,2015-05:SmartWeb#hash> <sha1:7c00581adefdd057fb95dbe7d2ee6404d8fcd205> } }
EOF
)"
) 2>&1 | less -S
