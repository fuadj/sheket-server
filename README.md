# Sheket backend in golang

This is the backend for Sheket, where the syncing
is done. When changes are pushed from the Android client,
applies the user's changes, then collects any changes
since last sync and returns that to the client.
Communication is done throught Google Grpc. 
