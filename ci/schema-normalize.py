#!/usr/bin/env python3
"""Normalize pg_dump administrative noise, preserving application DDL verbatim."""
import sys
for line in sys.stdin:
    s = line.strip()
    if not s or s.startswith('--') or s.startswith('\\restrict ') or s.startswith('\\unrestrict '):
        continue
    # Session configuration is not application structure. Owners/ACLs excluded by pg_dump.
    if s.startswith('SET ') or s.startswith('SELECT pg_catalog.set_config('):
        continue
    print(line.rstrip())
