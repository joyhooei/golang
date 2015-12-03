#!/bin/sh

dir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

cd $dir

date "+%Y-%m-%d %H:%M:%S" >> ../log/stater.log
if [ $# -eq 1 ];then
    ../bin/stater "mumu:jkwen2k3x@tcp(192.168.1.78:13307)/mumu_stat" $1 >> ../log/stater.log
    ../bin/stater "mumu:jkwen2k3x@tcp(192.168.1.78:13307)/mumu_dstat" $1 >> ../log/dstater.log
elif [ $# -gt 1 ];then
    echo $#
    ../bin/stater "mumu:jkwen2k3x@tcp(192.168.1.78:13307)/mumu_stat" $1 "$2" >> ../log/stater.log
    ../bin/stater "mumu:jkwen2k3x@tcp(192.168.1.78:13307)/mumu_dstat" $1 "$2" >> ../log/dstater.log
else
    echo "Usage : $0 [day|hour] [date(optional)]"
fi
