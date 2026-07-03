#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  ./extract.sh [-d OUTPUT_DIR] ARCHIVE_FILE

Examples:
  ./extract.sh app.tar.gz
  ./extract.sh -d /tmp/output app.zip
EOF
}

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Error: required command not found: $1" >&2
    exit 1
  fi
}

output_dir="."

while getopts ":d:h" opt; do
  case "$opt" in
    d)
      output_dir="$OPTARG"
      ;;
    h)
      usage
      exit 0
      ;;
    :)
      echo "Error: option -$OPTARG requires an argument" >&2
      usage >&2
      exit 1
      ;;
    \?)
      echo "Error: unknown option -$OPTARG" >&2
      usage >&2
      exit 1
      ;;
  esac
done
shift $((OPTIND - 1))

if [ "$#" -ne 1 ]; then
  usage >&2
  exit 1
fi

archive=$1

if [ ! -f "$archive" ]; then
  echo "Error: file does not exist: $archive" >&2
  exit 1
fi

mkdir -p "$output_dir"

case "$archive" in
  *.tar.gz|*.tgz)
    need_cmd tar
    tar -xzf "$archive" -C "$output_dir"
    ;;
  *.tar.bz2|*.tbz2)
    need_cmd tar
    tar -xjf "$archive" -C "$output_dir"
    ;;
  *.tar.xz|*.txz)
    need_cmd tar
    tar -xJf "$archive" -C "$output_dir"
    ;;
  *.tar)
    need_cmd tar
    tar -xf "$archive" -C "$output_dir"
    ;;
  *.zip)
    need_cmd unzip
    unzip "$archive" -d "$output_dir"
    ;;
  *.gz)
    need_cmd gzip
    gzip -dk "$archive"
    if [ "$output_dir" != "." ]; then
      mv "${archive%.gz}" "$output_dir/"
    fi
    ;;
  *.bz2)
    need_cmd bzip2
    bzip2 -dk "$archive"
    if [ "$output_dir" != "." ]; then
      mv "${archive%.bz2}" "$output_dir/"
    fi
    ;;
  *.xz)
    need_cmd xz
    xz -dk "$archive"
    if [ "$output_dir" != "." ]; then
      mv "${archive%.xz}" "$output_dir/"
    fi
    ;;
  *.7z)
    need_cmd 7z
    7z x "$archive" "-o$output_dir"
    ;;
  *.rar)
    need_cmd unrar
    unrar x "$archive" "$output_dir/"
    ;;
  *)
    echo "Error: unsupported archive format: $archive" >&2
    exit 1
    ;;
esac

echo "Extracted to: $output_dir"
