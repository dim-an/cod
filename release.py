#!/usr/bin/env python3
# -*- coding: utf-8 -*-

import subprocess
import os
import sys

def fatal(msg):
    print(msg, file=sys.stderr)
    exit(1)

if __name__ == "__main__":
    os.chdir(os.path.dirname(os.path.realpath(__file__)))
    if not os.path.exists("cod"):
        fatal("cod binary is not found")
    try:
        clients_text = subprocess.check_output(["cod", "api", "list-clients"])
    except subprocess.CalledProcessError:
        subprocess.call(["killall", "cod"])
        os.rename("cod", os.path.expanduser("~/.local/bin/cod"))
    else:
        subprocess.call(["killall", "cod"])
        os.rename("cod", os.path.expanduser("~/.local/bin/cod"))

        for line in clients_text.decode('utf-8').strip().split('\n'):
            pid, shell = line.strip().split()
            subprocess.check_call(["cod", "api", "attach", pid, shell])


