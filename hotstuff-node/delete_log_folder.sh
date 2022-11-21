#!/usr/bin/bash

read -r -p "Are you sure? [y/N] " response
case "$response" in
    [yY][eE][sS]|[yY])
       rm -rf logs 
        ;;
    *)
        echo "exiting"
	    ;;
esac
