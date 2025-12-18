#!/bin/bash

# Load Test Script for Image Processor
# Usage: ./scripts/load-test.sh
# Environment variables:
#   API_URL       - API endpoint (default: http://localhost:8080)
#   TOTAL_JOBS    - Total number of jobs to submit (default: 50)
#   CONCURRENT    - Number of concurrent requests (default: 5)
#   HEAVY_MODE    - Enable heavy processing (default: false)
#   IMAGE_PATH    - Path to test image (default: downloads a sample image)

set -e

# Configuration
API_URL="${API_URL:-http://localhost:8080}"
TOTAL_JOBS="${TOTAL_JOBS:-50}"
CONCURRENT="${CONCURRENT:-5}"
HEAVY_MODE="${HEAVY_MODE:-false}"
# IMAGE_PATH can be set to use a custom image, e.g., "/Users/timkrebs/Desktop/Screenshot.png"
IMAGE_PATH="${IMAGE_PATH:-}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Image Processor Load Test${NC}"
echo "================================"
echo "API URL:     $API_URL"
echo "Total Jobs:  $TOTAL_JOBS"
echo "Concurrent:  $CONCURRENT"
echo "Heavy Mode:  $HEAVY_MODE"
echo "================================"

# Check if API is available
echo -e "\n${YELLOW}Checking API health...${NC}"
if ! curl -s "${API_URL}/api/v1/health" > /dev/null; then
    echo -e "${RED}Error: API is not reachable at ${API_URL}${NC}"
    exit 1
fi
echo -e "${GREEN}API is healthy${NC}"

# Create test image if not provided
if [ -z "$IMAGE_PATH" ]; then
    echo -e "\n${YELLOW}Creating test image...${NC}"
    IMAGE_PATH="/tmp/test_image_$$.jpg"

    # Try to download a sample image
    if command -v curl &> /dev/null; then
        curl -sL -o "$IMAGE_PATH" "https://picsum.photos/1920/1080" || {
            # If download fails, create a simple test image using ImageMagick if available
            if command -v convert &> /dev/null; then
                convert -size 1920x1080 xc:blue -fill white -draw "circle 960,540 960,100" "$IMAGE_PATH"
            else
                echo -e "${RED}Error: Could not create test image. Please provide IMAGE_PATH or install ImageMagick.${NC}"
                exit 1
            fi
        }
    fi
    
    # Verify image was created successfully
    if [ ! -s "$IMAGE_PATH" ]; then
        echo -e "${RED}Error: Failed to create test image (file is empty)${NC}"
        exit 1
    fi
    CLEANUP_IMAGE=true
else
    CLEANUP_IMAGE=false
fi

echo "Using image: $IMAGE_PATH"

# Define operations based on mode
if [ "$HEAVY_MODE" = "true" ]; then
    OPERATIONS='[
        {"operation":"resize","parameters":{"width":3840,"height":2160}},
        {"operation":"blur","parameters":{"sigma":10}},
        {"operation":"sharpen","parameters":{"sigma":2}},
        {"operation":"brightness","parameters":{"amount":20}},
        {"operation":"contrast","parameters":{"amount":15}},
        {"operation":"saturation","parameters":{"amount":10}},
        {"operation":"blur","parameters":{"sigma":5}},
        {"operation":"thumbnail","parameters":{"size":300}}
    ]'
else
    OPERATIONS='[
        {"operation":"resize","parameters":{"width":800}},
        {"operation":"thumbnail","parameters":{"size":150}}
    ]'
fi

# Function to submit a job
submit_job() {
    local job_num=$1
    local response
    response=$(curl -s -w "\n%{http_code}" -X POST "${API_URL}/api/v1/jobs" \
        -F "image=@${IMAGE_PATH}" \
        -F "operations=${OPERATIONS}")

    local http_code=$(echo "$response" | tail -n1)
    local body=$(echo "$response" | sed '$d')

    if [ "$http_code" = "201" ]; then
        local job_id=$(echo "$body" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
        echo "Job $job_num: Created (ID: $job_id)"
        return 0
    else
        echo "Job $job_num: Failed (HTTP $http_code)"
        return 1
    fi
}

# Track statistics
SUCCESS_COUNT=0
FAIL_COUNT=0
START_TIME=$(date +%s)

# Submit jobs with concurrency control
echo -e "\n${YELLOW}Submitting $TOTAL_JOBS jobs with concurrency $CONCURRENT...${NC}\n"

for ((i=1; i<=TOTAL_JOBS; i++)); do
    # Submit job in background
    (
        if submit_job $i; then
            echo "SUCCESS"
        else
            echo "FAIL"
        fi
    ) &

    # Control concurrency
    if [ $((i % CONCURRENT)) -eq 0 ]; then
        wait
    fi
done

# Wait for remaining jobs
wait

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

# Get final queue stats
echo -e "\n${YELLOW}Queue Statistics:${NC}"
curl -s "${API_URL}/api/v1/stats/queue" | python3 -m json.tool 2>/dev/null || \
    curl -s "${API_URL}/api/v1/stats/queue"

# Summary
echo -e "\n================================"
echo -e "${GREEN}Load Test Complete${NC}"
echo "Duration:     ${DURATION}s"
if [ $DURATION -gt 0 ]; then
    echo "Jobs/second:  $(echo "scale=2; $TOTAL_JOBS / $DURATION" | bc)"
else
    echo "Jobs/second:  N/A (completed in <1s)"
fi
echo "================================"

# Cleanup
if [ "$CLEANUP_IMAGE" = "true" ] && [ -f "$IMAGE_PATH" ]; then
    rm -f "$IMAGE_PATH"
fi

echo -e "\n${YELLOW}Tip: Watch worker scaling with:${NC}"
echo "  kubectl get pods -n image-processor -w"
echo "  or"
echo "  docker compose ps"
