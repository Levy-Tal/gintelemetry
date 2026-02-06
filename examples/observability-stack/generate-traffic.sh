#!/bin/bash

echo "🔄 Generating traffic to the application..."
echo "Press Ctrl+C to stop"
echo ""

while true; do
  # Random ID
  ID=$RANDOM
  
  # Mix of different endpoints
  ENDPOINTS=(
    "/hello"
    "/process/$ID"
    "/slow"
    "/error"
  )
  
  # Pick a random endpoint
  ENDPOINT=${ENDPOINTS[$RANDOM % ${#ENDPOINTS[@]}]}
  
  # Make the request
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080$ENDPOINT)
  
  # Color output based on status
  if [ "$STATUS" -ge 200 ] && [ "$STATUS" -lt 300 ]; then
    echo "✅ $ENDPOINT -> $STATUS"
  elif [ "$STATUS" -ge 400 ]; then
    echo "❌ $ENDPOINT -> $STATUS"
  else
    echo "⚠️  $ENDPOINT -> $STATUS"
  fi
  
  # Random sleep between 0.5 and 2 seconds
  sleep $(awk -v min=0.5 -v max=2 'BEGIN{srand(); print min+rand()*(max-min)}')
done
