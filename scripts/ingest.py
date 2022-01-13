import csv
import ipaddress
import math
import itertools as it
import redis


def size_to_cidr_mask(c):
    """ c = 2^(32-m), m being the CIDR mask """
    return int(-math.log2(c) + 32)


def parse_rir_file(filename):
    with open(filename) as f:
        rows = csv.reader(f, delimiter='|')
        for r in rows:
            try:
                rir, country_code, ip_version, ip, mask, *_ = r
            except ValueError:
                continue
            if ip == '*':
                continue
            if ip_version == 'ipv4':
                length = int(mask)
                addr = ipaddress.ip_address(ip)
                yield {
                    'v': 4,
                    'int': int(addr),
                    'country': country_code,
                    'range': ip+'/'+str(size_to_cidr_mask(length)),
                    'rir': rir,
                }
            if ip_version == 'ipv6':
                yield {
                    'v': 6,
                    'int': int(ipaddress.IPv6Address(ip)) >> 64,
                    'country': country_code,
                    'range': ip+'/'+mask,
                    'rir': rir,
                }


data = list(it.chain(
    parse_rir_file('delegated-ripencc-extended-latest'),
    parse_rir_file('delegated-arin-extended-latest'),
    parse_rir_file('delegated-apnic-extended-latest'),
    parse_rir_file('delegated-afrinic-extended-latest'),
    parse_rir_file('delegated-lacnic-extended-latest')
))

r = redis.Redis()
with r.pipeline(transaction=False) as p:
    for d in data:
        if d['v'] == 4:
            p.zadd('ip4', {f'{d["country"]}|{d["range"]}': d['int']})
        elif d['v'] == 6:
            p.zadd('ip6', {f'{d["country"]}|{d["range"]}': d['int']})
    p.execute()
