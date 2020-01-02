#!/usr/bin/env python3

"""
Usage: foo [OPTION]...

  --foo1   useful option foo
  --bar1   useful option bar
"""
import sys

if __name__ == "__main__":
    print(__doc__, file=sys.stderr)
