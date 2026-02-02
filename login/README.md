# Lambda login

## Build
```powershell
cd "c:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\login"
$env:GOOS="linux"
$env:GOARCH="amd64"
$env:CGO_ENABLED="0"
go build -o bootstrap .
Compress-Archive -Path bootstrap -DestinationPath lambda.zip -Force
```
```powershell
$env:GOOS="linux"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"; go build -o bootstrap .; Compress-Archive -Path bootstrap -DestinationPath lambda.zip -Force
```

## Deploy (Terraform)
```powershell
cd "c:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\login\terraform"
terraform init
terraform apply -var "aws_region=us-east-1" -var "dynamodb_table=core" -var "lambda_zip=../lambda.zip"
```
```powershell
cd "c:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\login\terraform"
terraform apply -var "aws_region=us-east-1" -var "dynamodb_table=core" -var "lambda_zip=../lambda.zip" -var "jwt_secret=XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"

```

## Exemplo de uso (requests)
```bash
curl -X POST "$BASE_URL/login" \
  -H "Authorization: Basic QVBJX05BTUVfQUNDRVNTOkFQSV9TRUNSRVRfQUNDRVNT" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=password&username=joao@email.com&password=123456"
```
