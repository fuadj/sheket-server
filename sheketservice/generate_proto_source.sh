#!/bin/bash

# syntax protoc -I dir file.proto --go_out=plugins=grpc:dir
protoc -I ./ sheket_service.proto --go_out=plugins=grpc:./
