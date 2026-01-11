#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
FIRECRACKER_SOCKET="${FIRECRACKER_API_SOCKET:-/tmp/firecracker.socket}"
TAP_DEVICE="${SEAR_TAP_DEVICE:-tap0}"
TAP_IP="${SEAR_TAP_IP:-172.16.0.1}"
GUEST_IP="${SEAR_GUEST_IP:-172.16.0.2}"
FIRECRACKER_BINARY="${FIRECRACKER_BINARY:-./firecracker}"

print_status() {
    echo -e "${BLUE}[SEAR]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[!]${NC} $1"
}

print_error() {
    echo -e "${RED}[✗]${NC} $1"
}

# Check if running as root for network setup
check_root() {
    if [ "$EUID" -ne 0 ]; then
        print_warning "Network setup requires root privileges"
        print_status "Please run this script with sudo or setup network manually"
        return 1
    fi
    return 0
}

# Setup network infrastructure
setup_network() {
    print_status "Setting up network infrastructure..."
    
    # Get host network interface
    HOST_IFACE=$(ip -j route list default 2>/dev/null | jq -r '.[0].dev' 2>/dev/null)
    if [ -z "$HOST_IFACE" ] || [ "$HOST_IFACE" = "null" ]; then
        HOST_IFACE=$(ip route show default 2>/dev/null | grep -oP '(?<=dev )\S+' | head -1)
    fi
    
    if [ -z "$HOST_IFACE" ]; then
        print_error "Could not detect default network interface"
        HOST_IFACE="eth0"
        print_warning "Using default: $HOST_IFACE"
    else
        print_status "Detected host interface: $HOST_IFACE"
    fi
    
    # Remove existing TAP device if it exists
    if ip link show "$TAP_DEVICE" >/dev/null 2>&1; then
        print_status "Removing existing TAP device: $TAP_DEVICE"
        ip link del "$TAP_DEVICE" 2>/dev/null || true
    fi
    
    # Create TAP device
    print_status "Creating TAP device: $TAP_DEVICE"
    ip tuntap add dev "$TAP_DEVICE" mode tap 2>/dev/null || {
        print_error "Failed to create TAP device (may already exist)"
        ip link show "$TAP_DEVICE" >/dev/null 2>&1 || exit 1
    }
    
    # Configure TAP IP
    print_status "Configuring TAP IP: $TAP_IP/30"
    ip addr add "${TAP_IP}/30" dev "$TAP_DEVICE" 2>/dev/null || {
        print_warning "IP address may already be configured"
    }
    
    # Bring up TAP device
    ip link set dev "$TAP_DEVICE" up
    
    # Enable IP forwarding
    print_status "Enabling IP forwarding..."
    echo 1 > /proc/sys/net/ipv4/ip_forward
    
    # Set forward policy to ACCEPT
    iptables -P FORWARD ACCEPT
    
    # Setup NAT/MASQUERADE
    print_status "Configuring NAT for internet access..."
    iptables -t nat -D POSTROUTING -o "$HOST_IFACE" -j MASQUERADE 2>/dev/null || true
    iptables -t nat -A POSTROUTING -o "$HOST_IFACE" -j MASQUERADE
    
    print_success "Network setup complete"
    echo ""
    echo "Network configuration:"
    echo "  TAP Device:   $TAP_DEVICE"
    echo "  TAP IP:       $TAP_IP"
    echo "  Gateway IP:   $TAP_IP"
    echo "  Guest IP:     $GUEST_IP"
    echo "  Host Interface: $HOST_IFACE"
    echo ""
}

# Cleanup network on exit
cleanup_network() {
    if [ "$EUID" -eq 0 ]; then
        print_status "Cleaning up network..."
        ip link del "$TAP_DEVICE" 2>/dev/null || true
        print_success "Network cleanup complete"
    fi
}

# Start Firecracker
start_firecracker() {
    print_status "Starting Firecracker..."
    
    # Check if Firecracker binary exists
    if [ ! -f "$FIRECRACKER_BINARY" ]; then
        print_error "Firecracker binary not found: $FIRECRACKER_BINARY"
        print_status "Please specify path with: FIRECRACKER_BINARY=/path/to/firecracker $0 run"
        exit 1
    fi
    
    # Remove existing socket
    rm -f "$FIRECRACKER_SOCKET"
    
    # Start Firecracker in background
    sudo "$FIRECRACKER_BINARY" --api-sock "$FIRECRACKER_SOCKET" --enable-pci &
    FC_PID=$!
    
    # Wait for socket to be ready
    print_status "Waiting for Firecracker API socket..."
    for i in {1..30}; do
        if [ -S "$FIRECRACKER_SOCKET" ]; then
            print_success "Firecracker is ready"
            return 0
        fi
        sleep 0.1
    done
    
    print_error "Firecracker failed to start"
    exit 1
}

# Stop Firecracker
stop_firecracker() {
    if [ -n "$FC_PID" ] && kill -0 "$FC_PID" 2>/dev/null; then
        print_status "Stopping Firecracker..."
        kill "$FC_PID" 2>/dev/null || true
        wait "$FC_PID" 2>/dev/null || true
    fi
}

# Full cleanup
full_cleanup() {
    stop_firecracker
    cleanup_network
}

# Print usage
usage() {
    echo "SEAR - Firecracker VM Launcher"
    echo ""
    echo "Usage: $0 [command] [options]"
    echo ""
    echo "Commands:"
    echo "  setup       Setup network infrastructure only"
    echo "  start       Start Firecracker only"
    echo "  run         Run sear (default)"
    echo "  cleanup     Clean up network and Firecracker"
    echo "  all         Full setup + start + run (default when no command)"
    echo ""
    echo "Environment Variables:"
    echo "  FIRECRACKER_API_SOCKET  Firecracker API socket path (default: /tmp/firecracker.socket)"
    echo "  FIRECRACKER_BINARY      Path to Firecracker binary (default: ./firecracker)"
    echo "  SEAR_TAP_DEVICE         TAP device name (default: tap0)"
    echo "  SEAR_TAP_IP             TAP device IP (default: 172.16.0.1)"
    echo "  SEAR_GUEST_IP           VM guest IP (default: 172.16.0.2)"
    echo ""
    echo "Examples:"
    echo "  $0 all                    # Full setup and run"
    echo "  sudo $0 setup             # Setup network only (requires root)"
    echo "  $0 run                    # Run sear (requires running Firecracker)"
    echo "  $0 cleanup                # Clean up resources"
    echo ""
}

# Main run function
do_run() {
    local sear_binary="${1:-./sear}"
    
    # Check if sear binary exists
    if [ ! -f "$sear_binary" ]; then
        print_error "SEAR binary not found: $sear_binary"
        print_status "Please build it first: cd /home/nws/dev/sear && go build -o sear ."
        exit 1
    fi
    
    # Check if SSH key exists
    local ssh_key="$HOME/.config/sear/sear_key"
    if [ ! -f "$ssh_key" ]; then
        print_warning "SSH key not found: $ssh_key"
        print_status "Generating SSH key..."
        mkdir -p "$HOME/.config/sear"
        ssh-keygen -t rsa -b 4096 -f "$ssh_key" -N "" -C "sear vm access"
        print_success "SSH key generated"
        echo ""
        echo "IMPORTANT: You need to add this public key to your VM's authorized_keys:"
        echo "  cat $ssh_key.pub"
        echo ""
    fi
    
    print_status "Starting SEAR..."
    "$sear_binary" run "$@"
}

# Main entry point
main() {
    local command="${1:-all}"
    shift 2>/dev/null || true
    
    case "$command" in
        setup)
            check_root || exit 1
            setup_network
            ;;
        start)
            check_root || exit 1
            start_firecracker
            # Keep running
            wait
            ;;
        run)
            do_run "$@"
            ;;
        cleanup)
            full_cleanup
            ;;
        all)
            # Full workflow
            if [ "$EUID" -eq 0 ]; then
                setup_network
                start_firecracker
                trap full_cleanup EXIT
                do_run "$@"
            else
                print_warning "Not running as root - skipping network setup"
                print_status "Make sure network is already configured, then run:"
                echo "  $0 run"
                do_run "$@"
            fi
            ;;
        help|--help|-h)
            usage
            ;;
        *)
            print_error "Unknown command: $command"
            usage
            exit 1
            ;;
    esac
}

main "$@"
