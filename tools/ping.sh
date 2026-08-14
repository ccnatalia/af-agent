#!/bin/bash

# check parameters
if [ $# -lt 2 ] || [ $# -gt 3 ]; then
    echo "usage: $0 <address> <count> [output_file]"
    exit 1
fi

HOST=$1
COUNT=$2
OUTPUT_FILE=$3

# count should be a number
if ! [[ "$COUNT" =~ ^[0-9]+$ ]]; then
    echo "error: count must be a number"
    exit 1
fi

# execute ping
if [ -n "$OUTPUT_FILE" ]; then
    ping -c "$COUNT" "$HOST" > "$OUTPUT_FILE"
else
    ping -c "$COUNT" "$HOST"
fi

exit 0
