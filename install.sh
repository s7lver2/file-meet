#!/usr/bin/env bash

set -eo pipefail

REPO_PATH=/home/$USER/Projects/file-meet
#REPO_PATH=/home/$USER/file-meet

BINARY_PATH=/home/$USER/.local/
SERVICE_PATH=/etc/systemd/system/meet.service

if [[ $1 == "--zephyros-source-build" ]]; then
    echo "zephyros souce compilation mode enabled!"
    echo "[1/6] cleaning cache"
    sudo rm -r build

    echo "[2/6] Updating Repo"
    git pull

    echo "[3/6] Copying files"
    mkdir build
    cp -r backend build/backend
    cp -r cli build/cli

    echo "[4/6] building from source"
    cd build/backend
    go build -x -o meet-backend
    cd ../..

    cd build/cli
    go build -x -o meet
    cd ../..

    echo "[5/6] creating service file"
    sudo tee "meet.service" > /dev/null <<EOF
    [Unit]
    Description=meet - LAN fast and secure file sharing
    After=network.target

    [Service]
    User=$USER
    WorkingDirectory=$REPO_PATH
    ExecStart=$BINARY_PATH/bin/meet-backend
    Restart=always
    RestartSec=5
    Environment="CONFIG_PATH=$REPO_PATH/config.toml"

    [Install]
    WantedBy=multi-user.target
EOF

    echo "[6/6] copying all files to project root"
    cp -r build/backend/meet-backend .
    cp -r build/cli/meet .
    
else
    echo "[1/8] cleaning cache"
    sudo rm -r $REPO_PATH/build

    echo "[2/8] Updating Repo"
    git pull

    echo "[3/8] Copying files"
    mkdir build
    cp -r backend $REPO_PATH/build/backend
    cp -r cli $REPO_PATH/build/cli

    echo "[4/8] building from source"
    cd $REPO_PATH/build/backend
    go build -x -o meet-backend

    cd $REPO_PATH/build/cli
    go build -x -o meet

    echo "[5/8] copying binaries to PATH"
    mkdir -p $BINARY_PATH/bin
    cp -r $REPO_PATH/build/backend/meet-backend $BINARY_PATH/bin/meet-backend
    chmod +x $BINARY_PATH/bin/meet-backend

    cp -r $REPO_PATH/build/cli/meet /home/$USER/.local/bin/meet
    chmod +x /home/$USER/.local/bin/meet

    echo "[6/8] copying default config file"
    cp -r $REPO_PATH/default.config.toml  $REPO_PATH/config.toml

    echo "[7/8] creating service"
    sudo tee "$SERVICE_PATH" > /dev/null <<EOF
    [Unit]
    Description=meet - LAN fast and secure file sharing
    After=network.target

    [Service]
    User=$USER
    WorkingDirectory=$REPO_PATH
    ExecStart=$BINARY_PATH/bin/meet-backend
    Restart=always
    RestartSec=5
    Environment="CONFIG_PATH=$REPO_PATH/config.toml"

    [Install]
    WantedBy=multi-user.target
EOF
    systemctl start meet.service
    systemctl enable meet.service
fi
