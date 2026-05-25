#!/bin/bash
# SVG Download Script
# Run this in Git Bash or WSL

OUTPUT_DIR="../static/tabbar"
mkdir -p $OUTPUT_DIR

# Colors
REGULAR_COLOR="999999"
FILLED_COLOR="e82127"

echo "Downloading Fluent Icons..."

# home icons
curl -L "https://api.iconify.design/fluent/home-24-regular.svg?color=%23${REGULAR_COLOR}" -o "$OUTPUT_DIR/home.svg"
curl -L "https://api.iconify.design/fluent/home-24-filled.svg?color=%23${FILLED_COLOR}" -o "$OUTPUT_DIR/home-active.svg"

# vehicle icons
curl -L "https://api.iconify.design/fluent/vehicle-car-24-regular.svg?color=%23${REGULAR_COLOR}" -o "$OUTPUT_DIR/vehicle.svg"
curl -L "https://api.iconify.design/fluent/vehicle-car-24-filled.svg?color=%23${FILLED_COLOR}" -o "$OUTPUT_DIR/vehicle-active.svg"

# control icons
curl -L "https://api.iconify.design/fluent/phone-desktop-24-regular.svg?color=%23${REGULAR_COLOR}" -o "$OUTPUT_DIR/control.svg"
curl -L "https://api.iconify.design/fluent/phone-desktop-24-filled.svg?color=%23${FILLED_COLOR}" -o "$OUTPUT_DIR/control-active.svg"

# profile icons
curl -L "https://api.iconify.design/fluent/person-24-regular.svg?color=%23${REGULAR_COLOR}" -o "$OUTPUT_DIR/profile.svg"
curl -L "https://api.iconify.design/fluent/person-24-filled.svg?color=%23${FILLED_COLOR}" -o "$OUTPUT_DIR/profile-active.svg"

echo "Done! SVG files saved to $OUTPUT_DIR"
echo "Note: You need to convert SVG to PNG for tabBar usage."