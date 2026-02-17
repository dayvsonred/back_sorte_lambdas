# Lambda admin-thepuregrace

## Build
```powershell
cd "c:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\admin"
$env:GOOS="linux"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"; go build -o bootstrap .; Compress-Archive -Path bootstrap -DestinationPath lambda.zip -Force
```

## Deploy (Terraform)
```powershell
cd "c:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\admin\terraform"
terraform init
terraform apply -var "aws_region=us-east-1" -var "dynamodb_table=core" -var "lambda_zip=../lambda.zip" -var "admin_auth_key=XXXXXXXXXXXXXXXXXXX"
```

## Authorization
Todas as rotas exigem header `Authorization` com o valor da variavel `ADMIN_AUTH_KEY`.
A lambda tambem aceita `Authorization: Bearer <ADMIN_AUTH_KEY>`.

## Rotas iniciais
- `DELETE /admin/user/{id}`: remove dados do usuario e as doacoes vinculadas.
- `DELETE /admin/donation/{id}`: remove dados da doacao.
- `POST /admin/user/{id}/block`: seta `blocked=true` no usuario.

## Exemplo de uso
```bash
ADMIN_KEY="XXXXXXXX"
BASE_URL="https://SEU_API.execute-api.us-east-1.amazonaws.com"

curl -X DELETE "$BASE_URL/admin/user/USER_ID" \
  -H "Authorization: $ADMIN_KEY"

curl -X DELETE "$BASE_URL/admin/donation/DONATION_ID" \
  -H "Authorization: $ADMIN_KEY"

curl -X POST "$BASE_URL/admin/user/USER_ID/block" \
  -H "Authorization: $ADMIN_KEY"
```
