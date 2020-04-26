#!/usr/bin/env python3
import argparse

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--parser-argument", help="some help")

    subparsers = parser.add_subparsers()
    subcommand1_parser = subparsers.add_parser("sub-command1", help="some help")
    subcommand1_parser.add_argument("--sub-command1-argument")

    subcommand2_parser = subparsers.add_parser("sub-command2", help="some help")
    subcommand2_parser.add_argument("--sub-command2-argument")

    parser.parse_args()

