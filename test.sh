#!/bin/sh

go test jsparser.go scratch.go jsparser_test.go -v

go test jsparser.go scratch.go jsparser_test.go -v --minify

go test jsparser.go scratch.go jsparser_test.go -v --parseall