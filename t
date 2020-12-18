#!/bin/bash
IP=$( sed "$1q;d" awses )
echo "ubuntu@$IP"
scp -i ~/.ssh/bristol_aws.pem ./gameoflife ubuntu@"$IP":./gameoflife
