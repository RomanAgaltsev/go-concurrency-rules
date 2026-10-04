#!/bin/sh
# cpu-name.sh prints the CPU a measurement ran on, for its "# cpu:" header
# (spec §5.3), from a /proc/cpuinfo — the machine's own unless one is given:
#   sh scripts/cpu-name.sh [cpuinfo]
# It exits 1, printing nothing, when the CPU cannot be named: a measurement
# that cannot say where it was taken would be refused by the gate anyway.
set -eu

cpuinfo=${1:-/proc/cpuinfo}

field() {
	sed -n "s/^$1[[:space:]]*: *//p" "$cpuinfo" | head -n 1 | sed 's/[[:space:]]*$//'
}

name=$(field 'model name')
if [ -z "$name" ]; then
	# arm64 kernels print no model name, only the implementer and part
	# numbers, which name the core (implementer 0x41, part 0xd0c is an Arm
	# Neoverse N1).
	impl=$(field 'CPU implementer')
	part=$(field 'CPU part')
	if [ -n "$impl" ] && [ -n "$part" ]; then
		name="arm64 CPU implementer $impl, part $part"
	fi
fi
if [ -z "$name" ]; then
	echo "cpu-name.sh: $cpuinfo names no CPU (no model name, no CPU implementer and part)" >&2
	exit 1
fi
echo "$name"
