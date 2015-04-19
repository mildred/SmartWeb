SmartWeb
========

Smart web server that understands HTML and extra HTTP methods.

HTTP methods
============

PUT /resource
-------------

Create /resource for future availability using GET.

PROPGET /resource
-----------------

Get properties in JSON format for the resource.

PROPSET /resource
-----------------

Set properties in JSON format for the resource. The request must give a hash of
the previous properties document. If the properties were not changed between the
PROPGET and PROPSET, the document will be updated.

Properties
==========

Permissions
-----------

The perms object contains the permissions for the resource, or if the resource
is a  directiry, the sub resources. It is a key-value store. The key represents
the user identifier, the value the different permissions. For example:

    "Permissions": {
		"*": {
			"GET": true,
			"*": false
		},
		"user:guest:guest": {
			"GET": true,
			"PUT": true,
			"*": false
		},
		"ed25519:<base64-encoded-public-key>": {
			"*": true
		}
	}

Headers
-------

The headers set using PUT and returned on GET requests. Example:

    "Headers": {
		"Content-Type": "text/html"
	}
