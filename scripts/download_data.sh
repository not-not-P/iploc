#!/bin/bash

wget https://ftp.arin.net/pub/stats/arin/delegated-arin-extended-latest&
wget https://ftp.ripe.net/ripe/stats/delegated-ripencc-extended-latest&
wget https://ftp.apnic.net/stats/apnic/delegated-apnic-extended-latest&
wget https://ftp.apnic.net/stats/afrinic/delegated-afrinic-extended-latest&
wget https://ftp.apnic.net/stats/lacnic/delegated-lacnic-extended-latest&

wait
