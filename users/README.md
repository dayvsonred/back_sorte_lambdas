# Lambda users

## Build
```powershell
cd "c:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\users"
$env:GOOS="linux"
$env:GOARCH="amd64"
$env:CGO_ENABLED="0"
go build -o bootstrap .
Compress-Archive -Path bootstrap -DestinationPath lambda.zip -Force
```

## Deploy (Terraform)
```powershell
cd "c:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\users\terraform"
terraform init
terraform apply -var "aws_region=us-east-1" -var "dynamodb_table=core" -var "lambda_zip=../lambda.zip"
```

## Exemplo de uso (requests)
```bash
# Criar usuario
curl -X POST "$BASE_URL/users" \
  -H "Content-Type: application/json" \
  -d '{"name":"Joao","email":"joao@email.com","password":"123456","cpf":"12345678900"}'

# Alterar senha (JWT)
curl -X POST "$BASE_URL/users/passwordChange" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"old_password":"123456","new_password":"654321"}'

# Buscar imagem de perfil
curl "$BASE_URL/users/ProfileImage/USER_ID"
```


## Exemplo de uso (requests)
```bash

curl -X POST "https://bw3zzn1l2d.execute-api.us-east-1.amazonaws.com/users" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Lucas",
    "email": "lucas_dell01@gmail.com",
    "password": "123456",
    "cpf": "12345678900"
  }'



  {
  "version": "2.0",
  "routeKey": "POST /users",
  "rawPath": "/users",
  "rawQueryString": "",
  "headers": {
    "content-type": "application/json"
  },
  "requestContext": {
    "http": {
      "method": "POST",
      "path": "/users"
    }
  },
  "isBase64Encoded": false,
  "body": "{\"name\":\"Lucas\",\"email\":\"lucas_dell01@gmail.com\",\"password\":\"123456\",\"cpf\":\"12345678900\"}"
}

```