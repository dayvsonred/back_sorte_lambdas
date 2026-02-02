# Lambda donation

## Build
```powershell
cd "c:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\donation"
$env:GOOS="linux"
$env:GOARCH="amd64"
$env:CGO_ENABLED="0"
go build -o bootstrap .
Compress-Archive -Path bootstrap -DestinationPath lambda.zip -Force
```

## Deploy (Terraform)
```powershell
cd "c:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\donation\terraform"
terraform init
terraform apply -var "aws_region=us-east-1" -var "dynamodb_table=core" -var "lambda_zip=../lambda.zip"
```

## Exemplo de uso (requests)
```bash
# API Gateway (HTTP API)
BASE_URL="https://obx90nm3e5.execute-api.us-east-1.amazonaws.com"
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

Observacao: as rotas `POST /donation` e `POST /donation/createUserAndDonation` usam `multipart/form-data`. Para chamar direto a Lambda, o corpo deve ir em `body` base64 e com `content-type` contendo o boundary.
