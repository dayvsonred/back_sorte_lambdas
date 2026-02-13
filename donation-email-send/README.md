# Lambda `donation-email-send`

Worker de e-mail assíncrono:
- consome eventos da SQS `donation-email-events`
- envia e-mails por provider configurável (`ses` ou `brevo`)
- respeita limite diário (`DAILY_EMAIL_LIMIT`, default `199`)
- guarda pendentes na tabela DynamoDB
- reprocessa pendentes diariamente às `07:00 America/Sao_Paulo` (`10:00 UTC`)

## Variáveis de ambiente

Obrigatórias:
- `DYNAMODB_TABLE`
- `SES_FROM_EMAIL` (e-mail remetente)
- `APP_BASE_URL`
- `AWS_REGION`

Para provider:
- `EMAIL_PROVIDER=ses` (padrão) ou `EMAIL_PROVIDER=brevo`
- `BREVO_API_KEY` (obrigatória quando `EMAIL_PROVIDER=brevo`)
- `EMAIL_FROM_NAME` (opcional, padrão `The Pure Grace`)

Limite diário:
- `DAILY_EMAIL_LIMIT` (padrão `199`; para 200, configure `200`)

Exemplo (Brevo):

```env
EMAIL_PROVIDER=brevo
BREVO_API_KEY=xxxxx
SES_FROM_EMAIL=contato@seudominio.com
EMAIL_FROM_NAME=The Pure Grace
DAILY_EMAIL_LIMIT=200
APP_BASE_URL=https://www.thepuregrace.com
DYNAMODB_TABLE=core
AWS_REGION=us-east-1
```

## Sobre `BREVO_API_KEY` na compilação

Hoje a lambda lê `BREVO_API_KEY` de variável de ambiente em runtime.
Se quiser embutir no binário via `go build -ldflags -X`, é possível, mas não é recomendado para segredo em produção.
Em produção, prefira variável de ambiente, AWS Secrets Manager ou SSM Parameter Store.

## Build

```powershell
cd "C:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\donation-email-send"
$env:GOOS="linux"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"; go build -o bootstrap .; Compress-Archive -Path bootstrap -DestinationPath lambda.zip -Force
```

## Deploy (Terraform)

1. Pegue o ARN da fila criada no módulo `donation`:

```powershell
cd "C:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\donation\terraform"
terraform output email_events_queue_arn
```

2. Aplique este módulo:

```powershell
cd "C:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\donation-email-send\terraform"
terraform init
terraform apply `
  -var "aws_region=us-east-1" `
  -var "lambda_zip=../lambda.zip" `
  -var "queue_arn=arn:aws:sqs:us-east-1:727646486460:donation-email-events" `
  -var "dynamodb_table=core" `
  -var "ses_from_email=admin@thepuregrace.com" `
  -var "app_base_url=https://www.thepuregrace.com" `
  -var "daily_email_limit=200" `
  -var "email_provider=brevo" `
  -var "brevo_api_key=xxxx" `
  -var "email_from_name=The Pure Grace"
```

## Eventos aceitos da fila

- `email-validar-email-usuario`
- `email-cadastro-doacao`

## Itens gravados na tabela `core`

- Cota diária: `PK=EMAIL#QUOTA#YYYY-MM-DD`, `SK=COUNTER`
- Pendentes: `PK=EMAIL#PENDING`, `SK=TS#...`
- Token de validação: `PK=EMAIL#VERIFY#{token}`, `SK=USER#{userId}`
