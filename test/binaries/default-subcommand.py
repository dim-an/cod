#!/usr/bin/env python3
# -*- coding: utf-8 -*-

import sys


DEFAULT_HELP = """\
usage: default-subcommand.py FLAGS...

  --no-sub-command-flag
"""


SUBCOMMAND_1_HELP = """\
usage: default-subcommand.py sub-command1 FLAGS...

  --sub-command1-flag
"""

SUBCOMMAND_2_HELP = """\
usage: default-subcommand.py sub-command2 FLAGS...

  --sub-command2-flag
"""

if __name__ == "__main__":
    for a in sys.argv:
        if a == "sub-command1":
            sys.stderr.write(SUBCOMMAND_1_HELP)
            exit()
        elif a == "sub-command2":
            sys.stderr.write(SUBCOMMAND_2_HELP)
            exit()
    else:
        sys.stderr.write(DEFAULT_HELP)
        exit()
