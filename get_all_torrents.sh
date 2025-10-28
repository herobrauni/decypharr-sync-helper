#!/bin/bash

# Configuration - Update these values to match your qBittorrent setup
QB_BASE_URL="http://localhost:8282"
QB_USERNAME="admin"
QB_PASSWORD="adminadmin"

# Create a temporary file to store cookies
COOKIE_FILE=$(mktemp)

# Function to cleanup on exit
cleanup() {
    rm -f "$COOKIE_FILE"
}
trap cleanup EXIT

echo "Logging into qBittorrent..."

# Login to get session cookie
LOGIN_RESPONSE=$(curl -s -c "$COOKIE_FILE" \
    -H "Content-Type: application/x-www-form-urlencoded" \
    -H "Referer: $QB_BASE_URL" \
    -H "Origin: $QB_BASE_URL" \
    -d "username=$QB_USERNAME&password=$QB_PASSWORD" \
    -X POST "$QB_BASE_URL/api/v2/auth/login")

# Check if login was successful
if [ "$LOGIN_RESPONSE" != "Ok." ]; then
    echo "Login failed: $LOGIN_RESPONSE"
    exit 1
fi

echo "Login successful. Retrieving all torrents..."

# Get all torrents with all available information
curl -s -b "$COOKIE_FILE" \
    -H "Referer: $QB_BASE_URL" \
    -H "Origin: $QB_BASE_URL" \
    "$QB_BASE_URL/api/v2/torrents/info" | jq .

echo ""
echo "To get torrents with specific filters, you can modify the URL with parameters:"
echo "  - Filter by state: ?filter=completed|downloading|paused|active|inactive|resumed"
echo "  - Filter by category: ?category=your_category"
echo "  - Filter by tag: ?tag=your_tag"
echo "  - Sort results: ?sort=hash&reverse=true"
echo ""
echo "Example for completed torrents in a specific category:"
echo "curl -b \"$COOKIE_FILE\" -H \"Referer: $QB_BASE_URL\" \"$QB_BASE_URL/api/v2/torrents/info?filter=completed&category=movies\" | jq ."