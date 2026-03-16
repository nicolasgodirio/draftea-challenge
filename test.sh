#!/bin/bash

BASE_URL="http://localhost:8080/2015-03-31/functions/function/invocations"
USER_ID="b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"
NONEXISTENT_USER="00000000-0000-0000-0000-000000000000"

echo "=== Get Balance (valid user) ==="
curl -s -XPOST "$BASE_URL" \
  -d "{
    \"httpMethod\": \"GET\",
    \"path\": \"/wallets/$USER_ID/balance\",
    \"headers\": {
      \"Authorization\": \"Bearer test-token\"
    }
  }" | python3 -m json.tool

echo ""
echo "=== Get Transactions (valid user) ==="
curl -s -XPOST "$BASE_URL" \
  -d "{
    \"httpMethod\": \"GET\",
    \"path\": \"/wallets/$USER_ID/transactions\",
    \"headers\": {
      \"Authorization\": \"Bearer test-token\"
    }
  }" | python3 -m json.tool

echo ""
echo "=== Get Balance (nonexistent user) ==="
curl -s -XPOST "$BASE_URL" \
  -d "{
    \"httpMethod\": \"GET\",
    \"path\": \"/wallets/$NONEXISTENT_USER/balance\",
    \"headers\": {
      \"Authorization\": \"Bearer test-token\"
    }
  }" | python3 -m json.tool

echo ""
echo "=== Get Balance (no auth token) ==="
curl -s -XPOST "$BASE_URL" \
  -d "{
    \"httpMethod\": \"GET\",
    \"path\": \"/wallets/$USER_ID/balance\",
    \"headers\": {}
  }" | python3 -m json.tool
