#!/usr/bin/env bash
set -euo pipefail

binary=$1
export LC_ALL=C

header=$(readelf -h "$binary")
grep -Eq 'Class:[[:space:]]+ELF32' <<< "$header"
grep -Eq 'Machine:[[:space:]]+ARM' <<< "$header"
segments=$(readelf -l "$binary")
dynamic=$(readelf -d "$binary")
if grep -qw INTERP <<< "$segments" || grep -q '(NEEDED)' <<< "$dynamic"; then
    echo "ARMv6 binary requires a dynamic linker or shared libraries" >&2
    exit 1
fi
metadata=$(go version -m "$binary")
grep -Eq '[[:space:]]GOARM=6(,hardfloat)?$' <<< "$metadata"
grep -qw CGO_ENABLED=0 <<< "$metadata"
