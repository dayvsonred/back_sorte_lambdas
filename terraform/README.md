# Terraform (Stack Completo)

Este Terraform cria:
- Roles e policies de IAM para Lambda
- Lambdas (`users`, `login`, `donation`, `pix`, `contact`)
- Um unico API Gateway HTTP (v2) unificado e integrado a todas as Lambdas

## Pre-requisitos
- Terraform >= 1.3
- Credenciais AWS configuradas (AWS CLI, variaveis de ambiente ou perfil)
- Zips das Lambdas disponiveis (veja `variables.tf`)

## Como executar
```powershell
cd "c:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\terraform"
terraform init
terraform apply -var "aws_region=us-east-1" -var "project_name=back-sorte" `
  -var "lambda_users_zip=C:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\users\lambda.zip" `
  -var "lambda_login_zip=C:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\login\lambda.zip" `
  -var "lambda_donation_zip=C:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\donation\lambda.zip" `
  -var "lambda_pix_zip=C:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\pix\lambda.zip" `
  -var "lambda_contact_zip=C:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\contact\lambda.zip"
```

## Observacoes
- O endpoint do gateway aparece no output `http_api_endpoint` em `outputs.tf`.
- Se o `project_name` for diferente, ajuste no `terraform apply`.


cd "c:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\gateway\terraform"
terraform init
terraform apply -var "aws_region=us-east-1" -var "project_name=back-sorte"


cd "c:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\terraform"
terraform apply -target=aws_apigatewayv2_api.http

cd "c:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\terraform"
terraform apply

cd "c:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\terraform"
terraform init
terraform apply -var "aws_region=us-east-1" -var "project_name=back-sorte" `
  -var "lambda_users_zip=C:\caminho\users.zip" `
  -var "lambda_login_zip=C:\caminho\login.zip" `
  -var "lambda_donation_zip=C:\caminho\donation.zip" `
  -var "lambda_pix_zip=C:\caminho\pix.zip" `
  -var "lambda_contact_zip=C:\caminho\contact.zip"
