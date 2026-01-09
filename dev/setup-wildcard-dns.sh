#!/bin/bash

# Setup wildcard DNS for *.unkey.local using dnsmasq

set -e

# Detect OS
OS="unknown"
if [[ "$OSTYPE" == "darwin"* ]]; then
    OS="macos"
elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    OS="linux"
else
    echo "Unsupported OS: $OSTYPE"
    exit 1
fi

echo "Detected OS: $OS"
echo ""
echo "This script will set up dnsmasq to resolve *.unkey.local to 127.0.0.1"
echo "This allows you to use any subdomain like my-deployment.unkey.local"
echo ""

# Check if dnsmasq is installed
if command -v dnsmasq &> /dev/null; then
    echo "dnsmasq is already installed"
else
    echo "dnsmasq is not installed"
    echo ""
    if [[ "$OS" == "macos" ]]; then
        echo "Would you like to install dnsmasq using Homebrew? (y/n)"
    else
        echo "Would you like to install dnsmasq using your package manager? (y/n)"
    fi
    read -r response
    if [[ "$response" != "y" ]]; then
        echo "Installation cancelled"
        exit 1
    fi

    # Install dnsmasq based on OS
    if [[ "$OS" == "macos" ]]; then
        if ! command -v brew &> /dev/null; then
            echo "Homebrew is not installed. Please install Homebrew first."
            exit 1
        fi
        echo "Installing dnsmasq with Homebrew..."
        brew install dnsmasq
    else
        # Linux installation
        if command -v apt-get &> /dev/null; then
            echo "Installing dnsmasq with apt..."
            sudo apt-get update && sudo apt-get install -y dnsmasq
        elif command -v yum &> /dev/null; then
            echo "Installing dnsmasq with yum..."
            sudo yum install -y dnsmasq
        elif command -v dnf &> /dev/null; then
            echo "Installing dnsmasq with dnf..."
            sudo dnf install -y dnsmasq
        elif command -v pacman &> /dev/null; then
            echo "Installing dnsmasq with pacman..."
            sudo pacman -S --noconfirm dnsmasq
        else
            echo "Could not detect package manager. Please install dnsmasq manually."
            exit 1
        fi
    fi
fi

echo ""
echo "Configuring dnsmasq for *.unkey.local..."

if [[ "$OS" == "macos" ]]; then
    # macOS configuration
    DNSMASQ_CONF="$(brew --prefix)/etc/dnsmasq.conf"

    # Backup existing config if it exists
    if [[ -f "$DNSMASQ_CONF" ]]; then
        cp "$DNSMASQ_CONF" "$DNSMASQ_CONF.backup.$(date +%Y%m%d_%H%M%S)"
        echo "Backed up existing config"
    fi

    # Add our configuration
    echo "address=/unkey.local/127.0.0.1" > "$DNSMASQ_CONF"
    echo "Configured dnsmasq to resolve *.unkey.local to 127.0.0.1"

    # Start dnsmasq service
    echo ""
    echo "Would you like to start dnsmasq as a service? (y/n)"
    read -r response
    if [[ "$response" == "y" ]]; then
        sudo brew services start dnsmasq
        echo "dnsmasq service started"
    fi

    # Setup resolver
    echo ""
    echo "Setting up macOS resolver for .unkey.local domain..."
    sudo mkdir -p /etc/resolver
    echo "nameserver 127.0.0.1" | sudo tee /etc/resolver/unkey.local > /dev/null
    echo "Resolver configured"

else
    # Linux configuration
    DNSMASQ_CONF="/etc/dnsmasq.d/unkey.local.conf"

    # Create configuration in dnsmasq.d directory (included by default in most dnsmasq setups)
    # This keeps our config separate from the main dnsmasq configuration
    {
        echo "# Unkey local development DNS configuration"
        echo "# Resolve all *.unkey.local domains to localhost"
        echo "address=/unkey.local/127.0.0.1"
    } | sudo tee "$DNSMASQ_CONF" > /dev/null
    echo "Configured dnsmasq to resolve *.unkey.local to 127.0.0.1"

    # Restart dnsmasq service
    echo ""
    echo "Would you like to restart dnsmasq service? (y/n)"
    read -r response
    if [[ "$response" == "y" ]]; then
        if systemctl is-active --quiet dnsmasq; then
            sudo systemctl restart dnsmasq
            echo "dnsmasq service restarted"
        else
            sudo systemctl start dnsmasq
            sudo systemctl enable dnsmasq
            echo "dnsmasq service started and enabled"
        fi
    fi

    # Configure systemd-resolved or NetworkManager if present
    if systemctl is-active --quiet systemd-resolved; then
        echo ""
        echo "systemd-resolved detected. You may need to configure it to use dnsmasq."
        echo "Add 'DNS=127.0.0.1' to /etc/systemd/resolved.conf and restart systemd-resolved"
    fi
fi

echo ""
echo "Setup complete!"
echo ""
echo "Test your setup with:"
echo "  dig test.unkey.local"
echo "  ping my-deployment.unkey.local"
echo "  curl http://anything.unkey.local"
echo ""
echo "To undo these changes:"
if [[ "$OS" == "macos" ]]; then
    echo "  sudo brew services stop dnsmasq"
    echo "  sudo rm /etc/resolver/unkey.local"
    echo "  brew uninstall dnsmasq  # optional"
else
    echo "  sudo systemctl stop dnsmasq"
    echo "  sudo rm $DNSMASQ_CONF"
    echo "  sudo systemctl restart dnsmasq"
fi
