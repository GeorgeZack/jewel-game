#!/bin/bash
if [[ -e "/usr/bin/fd" && -e "/usr/bin/entr" ]]; then
	fd -e c -e h | entr sh -c 'make clean && make'
fi

