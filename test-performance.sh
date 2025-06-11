#!/bin/bash

# Получаем токен пользователя
TOKEN=$(curl -s -X POST http://localhost:8081/realms/movieship/protocol/openid-connect/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "username=lgegesl" \
  -d "password=password" \
  -d "grant_type=password" \
  -d "client_id=movieship-client" \
  -d "client_secret=your-client-secret" | jq -r '.access_token')

if [ -z "$TOKEN" ]; then
    echo "Failed to get token"
    exit 1
fi

echo -e "\nTesting GET /api/movies/{id}/stream endpoint..."
ab -n 100 -c 5 -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/movies/1/stream

echo -e "\nTesting GET /api/movies/{id}/stream/{quality} endpoint..."
ab -n 100 -c 5 -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/movies/1/stream/720p

# Тест с разным количеством одновременных соединений
echo -e "\nTesting with different concurrency levels..."
for c in 1 5 10 20 50; do
  echo -e "\nTesting with $c concurrent connections..."
  ab -n 1000 -c $c -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/movies
done 