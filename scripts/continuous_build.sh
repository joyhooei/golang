#!/bin/sh

export GOBIN=/root/go/bin
export GOPATH=/root/go
export GOROOT=/usr/local/go

dir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd $dir

while [ 1 ];do
git pull;/usr/local/go/bin/go install yuanfen/qq_service
killall godoc
nohup /usr/local/go/bin/godoc -tabwidth=2 -http=:8082 >/dev/null 2>&1 &
sleep 5
done
