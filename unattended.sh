#!/usr/bin/env bash

# Install nodekit, replacing any previous nodekit (FORCE_INSTALL) and skipping the interactive bootstrap
wget -qO- https://nodekit.run/install.sh | NODEKIT_FORCE_INSTALL=1 NODEKIT_SKIP_BOOTSTRAP=1 bash

# Install node
./nodekit install -f

# Wait a bit for connections to be established
sleep 2m

# Start a fast catchup
./nodekit catchup start
