ckeditor/get-version.sh: ckeditor/dev/builder/build.sh
	echo "#!/bin/bash" >$@
	echo 'cd $$(dirname "$$0")' >>$@
	echo "exec 3>&1 1>/dev/null" >>$@
	sed -n '/^VERSION=/,/^java/ { /^java/ d; p }' ckeditor/dev/builder/build.sh >>$@
	echo 'echo "VERSION='"'"'$$VERSION'"'"'">&3'   >>$@
	echo 'echo "REVISION='"'"'$$REVISION'"'"'">&3' >>$@
	chmod +x $@

ckeditor/version.sh: ckeditor/get-version.sh
	ckeditor/get-version.sh >$@
	chmod +x $@
.PHONY: ckeditor/version.sh

ckbuilder.%.jar:
	@CKBUILDER_VERSION='$(patsubst ckbuilder.%.jar,%,$@)' ; eval "$$(sed -n '/^CKBUILDER_URL=/ p' ckeditor/dev/builder/build.sh)"; set -x; curl -o $@ -R -z $@ $$CKBUILDER_URL

ckbuilder.jar:
	@eval "$$(sed -n '/^CKBUILDER_VERSION=/ p' ckeditor/dev/builder/build.sh)"; set -x; $(MAKE) ckbuilder.$$CKBUILDER_VERSION.jar; ln -sf ckbuilder.$$CKBUILDER_VERSION.jar ckbuilder.jar

build-ckeditor: ckeditor/version.sh ckbuilder.jar ckeditor/dev/builder/build-config.js
	#@for p in $(PLUGINS); do (set -x; ln -sf ../../$$p ckeditor/plugins/$$p); done
	@. ckeditor/version.sh; set -x; java -jar ckbuilder.jar --build ckeditor ckeditor-build --version="$$VERSION" --revision="$$REVISION" --overwrite --build-config ckeditor/dev/builder/build-config.js --no-zip --no-tar

edit.appcache:
	echo "CACHE MANIFEST" >$@
	echo "edit.html" >>$@
	(cd ckeditor-build; find ckeditor) >>$@
	find saveurl -print0 | xargs -0 -n 1 printf 'ckeditor/plugins/%s\n' >>$@
.PHONY: edit.appcache

saveurl/js/extensions.min.js: lz-string/libs/lz-string.min.js
	cat $+ >$@

install-ckeditor: edit.appcache saveurl/js/extensions.min.js
	mkdir -p web/localhost:8000.host.dir
	cp edit.html web/localhost:8000.host.dir/edit.html.data
	cp edit.appcache web/localhost:8000.host.dir/edit.appcache.data
	mkdir -p web/localhost:8000.host.dir/edit.appcache.metadir/headers.dir
	echo "text/cache-manifest" > web/localhost:8000.host.dir/edit.appcache.metadir/headers.dir/Content-Type.data
	./smart-copy.sh ckeditor-build/ckeditor web/localhost:8000.host.dir/ckeditor.dir
	./smart-copy.sh saveurl web/localhost:8000.host.dir/ckeditor.dir/plugins.dir/saveurl.dir
