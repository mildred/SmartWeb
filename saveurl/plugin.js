CKEDITOR.plugins.add( 'saveurl', {
    icons: 'openurl,saveurl',
    requires: ['ajax'],
    hidpi: true,
    init: function( editor ) {

        /***********************************
        ** Warn before closing the window **
        ***********************************/

        window.addEventListener("beforeunload", function (e) {
            if(!editor.checkDirty()) {
                console.log("Editor is clean.");
                return;
            }
            var confirmationMessage = "If you leave this page, you'll loose unsaved changes.";
            console.log(confirmationMessage);
            (e || window.event).returnValue = confirmationMessage;     //Gecko + IE
            return confirmationMessage;                                //Gecko + Webkit, Safari, Chrome etc.
        });

        /**************
        ** Auto Save **
        **************/

        var autoSaveDelay = 1000;
        var autoSaveAlready = false;
        var autoSaveEnabled = true;
        var autoSaveKeyPrefix = editor.config.autosave_SaveKey != null ? editor.config.autosave_SaveKey : 'autosave_' + window.location + "_" + editor.id;

        function autoSaveKey(){
          return autoSaveKeyPrefix + editor.documentURL;
        }

        function autoSave() {
            console.log("autoSave");
            if(!autoSaveEnabled) return {ok: false, msg:"auto-save disabled"};
            if(!autoSaveAlready && localStorage.getItem(autoSaveKey())) {
                return {ok: false, msg: "Not Saved: another editor is open"};
            }
            try {
                var compressedJSON = LZString.compressToUTF16(JSON.stringify({ data: editor.getSnapshot(), saveTime: new Date(), docURL: editor.documentURL }));
                autoSaveAlready = true;
                localStorage.setItem(autoSaveKey(), compressedJSON);
                return {ok: true};
            } catch(e) {
                return {ok: false, msg: (""+e)};
            }
        }

        var dirty = false;
        function autoSaveReset() {
            localStorage.removeItem(autoSaveKey());
            editor.resetDirty();
            dirty = false;
            console.log("reset dirty");
        }

        function autoSaveRestore() {
            var data = localStorage.getItem(autoSaveKey());
            if(!data) return autoSaveReset();

            data = JSON.parse(LZString.decompressFromUTF16(data));
            if(confirm("You have an unsaved document. Open it ?\n\nURL: " + data.docURL + "\nAutomatic save time: " + data.saveTime + "\n\nIf you say no, the document backup will be deleted")) {
                autoSaveAlready = true;
                editor.loadSnapshot(data.data);
                return
            } else {
                autoSaveReset();
            }
        }

        editor.on('change', function (ev) {
            console.log("change:");
            if(dirty) return;
            if(!editor.checkDirty()) return;
            console.log("change:dirty");
            dirty = true;

            showStatus("Not Saved")

            setTimeout(function(){
                console.log("change:autoSave? dirty="+dirty);
                if(!dirty) return;

                var saved = autoSave();
                if(!saved.ok) {
                    showStatus(saved.msg);
                    return;
                }

                dirty = false;
                showStatus("Saved in memory");
            }, autoSaveDelay);
        });

        /*******************
        ** Status Message **
        *******************/

        editor.on('uiSpace', function (event) {
            if (event.data.space != 'bottom') return;
            event.data.html += '<div class="autoSaveMessage" unselectable="on"><div unselectable="on" class="hidden" id="cke_saveurlMessage_' + editor.name + '"></div></div>';
        }, editor, null, 100);

        function showStatus(msg, timeout) {
            var autoSaveMessage = document.getElementById('cke_saveurlMessage_' + editor.name);
            if (autoSaveMessage) {
                autoSaveMessage.className = "show";
                autoSaveMessage.textContent = msg;
                if(timeout) {
                    setTimeout(function() {
                        autoSaveMessage.className = "hidden";
                    }, timeout);
                }
            }

        }

        function hideStatus() {
            var autoSaveMessage = document.getElementById('cke_saveurlMessage_' + editor.name);
            if (autoSaveMessage) {
                autoSaveMessage.className = "hidden";
                autoSaveMessage.textContent = "";
            }
        }

        /*****************
        ** HTTP Request **
        *****************/

        // opts.url
        // opts.content_type
        // opts.data
        // cb(ok, location, xhr)
        function httpPutFile(opts, cb) {
            var url = opts.url;
            var xhr = new XMLHttpRequest();
            xhr.open('PUT', url, true);
            xhr.onreadystatechange = httpCallback;
            xhr.setRequestHeader('Content-Type', opts.content_type);
            xhr.send(opts.data);

            function httpCallback() {
                if ( xhr.readyState != 4 ) return;
                var ok = (xhr.status >= 200 && xhr.status < 300) ||
                    xhr.status == 304 || xhr.status === 0 || xhr.status == 1223;

                var location = xhr.getResponseHeader("Location");
                if(!location) {
                    location = url;
                } else if(location[0] == '/') {
                    location = url.replace(/^(.*[^\/])?\/[^\/].*$/, "$1" + location);
                }

                cb(ok, location, xhr);
            }
        }

        function httpPutEditor(url, cb) {
            httpPutFile({
                url:          url,
                data:         editor.getData(),
                content_type: 'text/html; charset=utf-8'
            }, function(ok, location, xhr){
                if(ok) {
                    editor.documentURL = location;
                    if(location != url) {
                        alert("Document saved to a new location:\n" + location);
                    }
                    autoSaveReset();
                    showStatus("Saved at " + location);
                    editor.fire("documentURLChange", location);
                } else {
                    alert("Could not save document:\n" + xhr.responseText);
                }
                if(cb) cb();
            });
        }

        /******************
        ** Open and Save **
        ******************/

        function openUrl(url, cb){
          CKEDITOR.ajax.load(url, function(data){
              editor.setData(data, {callback: function(){
                editor.documentURL = url;
                if(cb) cb();
                autoSaveRestore();
                editor.fire("documentURLChange", url);
              }});
          });
        }

        /****************
        ** Open Dialog **
        ****************/

	CKEDITOR.dialog.add( 'openUrl', function( editor ) {
            var dialogDefinition = {
                title: 'Open URL',
                minWidth: 390,
                minHeight: 50,
                contents: [
                    {
                        type: 'hbox',
                        id: 'box',
                        elements: [
                            {
                                type: 'text',
                                id: 'urlId'
                            },
                            {
                                type: 'html',
                                id:   'html',
                                html: ''
                            }
                        ]
                    }
                ],
                buttons: [
                    CKEDITOR.dialog.okButton(editor, {label: "Open"}),
                    CKEDITOR.dialog.cancelButton(editor, {label: "Cancel"})
                ],
                onShow: function(){
                    document.getElementById(this.getContentElement('box', 'html').domId).innerHTML = "";                    this.getContentElement('box', 'urlId').enable();
                    this.getContentElement('box', 'urlId').setValue(editor.documentURL, true);
                },
                onCancel: function() {},
                onOk: function(ev) {
                    // "this" is now a CKEDITOR.dialog object.
                    // Accessing dialog elements:
                    var url = this.getContentElement('box', 'urlId').getValue();
                    document.getElementById(this.getContentElement('box', 'html').domId).innerHTML = "Loading...";
                    this.getContentElement('box', 'urlId').disable();
                    ev.data.hide = false;
                    var dlg = this;
                    openUrl(url, function(){
                      dlg.hide();
                    })
                }
            };

            return dialogDefinition;

        } );

        /****************
        ** Save Dialog **
        ****************/

	CKEDITOR.dialog.add( 'saveUrlAs', function( editor ) {
            return {
                title: 'Save URL',
                minWidth: 390,
                minHeight: 50,
                contents: [
                    {
                        type: 'vbox',
                        id: 'box',
                        elements: [
                            {
                                type: 'text',
                                id: 'urlId'
                            },
                            {
                                type: 'html',
                                id:   'html',
                                html: ''
                            }
                        ]
                    }
                ],
                buttons: [
                    CKEDITOR.dialog.okButton(editor, {label: "Save"}),
                    CKEDITOR.dialog.cancelButton(editor, {label: "Cancel"})
                ],
                onShow: function(){
                    this.getContentElement('box', 'urlId').setValue(editor.documentURL, true);
                    this.getContentElement('box', 'urlId').enable();
                    document.getElementById(this.getContentElement('box', 'html').domId).innerHTML = "";
                },
                onCancel: function() {},
                onOk: function(ev) {
                    // "this" is now a CKEDITOR.dialog object.
                    // Accessing dialog elements:
                    var url = this.getContentElement('box', 'urlId').getValue();
                    var dlg = this;
                    this.getContentElement('box', 'urlId').disable();
                    document.getElementById(this.getContentElement('box', 'html').domId).innerHTML = "Saving...";

                    if(!url) {
                        alert("Could not save document:\nURL not specified");
                        return;
                    }

                    ev.data.hide = false;
                    httpPutEditor(url, function(){
                        dlg.hide();
                    });
                }
            };
        } );

	CKEDITOR.dialog.add( 'saveUrl', function( editor ) {
            return {
                title: 'Save URL',
                minWidth: 390,
                minHeight: 50,
                contents: [
                    {
                        type: 'vbox',
                        id: 'box',
                        elements: [
                            {
                                type: 'html',
                                id:   'html',
                                html: ''
                            }
                        ]
                    }
                ],
                buttons: [],
                onShow: function(){
                    var html = document.getElementById(this.getContentElement('box', 'html').domId);
                    var dlg = this;
                    html.innerHTML = "Saving document to " + editor.documentURL + "...";
                    httpPutEditor(editor.documentURL, function(){
                        html.innerHTML = "Saved to " + editor.documentURL;
                        setTimeout(function(){
                          dlg.hide()
                        }, 500);
                    });
                },
            };
        } );

        /*************
        ** Commands **
        *************/

        editor.addCommand( 'openUrl', {
          exec: function(editor, data) {
            if(data && data.url && data.url === editor.documentURL) return true;

            if(editor.checkDirty()) {
              var res = autoSave();
              if (!res.ok) {
                if(res.msg) alert("Could not automatically save changes: " + res.msg);
                else        alert("Could not automatically save changes.");
                return false;
              }
              if (!confirm("You have unsaved changes, continue ?")) return false;
            }
            if(data && data.url) {
              editor.setData("Opening " + data.url + "...")
              openUrl(data.url);
            } else {
              editor.openDialog( 'openUrl' );
            }
            return true;
          },
          canUndo: false,
          editorFocus: 1
        });
        editor.addCommand( 'saveUrl', new CKEDITOR.dialogCommand( 'saveUrl' ) );
        editor.addCommand( 'saveUrlAs', new CKEDITOR.dialogCommand( 'saveUrlAs' ) );

        editor.ui.addButton( 'SaveUrl', {
            label: 'Save URL',
            command: 'saveUrl',
            toolbar: 'document',
            icon: 'saveurl'
        });
        editor.ui.addButton( 'SaveUrlAs', {
            label: 'Save URL As...',
            command: 'saveUrlAs',
            toolbar: 'document',
            icon: 'saveurl'
        });

        editor.ui.addButton( 'OpenUrl', {
            label: 'Open URL',
            command: 'openUrl',
            toolbar: 'document',
            icon: 'openurl'
        });

        /*******************
        ** Initialization **
        *******************/

        editor.on('instanceReady', function (ev) {
            CKEDITOR.scriptLoader.load(CKEDITOR.getUrl(CKEDITOR.plugins.getPath('saveurl') + 'js/extensions.min.js'), function(){

              editor.window.getFrame().$.contentWindow.addEventListener("keydown", function(e){
                  if(e.key == 's' && e.ctrlKey) {
                      editor.execCommand("saveUrlAs");
                      e.preventDefault();
                  }
              });

              if(editor.config.autosave_open){
                editor.setData("Loading...")
                openUrl(editor.config.autosave_open)
              } else {
                autoSaveRestore();
              }

            });
        });

    }
});
