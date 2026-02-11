# Lambda donation-email-send

Worker de e-mail assíncrono:
- consome eventos da SQS `donation-email-events`
- envia e-mails pelo SES
- respeita limite diário (`199`)
- guarda pendentes na tabela `core`
- reprocessa pendentes diariamente às `07:00 America/Sao_Paulo` (`10:00 UTC`)

## Build
```powershell
cd "c:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\donation-email-send"
$env:GOOS="linux"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"; go build -o bootstrap .; Compress-Archive -Path bootstrap -DestinationPath lambda.zip -Force
```

## Deploy (Terraform)
1. Pegue o ARN da fila criada no módulo `donation`:
```powershell
cd "c:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\donation\terraform"
terraform output email_events_queue_arn
```

2. Aplique este módulo:
```powershell
cd "c:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\donation-email-send\terraform"
terraform init
terraform apply -var "aws_region=us-east-1" -var "lambda_zip=../lambda.zip" -var "queue_arn=arn:aws:sqs:us-east-1:727646486460:donation-email-events" -var "dynamodb_table=core" -var "ses_from_email=admin@thepuregrace.com" -var "app_base_url=https://www.thepuregrace.com" -var "daily_email_limit=199"
```

## Eventos aceitos da fila
- `email-validar-email-usuario`
- `email-cadastro-doacao`

## Itens gravados na tabela `core`
- Cota diária:
  - `PK=EMAIL#QUOTA#YYYY-MM-DD`, `SK=COUNTER`
- Pendentes:
  - `PK=EMAIL#PENDING`, `SK=TS#...`
- Token de validação:
  - `PK=EMAIL#VERIFY#{token}`, `SK=USER#{userId}`
