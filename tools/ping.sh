#!/bin/bash

# check parameters
if [ $# -ne 2 ]; then
    echo "usage: $0 <address> <count>"
    exit 1
fi

HOST=$1
COUNT=$2

# count should be a number
if ! [[ "$COUNT" =~ ^[0-9]+$ ]]; then
    echo "error: count must be a number"
    exit 1
fi

# execute ping
ping -c "$COUNT" "$HOST"