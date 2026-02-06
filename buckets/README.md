# Buckets Terraform

Cria o bucket de imagens com permissao de leitura publica e CORS.

## Pre-requisitos
- Terraform >= 1.3
- Credenciais AWS configuradas

## Como executar
```powershell
cd "c:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\buckets"
terraform init
terraform apply -var "aws_region=us-east-1" -var "bucket_name_images=imgs-docao-post-v1" -var "bucket_name_users_profile=doacao-users-prefil-v1"
```
