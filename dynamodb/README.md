## Terraform (DynamoDB core + export diário para S3)

### Requisitos
- Terraform >= 1.5
- AWS CLI configurado (ou credenciais via env vars)

### O que este Terraform cria
- Bucket S3 `bd-thepuregrace-v1-dinamodb-core` com:
  - SSE-S3 (AES256), sem versioning, block public access, ownership controls, lifecycle (expira em 30 dias)
- Lambda (Python 3.12) que dispara `ExportTableToPointInTime`
- EventBridge Rule diário (03:10 America/Sao_Paulo -> 06:10 UTC)
- Políticas mínimas para DynamoDB gravar no bucket e para a Lambda

### Observações importantes
- O export via `ExportTableToPointInTime` requer **PITR habilitado** na tabela. Este Terraform habilita por padrão (`enable_pitr = true`).
- Custo principal: PITR + armazenamento S3. O lifecycle expira objetos antigos para reduzir custo.
- A tabela `core` já existe e continua gerenciada pelo Terraform atual. O `apply` **não apaga dados**; evite `terraform destroy` se não quiser remover recursos.

### Como subir após as modificações
```powershell
cd "c:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\dynamodb"
terraform init
terraform plan -var "aws_region=us-east-1"
terraform apply -var "aws_region=us-east-1"
```

### Variaveis
- `aws_region` (default: us-east-1)
- `table_name` (default: core)
- `s3_bucket_name` (default: bd-thepuregrace-v1-dinamodb-core)
- `export_prefix_base` (default: exports/core)
- `export_retention_days` (default: 30)
- `export_format` (default: AMAZON_ION)
- `schedule_expression` (default: cron(10 6 * * ? *) -> 03:10 America/Sao_Paulo)
- `enable_pitr` (default: true)
- `tags` (opcional)

### Exemplo com variaveis custom
```powershell
terraform plan -var "aws_region=us-east-1" -var "table_name=core" -var "export_retention_days=45"
terraform apply -var "aws_region=us-east-1" -var "table_name=core" -var "export_retention_days=45"
```

### Como verificar se o export rodou
- CloudWatch Logs da Lambda `dynamodb-export-core`
- S3: prefixos em `exports/core/YYYY-MM-DD/`

### Destruir com seguranca
```powershell
terraform destroy -var "aws_region=us-east-1"
```
