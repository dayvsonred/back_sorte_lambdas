# Gateway Terraform

Este Terraform cria **um unico API Gateway HTTP (v2)** (gateway unificado) e integra com as Lambdas ja existentes nas pastas `users`, `login`, `donation`, `pix`, `contact`.

## Pr�-requisitos
- Terraform >= 1.3
- Credenciais AWS configuradas (AWS CLI, vari�veis de ambiente ou perfil)
- Lambdas j� criadas com os nomes `${project_name}-users`, `${project_name}-login`, `${project_name}-donation`, `${project_name}-pix`, `${project_name}-contact`

## Como rodar
```powershell
cd "c:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\gateway\terraform"
terraform init
terraform apply -var "aws_region=us-east-1" -var "project_name=back-sorte"
```

Ao final, o endpoint do gateway ser� exibido no output `http_api_endpoint`.

## Observa��es
- Esse gateway **n�o cria Lambdas**; ele apenas integra as existentes.
- Se o `project_name` for diferente, ajuste o valor no `terraform apply`.

