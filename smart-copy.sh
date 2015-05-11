#!/bin/bash

src="$1"
dst="$2"

mkdir -p "$dst"

for f in "$src"/*; do
  bf="$(basename "$f")"
  if [[ -d "$f" ]]; then
    (set -x; mkdir -p "$dst/$bf.dir")
    "$0" "$f" "$dst/$bf.dir"
  else
    (set -x; cp "$f" "$dst/$bf.data")
  fi
done
