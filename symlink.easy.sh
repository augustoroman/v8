#!/usr/bin/env bash

rm include libv8 2>/dev/null
if [ "$(uname)" = "Darwin" ];then
# Mac OS X 操作系统
ln -s -f 6.3.292.48.1-x86_64-darwin/include include
ln -s -f 6.3.292.48.1-x86_64-darwin/libv8 libv8
elif [ "$(expr substr $(uname -s) 1 5)" = "Linux" ];then
# GNU/Linux操作系统
ln -s -f 6.3.292.48.1-x86_64-linux/include include
ln -s -f 6.3.292.48.1-x86_64-linux/libv8 libv8
fi
