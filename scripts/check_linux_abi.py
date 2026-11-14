"""Reject ELF artifacts requiring GLIBC newer than the supported release baseline."""
import re
import subprocess
import sys


def required_versions(output):
    # Only version-needs entries, not arbitrary embedded strings or definitions.
    needs = output.split("Version needs section", 1)
    if len(needs) != 2:
        raise ValueError("missing ELF version needs section")
    versions = re.findall(r"Name: GLIBC_(\d+(?:\.\d+)+)\b", needs[1])
    if not versions:
        raise ValueError("no GLIBC version requirements found")
    return {tuple(map(int, version.split('.'))) for version in versions}


def check(output, baseline):
    versions = required_versions(output)
    maximum = max(versions)
    if maximum > tuple(map(int, baseline.split('.'))):
        raise ValueError(f"requires GLIBC {'.'.join(map(str, maximum))}; maximum supported is {baseline}")
    return '.'.join(map(str, maximum))


if __name__ == '__main__':
    output = subprocess.check_output(['readelf', '--version-info', sys.argv[1]], text=True)
    try:
        maximum = check(output, sys.argv[2])
    except ValueError as error:
        sys.exit(str(error))
    print(f'PASS: maximum required GLIBC is {maximum} (baseline {sys.argv[2]})')
