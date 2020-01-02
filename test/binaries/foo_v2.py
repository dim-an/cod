#!/usr/bin/env python3

"""
Usage: foo [OPTION]...

  --foo2   useful option foo
  --bar2   useful option bar
  --qux2   useful option qux
"""
import sys

if __name__ == "__main__":
    print(__doc__, file=sys.stderr)
