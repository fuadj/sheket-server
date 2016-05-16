#!/usr/bin/env bash

mockgen -source=sh_store.go -destination=mock_sh_store.go -package=models

echo >&2 "Finished"