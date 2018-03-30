#!/bin/bash
rm -f build/nomad-firehose-linux-amd64
make dist
docker build -t edwardsmith/d3slack:latest .
