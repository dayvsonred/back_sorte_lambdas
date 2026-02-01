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
```
