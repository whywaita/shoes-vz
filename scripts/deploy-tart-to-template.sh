#!/bin/bash
# Script to convert Tart VM to shoes-vz template
# Usage: deploy-tart-to-template.sh <source-vm-name> <template-name>
#
# This script performs the following:
# 1. Copy disk image and NVRAM from Tart VM using APFS clone
# 2. Generate HardwareModel.json
# 3. Convert to format usable as shoes-vz template

set -euo pipefail

# Colored output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Argument check
if [ $# -ne 2 ]; then
    log_error "Usage: $0 <source-vm-name> <template-name>"
    log_error "  source-vm-name: Name of the Tart VM to convert"
    log_error "  template-name: Name for the shoes-vz template"
    exit 1
fi

SOURCE_VM="$1"
TEMPLATE_NAME="$2"

# Path configuration
TART_VMS_DIR="${HOME}/.tart/vms"
SOURCE_VM_DIR="${TART_VMS_DIR}/${SOURCE_VM}"
TEMPLATE_BASE_DIR="${TEMPLATE_BASE_DIR:-/opt/myshoes/vz/templates}"
TEMPLATE_DIR="${TEMPLATE_BASE_DIR}/${TEMPLATE_NAME}"

log_info "Converting Tart VM '${SOURCE_VM}' to shoes-vz template '${TEMPLATE_NAME}'"

# Check existence of Tart VM
if [ ! -d "${SOURCE_VM_DIR}" ]; then
    log_error "Source VM not found: ${SOURCE_VM_DIR}"
    log_error "Available VMs:"
    ls -1 "${TART_VMS_DIR}" 2>/dev/null || log_error "  No VMs found"
    exit 1
fi

log_info "Source VM found at: ${SOURCE_VM_DIR}"

# Check existence of required files
required_files=("disk.img" "nvram.bin" "config.json")
for file in "${required_files[@]}"; do
    if [ ! -f "${SOURCE_VM_DIR}/${file}" ]; then
        log_error "Required file not found: ${file}"
        exit 1
    fi
done

# Check if VM is not running
if [ -S "${SOURCE_VM_DIR}/control.sock" ]; then
    log_warn "control.sock found, checking if VM is actually running..."

    # Check if socket is actually in use with lsof
    if lsof "${SOURCE_VM_DIR}/control.sock" &>/dev/null; then
        log_error "VM is actually running. Please stop the VM before converting to template"
        log_error "Run: tart stop ${SOURCE_VM}"
        exit 1
    else
        log_info "control.sock is stale (VM is not running), removing it..."
        rm -f "${SOURCE_VM_DIR}/control.sock"
    fi
fi

# Create template directory
if [ -d "${TEMPLATE_DIR}" ]; then
    log_warn "Template directory already exists: ${TEMPLATE_DIR}"
    read -p "Do you want to overwrite it? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Aborted."
        exit 0
    fi
    log_info "Removing existing template directory..."
    rm -rf "${TEMPLATE_DIR}"
fi

log_info "Creating template directory: ${TEMPLATE_DIR}"
mkdir -p "${TEMPLATE_DIR}"

# Copy disk image using APFS clone
log_info "Cloning disk image using APFS clone (this should be fast)..."
START_TIME=$(date +%s)

# Create APFS clone with macOS cp -c option
# This uses Copy-on-Write (COW), so essentially only metadata is copied
if ! cp -c "${SOURCE_VM_DIR}/disk.img" "${TEMPLATE_DIR}/Disk.img" 2>/dev/null; then
    log_warn "APFS clone failed, falling back to regular copy..."
    log_warn "This may take a while for large disk images..."
    cp "${SOURCE_VM_DIR}/disk.img" "${TEMPLATE_DIR}/Disk.img"
fi

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))
log_info "Disk image copied in ${DURATION} seconds"

# Copy NVRAM as AuxiliaryStorage
log_info "Copying NVRAM as AuxiliaryStorage..."
cp -c "${SOURCE_VM_DIR}/nvram.bin" "${TEMPLATE_DIR}/AuxiliaryStorage" 2>/dev/null || \
    cp "${SOURCE_VM_DIR}/nvram.bin" "${TEMPLATE_DIR}/AuxiliaryStorage"

# Generate HardwareModel.json
log_info "Extracting hardware model from config.json..."

# Extract hardwareModel field using jq
if ! command -v jq &> /dev/null; then
    log_error "jq is not installed. Please install it with: brew install jq"
    exit 1
fi

# Extract hardwareModel from config.json and create HardwareModel.json
HARDWARE_MODEL=$(jq -r '.hardwareModel' "${SOURCE_VM_DIR}/config.json")

if [ -z "${HARDWARE_MODEL}" ] || [ "${HARDWARE_MODEL}" = "null" ]; then
    log_error "Failed to extract hardwareModel from config.json"
    exit 1
fi

# Create HardwareModel.json
cat > "${TEMPLATE_DIR}/HardwareModel.json" << EOF
{
  "hardwareModel": "${HARDWARE_MODEL}"
}
EOF

log_info "HardwareModel.json created"

# Display template information
log_info "Template created successfully!"
log_info ""
log_info "Template location: ${TEMPLATE_DIR}"
log_info "Files:"
ls -lh "${TEMPLATE_DIR}"
log_info ""
log_info "Disk space usage (APFS shows apparent size, actual space is shared):"
du -sh "${TEMPLATE_DIR}"
log_info ""
log_info "To use this template with shoes-vz-agent, run:"
log_info "  shoes-vz-agent -template-path ${TEMPLATE_DIR} ..."
log_info ""
log_info "Note: The template files use APFS cloning (Copy-on-Write)."
log_info "      They share space with the source VM until modified."
