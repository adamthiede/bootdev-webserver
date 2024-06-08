#!/bin/sh
find . -iname '*.go'|xargs gofmt -w 
