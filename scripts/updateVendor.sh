#!/bin/bash
# This script is useful for keeping projects up to date with dependency changes
# specifically with changes to the iced-mocha/shared repository. Simply doing `dep ensure`
# is not enough to bring in new changes.

if [ -z ${GOPATH} ] 
 then 
	# GOPATH is not set so we must assume user ran this in their go/ directory
	GOPATH=$(pwd)
fi

PROJECT_PATH=${GOPATH}/src/github.com/iced-mocha

rm -r ${PROJECT_PATH}/core/vendor/ 2> /dev/null
rm -r ${PROJECT_PATH}/core/Gopkg.lock 2> /dev/null
cd ${PROJECT_PATH}/core && dep ensure -v ; cd ${PROJECT_PATH}

rm -r ${PROJECT_PATH}/reddit-client/vendor/ 2> /dev/null
rm -r ${PROJECT_PATH}/reddit-client/Gopkg.lock 2> /dev/null
cd ${PROJECT_PATH}/reddit-client && dep ensure -v ; cd ${PROJECT_PATH}

rm -r ${PROJECT_PATH}/hacker-news-client/vendor/ 2> /dev/null
rm -r ${PROJECT_PATH}/hacker-news-client/Gopkg.lock 2> /dev/null
cd ${PROJECT_PATH}/hacker-news-client && dep ensure -v ; cd ${PROJECT_PATH}

rm -r ${PROJECT_PATH}/google-news-client/vendor/ 2> /dev/null
rm -r ${PROJECT_PATH}/google-news-client/Gopkg.lock 2> /dev/null
cd ${PROJECT_PATH}/google-news-client && dep ensure -v ; cd ${PROJECT_PATH}
