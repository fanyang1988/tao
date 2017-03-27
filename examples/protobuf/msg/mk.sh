#!/bin/bash

protoc --cpp_out="./cpp/" ./*.proto
protoc --go_out="./go/" ./*.proto