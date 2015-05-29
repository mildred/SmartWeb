SmartWeb2
=========

Smart web server that will understands HTML and handlme the HTTP protocol in a
smart way. Under the hood, RDF is used to store metadata about each URI that can
be queried. Future versions will also extract some metadata from stored HTML
documents and will make it possible to search for them using SPARQL.

The end goal is to have a generic server implementation for websites. web
applications that might require special API from the server are not in scope.
But anything like a blog, an index page for a blog, a list of comments or
talkbacks should be possible.

SmartWeb also store backlinks to URI. When a request is made, the Referer header
is stored in the RDF database. Future versions will try to find a SPARQL/RDF
endpoint from the referer URL (in case it is also a SmartWeb implementation) and
will store more meta information about this backlings. The goal is to expose
this information to the pages and they should make use of it. Thisis the basic
feature that would be used to implement comments, talkbacks, index pages and so
on.

Getting started
---------------

First, build smartweb:

	go build ./cmd/smartweb2

Then, run the Blazegraph database:

	java -server -Xmx4G -jar bigdata-bundled.jar

Go to the web interface, and create a namespace called `smartweb` for Storing
quads. You can optionally enable full text indexing. If you did all that, you
should be able to execute smartweb this way:

	./smartweb2 --noacl --sparql=http://localhost:9999/bigdata/namespace/smartweb/sparql

Installing the page editing application
---------------------------------------

Build the `edit.web` bundle that contains the web application:

    make edit.web

Then, insert this application in smartweb (you can chane the URL you insert the
application in):

    curl -v -X POST \
	  --data-binary @edit.web \
	-H 'Content-Type: application/smartweb-bundle+zip' \
	http://localhost:8000/edit/

This can take some time, at least a minute. Currently, a bug prevents the HTTP
response from being received, this is investigated. You should see in the
smartweb logs a message like this:

	2015/05/29 07:30:36 67.351099ms to download the bundle
	17.26946686s to read the graphs and make the SPARQL query
	2.114954285s to copy the files in the bundle
	33.11796985s to run the SPARQL Update query
	
	2015/05/29 07:30:36 POST Bundle: updated RDF

You can then go to http://localhost:8000/edit/edit.html

Security and privacy for the client
===================================

Security and privacy of the client is important.

  * The Etag header is tagged with a hash function that the client can check
    against the content. This can be used to implement caching that do not leak
    too much information to the server.

  * Ideally, javascript would be disabled on the web, and HTML imports would be
    used to import trusted (from the client point of view) web components that
    enable dynamic processing. This should be done in the client.
	
	This could be implemented by disabling javascript over HTTP/HTTPS and have
	a special URI scheme indexing files by their content hash, with only
	whitelisted components having javascript enabled. NoScript should be able to
	do that.

SmartWeb Details
================

Bundles
-------

The server should allow whole hierarchies to be exported and imported. Export is
not yet implemented, but import can be performed using the HTTP POST method.
HTTP PUT should also be possible (in which case, all sub-resources would be
deleted before recursively).

Bundles are ZIP files that contain a file stored first in the archive, called
`mimetype`, stored with no compression and containing
`application/smartweb-bundle+zip`. The file should also contains a RDF quads
serialization in `graphs.nq` and all the files of the hierarchy indexed by their
content hash. All information about the tree structure of the bundle is stored
in RDF graphs.

Bundles can be looked up and created locally using the utility in
`cmd/swbundle`:

* `swbundle BUNDLE` shows the content of the bundle
* `swbundle BUNDLE DIR` creates the bundle with all files contained in `DIR`
  (without including `DIR` in the hierarchy)

TLS Connections
---------------

The server can accept both HTTP and HTTPS connections on the same port. It uses
heuristics to determine if the TLS tunnel is to be started depending on the
client first message. If there is no certificate, the server will generate one
but it might not be appropriate for all user agents. To generate a self signed
certificate yourself, use the following command line:

    openssl req -x509 -nodes -newkey rsa:2048 -keyout key.pem -out cert.pem -days 365 -subj '/CN=*'

Client certificates are accepted. To generate a self signed client certificate,
use:

    openssl req -x509 -newkey rsa:2048 -keyout key.pem -out cert.pem -days 3650 -nodes
    openssl pkcs12 -export -nodes -in cert.pem -inkey key.pem -out client-cert.p12

Then import the `client-cert.p12` into Firefox

HTTP Methods
------------

The standard HTTP methods implemented for these entries are:

* `GET` return the data contained in the entry, and add headers from the entries
  `/headers/*` are added to the response

* `HEAD` return the headers that the `GET` request would have returned, without
  the body

* `PUT` set the entry to the data contained in the request, and reset the header
  meta-entries. The `Content-Type` header meta-entry is set from the request.

* `POST` accepts a bundle and insert it in at the given point.

* `DELETE` removes an entry with its meta entry and its children. If the request
  path end with `/`, the entry's children will be removed as well.

### Future Ideas ###

* Referer parsing:

    - When a query to a resource has a referer field, and if that referer was not
	  processed within a configurable time frame
	- The referer is contacted with a SPARQL query SELECT ?subj, ?pred WHERE { ?subj ?pred <our-uri> }
	- If the request was successfull, store the results in a graph with the URI of
	  the referrer, containing: ?subj ?pred <our-uri>
	- To validate the backlink, the administrator has to add this link to the default graph of the dataset

* Full text search: `POST` (or `QUERY`, or `SEARCH`) with `Content-Type: application/sparql-query`
  [REC-sparql11-protocol](http://www.w3.org/TR/2013/REC-sparql11-protocol-20130321/)
  using the full text search namespace `http://www.tracker-project.org/ontologies/fts#`.
  Will search in the entry and the sub-entries for full text match.

* `EXPORTARCHIVE` exports a tree in an archive for backup

* `IMPORTARCHIVE` imports a tree in an archive for backup

* `LIST` to list children in a directory

* `REQUEST-LINK` request a link to be made from the resource to another given as
  argument. This could be used to advertise backlinks. Perhaps this is a
  candidate for `POST`.

* `POST` with `Content-Type: text/turtle; charset="utf-8"` to insert backlinks
  in documents. Reply with `202 Accepted` if the request is pending moderation,
  or `204 No Content` if the request was successfull.


Backlinks (ideas only)
----------------------

NOTE: in smartweb2, the `?rdf` suffix is no longer there. The graph and the page
share the same URI.

<blog-post.html> contains {
  <blog-post.html> a            html:Document ;
                   described:in <blog-post.html?rdf>
}

in blog-post.html we have: <link rel="described:in" src="store.rdf"/>

<blog-post.html?rdf> contains {
  <comment.html> a html:Document, text:comment ; talk:about <blog-post.html>
  <comment.html> a html:Document, text:comment ; talk:about <blog-post.html>
  <comment.html> a html:Document, text:comment ; talk:about <blog-post.html>
  <other-store.rdf> claims { <unvalidated-comment.html> talk:about <blog-post.html> }
  <unvalidated-comment2.html> claims { <unvalidated-comment2.html> talk:about <blog-post.html> }
  <referrer.html> seems-to-refers:to <blog-post.html> // unchecked referrer, removed once checked
}

in unvalidated-comment.html we have: <link rel="described:in" src="other-store.rdf"/>
in other-store.n3 we have: <unvalidated-comment.html> talk:about <blog-post.html>

in unvalidated-comment2.html we have: <link rel="talk:about" src="blog-post.html"/>


About Index Pages (ideas)
-------------------------

Have a HTML element that does templating based on RDF:

  <template rdf-source="?rdf&query=<SPARQL QUERY>">
    <iframe src="?href"></iframe>
  </template>

See xul:template reference: http://www-archive.mozilla.org/rdf/doc/xul-template-reference.html
See promising research in: http://referaat.cs.utwente.nl/conference/10/paper/6949/rdf-integration-in-html-5-web-pages.pdf
Or use web components http://webcomponents.org/articles/introduction-to-shadow-dom/

Authentication (incomplete)
---------------------------

Authentication is done via client-side certificates. The server can generate one
for you if you pass `?keygen` on the request URI. ACL can specify rights based
on those certificates, they are not required to be signed by the server private
key.
