#!/usr/bin/env bash
# Bootstrap + reattach the LVM volume group that backs TopoLVM in minikube.
# Idempotent: safe to run on first install, every Tilt up, and after
# `minikube stop`/`start` (which loses the loop device and deactivates the VG).
#
# Runs INSIDE the minikube node via `minikube ssh -- sudo bash -s`.
# Usage:
#   minikube ssh -- sudo bash -s -- <size> < setup-vg.sh
#   <size> e.g. "20G" — sparse, only consumes what's actually written.

set -euo pipefail

SIZE="${1:-20G}"
IMG=/var/lib/topolvm-backing.img
VG=topolvm-vg

# 0. lvm2 isn't shipped in minikube's kicbase image. apt-get install is
#    idempotent (no-op once present), so the re-run cost on every Tilt up
#    is just one cache check.
if ! command -v pvcreate >/dev/null 2>&1; then
  echo "topolvm: installing lvm2 (one-time)..."
  DEBIAN_FRONTEND=noninteractive apt-get update -qq
  DEBIAN_FRONTEND=noninteractive apt-get install -y -qq lvm2 >/dev/null
fi

# Disable LVM's udev integration. Inside minikube's containerized "node",
# udev isn't running, so LVM waits forever for /dev/<vg>/<lv> symlinks that
# never appear. With these flags LVM creates the device nodes itself
# instead of relying on udev events. Without this, every `lvcreate` from
# topolvm fails with "device not cleared / Failed to wipe start of new LV".
LVM_OVERRIDE=/etc/lvm/lvm.conf
if ! grep -q "# unkey-dev-overrides" "$LVM_OVERRIDE" 2>/dev/null; then
  cat >> "$LVM_OVERRIDE" <<'EOF'

# unkey-dev-overrides
activation {
    udev_sync = 0
    udev_rules = 0
    verify_udev_operations = 0
}
EOF
  echo "topolvm: patched $LVM_OVERRIDE to bypass udev"
fi

# 1. Sparse backing file. truncate -s creates a hole of the requested size;
#    actual disk usage grows only as data is written, so a 20 GiB ceiling
#    costs ~0 bytes on a fresh cluster.
if [ ! -f "$IMG" ]; then
  truncate -s "$SIZE" "$IMG"
  echo "topolvm: created backing file $IMG ($SIZE sparse)"
fi

# 2. Loopback device. losetup -j lists existing attachments for the file
#    so we don't double-attach across restarts.
LOOP=$(losetup -j "$IMG" | cut -d: -f1 | head -n1)
if [ -z "$LOOP" ]; then
  LOOP=$(losetup -f --show "$IMG")
  echo "topolvm: attached $IMG to $LOOP"
fi

# 3. PV + VG. vgs returns nonzero if the VG doesn't exist; only run the
#    create commands then. Re-running pvcreate on an in-use PV would error.
if ! vgs "$VG" >/dev/null 2>&1; then
  pvcreate "$LOOP"
  vgcreate "$VG" "$LOOP"
  echo "topolvm: created PV $LOOP and VG $VG"
fi

# 4. Activate any LVs in the VG. After `minikube stop`/`start` the device
#    nodes under /dev/$VG/ disappear; vgchange -ay rebuilds them so kubelet
#    can mount existing PVCs again.
vgchange -ay "$VG" >/dev/null

echo "topolvm: VG $VG ready (backing $LOOP)"
vgs "$VG"
