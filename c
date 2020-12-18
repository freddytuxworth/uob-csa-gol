#!/bin/bash
ssh -i ~/.ssh/bristol_aws.pem ubuntu@$( sed "$1q;d" awses ) "$2"
