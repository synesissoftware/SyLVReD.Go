#! /bin/bash

ScriptPath=$0
Dir=$(cd $(dirname "$ScriptPath"); pwd)
ProjectNameFile="$Dir/.sis/project_name.txt"
ProjectName=$(tr -d '[:space:]' < "$ProjectNameFile")


# ##########################################################
# command-line handling

while [[ $# -gt 0 ]]; do

  case $1 in
    --help)

      [ -f "$Dir/.sis/script_info_lines.txt" ] && cat "$Dir/.sis/script_info_lines.txt"
      cat << EOF
Builds the ${ProjectName} library, command-line tools, and (if present) examples

$ScriptPath [ ... flags/options ... ]

Flags/options:

    standard flags:

    --help
        displays this help and terminates

EOF

      exit 0
      ;;
    *)

      >&2 echo "$ScriptPath: unrecognised argument '$1'; use --help for usage"

      exit 1
      ;;
  esac

  shift
done


# ##########################################################
# main()

mkdir -p "$Dir/scratch/build-artefacts/"


# library
echo "building ${ProjectName} library ..."
go build -v "$Dir"


# commands (cmd/<name>/main.go)
if [ -d "$Dir/cmd" ]; then

  echo "building ${ProjectName} commands ..."

  for cmd_dir in "$Dir"/cmd/*/; do

    [ -d "$cmd_dir" ] || continue

    if [ -f "${cmd_dir}main.go" ]; then

      cmd_name=$(basename "$cmd_dir")
      echo "building command ${cmd_name} ..."
      go build -v -o "$Dir/scratch/build-artefacts/${cmd_name}" "$cmd_dir"
    fi
  done
fi


# examples
if [ -d "$Dir/examples" ]; then

  echo "building ${ProjectName} examples ..."

  # subdirectory examples (examples/<name>/main.go)
  for example_dir in "$Dir"/examples/*/; do

    [ -d "$example_dir" ] || continue

    if [ -f "${example_dir}main.go" ]; then

      example_name=$(basename "$example_dir")
      echo "building example ${example_name} ..."
      go build -v -o "$Dir/scratch/build-artefacts/${example_name}" "$example_dir"
    fi
  done

  # loose single-file examples (examples/*.go)
  for example_file in "$Dir"/examples/*.go; do

    [ -f "$example_file" ] || continue

    example_name=$(basename "$example_file" .go)
    echo "building example ${example_name} ..."
    go build -v -o "$Dir/scratch/build-artefacts/${example_name}" "$example_file"
  done
fi


# ############################## end of file ############################# #
