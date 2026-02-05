# Lambda pix

## Build
```powershell
cd "c:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\pix"
$env:GOOS="linux"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"; go build -o bootstrap .; Compress-Archive -Path bootstrap -DestinationPath lambda.zip -Force
```

## Deploy (Terraform)
```powershell
cd "c:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\pix\terraform"
terraform init
terraform apply -var "aws_region=us-east-1" -var "dynamodb_table=core" -var "lambda_zip=../lambda.zip"
```

## Exemplo de uso (requests)
```bash
# Criar cobranca PIX
curl -X POST "$BASE_URL/pix/create" \
  -H "Content-Type: application/json" \
  -d '{"valor":"10.00","cpf":"12345678900","nome":"Joao","chave":"SUA_CHAVE","mensagem":"Obrigado","anonimo":false,"id":"DONATION_ID"}'

# Consultar status
curl "$BASE_URL/pix/status/TXID"
```
