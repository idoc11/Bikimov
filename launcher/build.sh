#!/bin/sh
# Сборка dist/Shtorm.exe: нужен только Go. Запускать из любой папки.
set -e
cd "$(dirname "$0")"
# копия игры со шрифтами из папки, а не из интернета — exe работает без сети
sed -e '/fonts.googleapis.com" *>$/d' -e 's|<link href="https://fonts.googleapis.com/css2[^"]*" rel="stylesheet">|<link href="fonts/fonts.css" rel="stylesheet">|' ../index.html > web/index.html
grep -q 'fonts/fonts.css' web/index.html
mkdir -p ../dist
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-H windowsgui -s -w" -o ../dist/Shtorm.exe .
ls -l ../dist/Shtorm.exe
