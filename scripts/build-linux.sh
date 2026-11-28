#!/usr/bin/env bash
set -euo pipefail
export LC_ALL=C

# Both CI and release packaging use the same architecture list and checks.
version=${1:-dev}
commit=$(git rev-parse HEAD)
targets=(amd64 arm64 armv7 armv6)

for target in "${targets[@]}"; do
    case "$target" in
        amd64)
            arch=amd64
            machine='Advanced Micro Devices X86-64'
            class=ELF64
            ;;
        arm64)
            arch=arm64
            machine='AArch64'
            class=ELF64
            ;;
        armv*)
            arch=arm
            machine='ARM'
            class=ELF32
            ;;
    esac

    build_vars=("ARCH=$arch" "VERSION=$version" "COMMIT=$commit")
    if [[ "$arch" == arm ]]; then
        build_vars+=("ARM_VERSION=${target#armv}")
    fi
    task build-linux "${build_vars[@]}"

    binary="dist/switchbot-exporter-linux-$target"
    header=$(readelf -h "$binary")
    grep -Eq "Class:[[:space:]]+$class$" <<< "$header"
    grep -Eq "Machine:[[:space:]]+$machine$" <<< "$header"
    segments=$(readelf -l "$binary")
    dynamic=$(readelf -d "$binary")
    if grep -qw INTERP <<< "$segments" || grep -q '(NEEDED)' <<< "$dynamic"; then
        echo "$binary requires a dynamic linker or shared libraries" >&2
        exit 1
    fi
    metadata=$(go version -m "$binary")
    grep -qw GOOS=linux <<< "$metadata"
    grep -qw "GOARCH=$arch" <<< "$metadata"
    grep -qw CGO_ENABLED=0 <<< "$metadata"
    if [[ "$arch" == arm ]]; then
        grep -Eq "[[:space:]]GOARM=${target#armv}(,hardfloat)?$" <<< "$metadata"
    fi
done
