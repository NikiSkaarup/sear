# SEAR

`sear` is a cli tool written in golang use an already running instance firecracker to spawn a microvm with the given configured profile eg. `sear rust-dev` with `pwd` mounted in the vm, and put you in an interactive shell inside the vm

if tools are defined in the profile those commands are run inside the vm

## Firecracker
Firecracker is started separately using

```sh
FIRECRACKER_API_SOCKET="/tmp/firecracker.socket"
# Remove API unix socket
sudo rm -f $FIRECRACKER_API_SOCKET
# Run firecracker
sudo ./firecracker --api-sock "${FIRECRACKER_API_SOCKET}" --enable-pci
```


## rootfs and kernel image
a rootfs and kernel image is prepared using

```sh
ARCH="$(uname -m)"
release_url="https://github.com/firecracker-microvm/firecracker/releases"
latest_version=$(basename $(curl -fsSLI -o /dev/null -w  %{url_effective} ${release_url}/latest))
CI_VERSION=${latest_version%.*}
latest_kernel_key=$(curl "http://spec.ccfc.min.s3.amazonaws.com/?prefix=firecracker-ci/$CI_VERSION/$ARCH/vmlinux-&list-type=2" \
    | grep -oP "(?<=<Key>)(firecracker-ci/$CI_VERSION/$ARCH/vmlinux-[0-9]+\.[0-9]+\.[0-9]{1,3})(?=</Key>)" \
    | sort -V | tail -1)

# Download a linux kernel binary
wget "https://s3.amazonaws.com/spec.ccfc.min/${latest_kernel_key}"

latest_ubuntu_key=$(curl "http://spec.ccfc.min.s3.amazonaws.com/?prefix=firecracker-ci/$CI_VERSION/$ARCH/ubuntu-&list-type=2" \
    | grep -oP "(?<=<Key>)(firecracker-ci/$CI_VERSION/$ARCH/ubuntu-[0-9]+\.[0-9]+\.squashfs)(?=</Key>)" \
    | sort -V | tail -1)
ubuntu_version=$(basename $latest_ubuntu_key .squashfs | grep -oE '[0-9]+\.[0-9]+')

# Download a rootfs from Firecracker CI
wget -O ubuntu-$ubuntu_version.squashfs.upstream "https://s3.amazonaws.com/spec.ccfc.min/$latest_ubuntu_key"

# The rootfs in our CI doesn't contain SSH keys to connect to the VM
# For the purpose of this demo, let's create one and patch it in the rootfs
unsquashfs ubuntu-$ubuntu_version.squashfs.upstream
ssh-keygen -f id_rsa -N ""
cp -v id_rsa.pub squashfs-root/root/.ssh/authorized_keys
mv -v id_rsa ./ubuntu-$ubuntu_version.id_rsa
# create ext4 filesystem image
sudo chown -R root:root squashfs-root
truncate -s 1G ubuntu-$ubuntu_version.ext4
sudo mkfs.ext4 -d squashfs-root -F ubuntu-$ubuntu_version.ext4

# Verify everything was correctly set up and print versions
echo
echo "The following files were downloaded and set up:"
KERNEL=$(ls vmlinux-* | tail -1)
[ -f $KERNEL ] && echo "Kernel: $KERNEL" || echo "ERROR: Kernel $KERNEL does not exist"
ROOTFS=$(ls *.ext4 | tail -1)
e2fsck -fn $ROOTFS &>/dev/null && echo "Rootfs: $ROOTFS" || echo "ERROR: $ROOTFS is not a valid ext4 fs"
KEY_NAME=$(ls *.id_rsa | tail -1)
[ -f $KEY_NAME ] && echo "SSH Key: $KEY_NAME" || echo "ERROR: Key $KEY_NAME does not exist"
```
and placed in a location defined by configuration

## Configuration

File Location

- Default: ~/.config/sear/config.yaml
- Environment Override: $XDG_CONFIG_HOME/sear/config.yaml

example:
```yaml
# Default profile to use
default_profile: minimal

# Profile definitions
profiles:
  rust-dev:
    tools:
      - name: rustc
        manager: apt
      - name: cargo
        manager: apt
      - name: git
        manager: apt
    vm:
      vcpus: 2
      memory_mib: 4096
      rootfs: ~/.cache/sear/rootfses/ubuntu-noble.ext4
      kernel: ~/.cache/sear/kernels/vmlinux
      kernel_args: "console=ttyS0 reboot=k panic=1 pci=off nomodules"

  minimal:
    tools: []
    vm:
      vcpus: 1
      memory_mib: 512
```
