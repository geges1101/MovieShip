#!/bin/bash

# Ждем пока Keycloak запустится
echo "Waiting for Keycloak to start..."
sleep 30

# Получаем токен администратора
ADMIN_TOKEN=$(curl -s -X POST http://localhost:8081/realms/master/protocol/openid-connect/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "username=admin" \
  -d "password=admin" \
  -d "grant_type=password" \
  -d "client_id=admin-cli" | jq -r '.access_token')

# Создаем realm
curl -s -X POST http://localhost:8081/admin/realms \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "realm": "movieship",
    "enabled": true,
    "sslRequired": "none",
    "registrationAllowed": true,
    "loginWithEmailAllowed": true,
    "duplicateEmailsAllowed": false,
    "resetPasswordAllowed": true,
    "editUsernameAllowed": true,
    "bruteForceDetectionEnabled": false
  }'

# Создаем клиент
curl -s -X POST http://localhost:8081/admin/realms/movieship/clients \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "clientId": "movieship-client",
    "enabled": true,
    "publicClient": false,
    "redirectUris": ["http://localhost:8080/*"],
    "webOrigins": ["http://localhost:8080"],
    "directAccessGrantsEnabled": true,
    "serviceAccountsEnabled": true,
    "authorizationServicesEnabled": true,
    "clientAuthenticatorType": "client-secret",
    "secret": "your-client-secret"
  }'

# Создаем пользователя
curl -s -X POST http://localhost:8081/admin/realms/movieship/users \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "enabled": true,
    "credentials": [{
      "type": "password",
      "value": "admin",
      "temporary": false
    }]
  }'

echo "Keycloak setup completed!" 