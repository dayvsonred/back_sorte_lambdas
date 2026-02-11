# Lambda donation

## Build
```powershell
cd "c:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\donation"
$env:GOOS="linux"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"; go build -o bootstrap .; Compress-Archive -Path bootstrap -DestinationPath lambda.zip -Force
```

## Deploy (Terraform)
```powershell
cd "c:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\donation\terraform"
terraform init
terraform apply -var "aws_region=us-east-1" -var "dynamodb_table=core" -var "lambda_zip=../lambda.zip" -var "aws_bucket_name_img_doacao=imgs-docao-post-v1" -var "email_events_queue_name=donation-email-events" -var "app_base_url=https://www.thepuregrace.com"
```

## Email assíncrono (SQS)
- Este módulo cria a fila SQS `donation-email-events`.
- As rotas `POST /donation` e `POST /donation/createUserAndDonation` publicam eventos de e-mail nessa fila.
- A lambda `donation-email-send` (módulo separado) consome a fila e envia os e-mails via SES.

### Outputs úteis
```powershell
terraform output email_events_queue_arn
terraform output email_events_queue_url
```

## Exemplo de uso (requests)
```bash
# API Gateway (HTTP API)
# Caso seu endpoint tenha base path "/donation", use:
BASE_URL="https://obx90nm3e5.execute-api.us-east-1.amazonaws.com/donation"
# Se o endpoint nao tiver base path, use:
# BASE_URL="https://obx90nm3e5.execute-api.us-east-1.amazonaws.com"
TOKEN="SEU_JWT_AQUI"

# Criar doacao (multipart)
curl -X POST "$BASE_URL/donation" \
  -H "Authorization: Bearer $TOKEN" \
  -F "name=Minha campanha" \
  -F "valor=100" \
  -F "texto=Texto da doacao" \
  -F "area=Saude" \
  -F "image=@./foto.jpg"

# Listar doacoes por usuario
curl "$BASE_URL/donation/list?id_user=USER_ID&page=1&limit=10"

# Buscar doacao por link (nome_link precisa iniciar com @)
curl "$BASE_URL/donation/link/@minha-campanha"

# Mensagens da doacao
curl "$BASE_URL/donation/mensagem?id=DONATION_ID&page=1&limit=10"

# Encerrar doacao (precisa ser o dono)
curl -H "Authorization: Bearer $TOKEN" \
  "$BASE_URL/donation/closed/DONATION_ID"

# Solicitar resgate (precisa ser o dono)
curl -H "Authorization: Bearer $TOKEN" \
  "$BASE_URL/donation/rescue/DONATION_ID"

# Registrar visualizacao
curl -X POST "$BASE_URL/donation/visualization" \
  -H "Content-Type: application/json" \
  -d '{"id_doacao":"DONATION_ID","id_user":"USER_ID","visuaization":true}'

# Criar usuario + doacao (multipart)
curl -X POST "$BASE_URL/donation/createUserAndDonation" \
  -F "fullName=Nome Completo" \
  -F "cpf=12345678901" \
  -F "email=teste@email.com" \
  -F "senha=123456" \
  -F "titulo=Minha campanha" \
  -F "meta=150.00" \
  -F "categoria=Saude" \
  -F "texto=Texto da doacao" \
  -F "image=@./foto.jpg"

# Deletar doacao (soft delete, precisa ser o dono)
curl -X DELETE "$BASE_URL/donation/DONATION_ID" \
  -H "Authorization: Bearer $TOKEN"
```

## Exemplo de evento JSON (chamando a Lambda direto)
Use no teste da AWS Console (Test event) ou via CLI com o payload abaixo.

### POST /donation (multipart)
```json
{
  "version": "2.0",
  "routeKey": "POST /donation",
  "rawPath": "/donation",
  "headers": {
    "content-type": "multipart/form-data; boundary=----boundary",
    "authorization": "Bearer SEU_JWT_AQUI"
  },
  "requestContext": {
    "http": {
      "method": "POST",
      "path": "/donation",
      "sourceIp": "127.0.0.1",
      "userAgent": "aws-lambda-test"
    }
  },
  "body": "LS0tLS1ib3VuZGFyeQ0KQ29udGVudC1EaXNwb3NpdGlvbjogZm9ybS1kYXRhOyBuYW1lPSJuYW1lIg0KDQpNaW5oYSBjYW1wYW5oYQ0KLS0tLS1ib3VuZGFyeQ0KQ29udGVudC1EaXNwb3NpdGlvbjogZm9ybS1kYXRhOyBuYW1lPSJ2YWxvciINCg0KMTAwDQotLS0tLWJvdW5kYXJ5DQpDb250ZW50LURpc3Bvc2l0aW9uOiBmb3JtLWRhdGE7IG5hbWU9InRleHRvIg0KDQpUZXh0byBkYSBkb2FjYW8NCi0tLS0tYm91bmRhcnkNCkNvbnRlbnQtRGlzcG9zaXRpb246IGZvcm0tZGF0YTsgbmFtZT0iYXJlYSINCg0KU2F1ZGUNCi0tLS0tYm91bmRhcnkNCkNvbnRlbnQtRGlzcG9zaXRpb246IGZvcm0tZGF0YTsgbmFtZT0iaW1hZ2UiOyBmaWxlbmFtZT0iZm90by5qcGciDQpDb250ZW50LVR5cGU6IGltYWdlL2pwZWcNCg0KLi4uQklOQVJZREFUQS4uLg0KLS0tLS1ib3VuZGFyeS0t",
  "isBase64Encoded": true
}
```

### GET /donation/list
```json
{
  "version": "2.0",
  "routeKey": "GET /donation/list",
  "rawPath": "/donation/list",
  "rawQueryString": "id_user=USER_ID&page=1&limit=10",
  "headers": {
    "host": "obx90nm3e5.execute-api.us-east-1.amazonaws.com",
    "user-agent": "curl/8.0"
  },
  "requestContext": {
    "http": {
      "method": "GET",
      "path": "/donation/list",
      "sourceIp": "127.0.0.1",
      "userAgent": "curl/8.0"
    }
  },
  "isBase64Encoded": false
}
```

### DELETE /donation/{id}
```json
{
  "version": "2.0",
  "routeKey": "DELETE /donation/{id}",
  "rawPath": "/donation/DONATION_ID",
  "pathParameters": {
    "id": "DONATION_ID"
  },
  "headers": {
    "authorization": "Bearer SEU_JWT_AQUI"
  },
  "requestContext": {
    "http": {
      "method": "DELETE",
      "path": "/donation/DONATION_ID",
      "sourceIp": "127.0.0.1",
      "userAgent": "aws-lambda-test"
    }
  },
  "isBase64Encoded": false
}
```

### GET /donation/link/{nome_link}
```json
{
  "version": "2.0",
  "routeKey": "GET /donation/link/{nome_link}",
  "rawPath": "/donation/link/@minha-campanha",
  "pathParameters": {
    "nome_link": "@minha-campanha"
  },
  "requestContext": {
    "http": {
      "method": "GET",
      "path": "/donation/link/@minha-campanha",
      "sourceIp": "127.0.0.1",
      "userAgent": "aws-lambda-test"
    }
  },
  "isBase64Encoded": false
}
```

### GET /donation/mensagem
```json
{
  "version": "2.0",
  "routeKey": "GET /donation/mensagem",
  "rawPath": "/donation/mensagem",
  "rawQueryString": "id=DONATION_ID&page=1&limit=10",
  "requestContext": {
    "http": {
      "method": "GET",
      "path": "/donation/mensagem",
      "sourceIp": "127.0.0.1",
      "userAgent": "aws-lambda-test"
    }
  },
  "isBase64Encoded": false
}
```

### GET /donation/closed/{id}
```json
{
  "version": "2.0",
  "routeKey": "GET /donation/closed/{id}",
  "rawPath": "/donation/closed/DONATION_ID",
  "pathParameters": {
    "id": "DONATION_ID"
  },
  "headers": {
    "authorization": "Bearer SEU_JWT_AQUI"
  },
  "requestContext": {
    "http": {
      "method": "GET",
      "path": "/donation/closed/DONATION_ID",
      "sourceIp": "127.0.0.1",
      "userAgent": "aws-lambda-test"
    }
  },
  "isBase64Encoded": false
}
```

### GET /donation/rescue/{id}
```json
{
  "version": "2.0",
  "routeKey": "GET /donation/rescue/{id}",
  "rawPath": "/donation/rescue/DONATION_ID",
  "pathParameters": {
    "id": "DONATION_ID"
  },
  "headers": {
    "authorization": "Bearer SEU_JWT_AQUI"
  },
  "requestContext": {
    "http": {
      "method": "GET",
      "path": "/donation/rescue/DONATION_ID",
      "sourceIp": "127.0.0.1",
      "userAgent": "aws-lambda-test"
    }
  },
  "isBase64Encoded": false
}
```

### POST /donation/visualization
```json
{
  "version": "2.0",
  "routeKey": "POST /donation/visualization",
  "rawPath": "/donation/visualization",
  "headers": {
    "content-type": "application/json"
  },
  "requestContext": {
    "http": {
      "method": "POST",
      "path": "/donation/visualization",
      "sourceIp": "127.0.0.1",
      "userAgent": "aws-lambda-test"
    }
  },
  "body": "{\"id_doacao\":\"DONATION_ID\",\"id_user\":\"USER_ID\",\"visuaization\":true,\"donation_like\":false,\"love\":false,\"shared\":false}",
  "isBase64Encoded": false
}
```

### POST /donation/createUserAndDonation (multipart)
```json
{
  "version": "2.0",
  "routeKey": "POST /donation/createUserAndDonation",
  "rawPath": "/donation/createUserAndDonation",
  "headers": {
    "content-type": "multipart/form-data; boundary=----boundary"
  },
  "requestContext": {
    "http": {
      "method": "POST",
      "path": "/donation/createUserAndDonation",
      "sourceIp": "127.0.0.1",
      "userAgent": "aws-lambda-test"
    }
  },
  "body": "LS0tLS1ib3VuZGFyeQ0KQ29udGVudC1EaXNwb3NpdGlvbjogZm9ybS1kYXRhOyBuYW1lPSJmdWxsTmFtZSINCg0KTm9tZSBDb21wbGV0bw0KLS0tLS1ib3VuZGFyeQ0KQ29udGVudC1EaXNwb3NpdGlvbjogZm9ybS1kYXRhOyBuYW1lPSJjcGYiDQoNCjEyMzQ1Njc4OTAxDQotLS0tLWJvdW5kYXJ5DQpDb250ZW50LURpc3Bvc2l0aW9uOiBmb3JtLWRhdGE7IG5hbWU9ImVtYWlsIg0KDQp0ZXN0ZUBtYWlsLmNvbQ0KLS0tLS1ib3VuZGFyeQ0KQ29udGVudC1EaXNwb3NpdGlvbjogZm9ybS1kYXRhOyBuYW1lPSJzZW5oYSINCg0KMTIzNDU2DQotLS0tLWJvdW5kYXJ5DQpDb250ZW50LURpc3Bvc2l0aW9uOiBmb3JtLWRhdGE7IG5hbWU9InRpdHVsbyINCg0KTWluaGEgY2FtcGFuaGENCi0tLS0tYm91bmRhcnkNCkNvbnRlbnQtRGlzcG9zaXRpb246IGZvcm0tZGF0YTsgbmFtZT0ibWV0YSINCg0KMTUwLjAwDQotLS0tLWJvdW5kYXJ5DQpDb250ZW50LURpc3Bvc2l0aW9uOiBmb3JtLWRhdGE7IG5hbWU9ImNhdGVnb3JpYSINCg0KU2F1ZGUNCi0tLS0tYm91bmRhcnkNCkNvbnRlbnQtRGlzcG9zaXRpb246IGZvcm0tZGF0YTsgbmFtZT0idGV4dG8iDQoNClRleHRvIGRhIGRvYWNhbwpbYmFzZTY0IGltYWdlIGVtIGNhbXBvIGltYWdlXQ0KLS0tLS1ib3VuZGFyeS0t",
  "isBase64Encoded": true
}
```

Observacao: as rotas `POST /donation` e `POST /donation/createUserAndDonation` usam `multipart/form-data`. Para chamar direto a Lambda, o corpo deve ir em `body` base64 e com `content-type` contendo o boundary.
