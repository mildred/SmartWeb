#!/bin/bash

verb=

smartput(){
    local type=""
    local recursive=false
    local src
    local url

    while true; do
        case $1 in
            -r)   recursive=true ; shift ;;
            -t)   type="$2"; shift 2 ;;
            -t=*) type="${1#*=}"; shift ;;
            *=*)  local "${1%%=*}"; eval "$1"; shift ;;
            --)   shift; break ;;
            *)    break ;;
        esac
    done

    if [[ $# -gt 0 ]]; then
        src="$1"
    fi
    if [[ $# -gt 1 ]]; then
        url="$2"
    fi
    if [[ -z "$type" ]]; then
        type="$(file -i "$src" | cut -d: -f2 | cut -c2-)"
        case "$type" in
            text/plain*)
                case "$src" in
                    *.js)  type=${type/plain/javascript} ;;
                    *.css) type=${type/plain/css} ;;
                    *)  ;;
                esac
                ;;
            *)  ;;
        esac
    fi

    if [[ -d "$src" ]]; then
        if [[ -z "$verb" ]]; then
            verb=" "
        fi
        if ! $recursive; then
            echo "Please specify -r flag" >&2
            exit 1
        fi
        local f
        for f in "$src"/*; do
            smartput -r "src=$f" "url=${url%/}/${f##*/}"
        done
    else
        if [[ -z "$verb" ]]; then
            verb=-v
        fi
        ( set -x
          curl $verb -X PUT --data-binary "@$src" -H "Content-Type: $type" "$url"
        )
    fi
}

smartput "$@"