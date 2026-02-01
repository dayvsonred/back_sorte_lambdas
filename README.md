## Lambdas por dominio

Pastas:
- users
- login
- donation
- pix
- contact

### Build automatico (gera os ZIPs)
```powershell
cd "c:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go"
powershell -ExecutionPolicy Bypass -File back_sorte_lambdas\build_all.ps1
```

### Deploy (Terraform por dominio)
Cada dominio tem seu proprio Terraform em `back_sorte_lambdas/<dominio>/terraform`.

Exemplo (users):
```powershell
cd "c:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\users\terraform"
terraform init
terraform apply -var "aws_region=us-east-1" -var "dynamodb_table=core" -var "lambda_zip=../lambda.zip"
```
