#!/bin/bash
IP=$( sed "$1q;d" awses )
echo "$IP"
