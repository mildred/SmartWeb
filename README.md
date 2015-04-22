SmartWeb
========

Smart web server that understands HTML and extra HTTP methods. The content model
is composed of entries. Each entry can have :

* a data, series of bytes stored in a file
* a meta entry associated with the entry
* a directory, list of children entries

This is very powerful, especially considering that the meta entries can
themselves have meta entries of their own, and directories and non-directories
entry are separate.

Entries URI
-----------

Each entry can be accessed using a unique request URI. It is composed as
follows:

* `/` is the root entry
* `/entry` is a file `entry` stored under the root entry. It cannot contains children
* `/entry/` is a directory entry `entry/` stored under the root entry. It contains both data and optional children
* `<URI>?meta=/` is the root meta entry relative to `<URI>`
* `<URI>?meta=/headers/Content-Type` is a file meta entry relative to `<URI>`
* `<URI>?meta=/a&meta=/b` is a meta entry `/b` assigned to the meta entry `/a` itself assigned to `<URI>`

Disk format
-----------

Storing these entries on disk is done using normal files and directories. Each
filesystem entry has a suffix indicating its type. A single SmartWeb entry can
be composed of more than one file on disk, but all those filesystem entries
share the exact same prefix path.

If a SmartWeb entry is stored on disk using the path prefix `~/web/` (this will
generally be the root entry):

* the data part of that entry will be stored at `~/web/data`
* the meta entry will be stored at prefix `~/web/meta` (the metadata will be at
  `~/web/metadata` and the child entries will be at
  `~/web/metadir/<child-name>.`)
* the child entries will be stored with prefix `~/web/<child-name>.` (because
  `~/web/` is already a directory)

Now, an entry `e` stored under the above entry will have:

* its data part in `~/web/e.data`
* its meta part under `~/web/e.meta` (the metadata in `~/web/e.metadata` and
  meta children in `~/web/e.metadir/<child-name>.`)
* its children borrowed from entry `e/` (see below)

Now, an entry `e/` stored under the first described entry will have:

* its data part in `~/web/e.dir/data`
* its meta part under `~/web/e.dir/meta` (the metadata in `~/web/e.dir/metadata`
  and meta children in `~/web/e.dir/metadir/<child-name>.`)
* its children stored under prefix `~/web/e.dir/<child-name>.`

HTTP Methods
------------

The standard HTTP methods implemented for these entries are:

* `GET` return the data contained in the entry, and add headers from the entries
  `/headers/*` are added to the response

* `HEAD` return the headers that the `GET` request would have returned, without
  the body

* `PUT` set the entry to the data contained in the request, and reset the header
  meta-entries. The `Content-Type` header meta-entry is set from the request.

* `DELETE` removes an entry with its meta entry and its children. If the request
  path end with `/`, the entry's children will be removed as well.

Most notably, the HTTP `POST` request is not implemented because it has no clear
semantic.

### Future Ideas ###

* `SEARCH` method to perform a full text search. It will search the child
  entries for the required pattern. If there is no clear semantic given for the
  FTS syntax, provide in a header the link to the syntax definition (accessible
  via `HEAD`) so a script can tell if it knows the syntax, and the human can
  understand it.

* `EXPORTARCHIVE` exports a tree in an archive for backup

* `IMPORTARCHIVE` imports a tree in an archive for backup

* `LIST` to list children in a directory

* `REQUEST-LINK` request a link to be made from the resource to another given as
  argument. This could be used to advertise backlinks. Perhaps this is a
  candidate for `POST`.


Authentication
--------------

Only Digest authentication is implemented for the moment. Authentication is
configured using meta entries, and is inherited from parent directories.

Multiple authentication realms can be defined. A realm is a group of users like
*Normal Users*, *Editors* or *Administrators*. When logging-in, the user agent
is responsible to choose the correct realm. When configuring access policies,
different realms can have different permissions.

The following entries can be used to configure the authentication and
permissions:

* `?meta=/auth/<realm>/`: Defines a realm with identifier `<realm>`. Only if
  `<realm>` is not `anonymous`.
* `?meta=/auth/<realm>/realm`: The realm name provided to the user agent
* `?meta=/auth/<realm>/Digest.users/<username>`: The password for `<username>`
  in `<realm>`
* `?meta=/auth/<realm>/*.perm`: Permissions for users of the specified realm
* `?meta=/auth/<realm>/inherit`: If present, additional settings for this realm
  will be searched in parent directories.
* `?meta=/auth/anonymous/*.perm`: Permissions for anonymous users
* `?meta=/auth/inherit`: If present, additional realms will be searched in
  parent directories. If `?meta=/auth/` does not exists, this is implicit.

The permissions files `*.perm` are looked-up depending on the HTTP method. When
performing a `DELETE` request on `/dir/file`, the following files will be
searched:

* `/dir/file?meta=/auth/<realm>/DELETE.perm`
* `/dir/file?meta=/auth/<realm>/default.perm`
* if `/dir/file?meta=/auth/<realm>/inherit` does not exist, search is stopped there
* `/dir/?meta=/auth/<realm>/DELETE.perm`
* `/dir/?meta=/auth/<realm>/default.perm`
* if `/dir/?meta=/auth/<realm>/inherit` does not exist, search is stopped there
* `/?meta=/auth/<realm>/DELETE.perm`
* `/?meta=/auth/<realm>/default.perm`

The first entry found will be read. If it contains the string `"allow"`
(5 bytes, no carriage return allowed) the access will be granted. Else, the file
is supposed to contain `"deny"` (4 bytes).

If no perm entry is found, access will be denied.

### Future Work ###

Work should be performed in the current browsers to allow the following
functions:

* Allow to choose different realms if the server provides many
* Allow log-out
* Clearly show in the browser chrome if the user is logged-in
* Make sure the domain attribute of the digest authentication is honored

The server could be made to play well with user agents by only showing a single
realm, and check credentials against all.

Ed25519 authentication should be implemented both server-side and client-side.
