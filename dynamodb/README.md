## Terraform (DynamoDB core)

### Requisitos
- Terraform >= 1.3
- AWS CLI configurado (ou credenciais via env vars)

### Uso
```powershell
cd "c:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\dynamodb"
terraform init
terraform plan -var "aws_region=us-east-1"
terraform apply -var "aws_region=us-east-1"
```

### Variaveis
- `aws_region` (obrigatoria)
- `table_name` (default: core)
- `tags` (opcional)

### Exemplo com nome custom
```powershell
terraform apply -var "aws_region=us-east-1" -var "table_name=core"
```
