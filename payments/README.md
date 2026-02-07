# Payments Lambda (ThePureGrace)

Modulo de pagamentos usando AWS Lambda (Go), API Gateway HTTP API e DynamoDB (tabela `core`). Um unico handler roteia internamente as rotas HTTP e tambem recebe eventos da Stripe via EventBridge:

- `POST /payments/donations`
- `POST /payments/intents`

Base URL atual:
`https://rm0t2sapef.execute-api.us-east-1.amazonaws.com`

## Pre-requisitos
- Go 1.22
- AWS CLI configurado
- Terraform >= 1.3
- Conta Stripe com Event Destinations habilitado (EventBridge)

## Build e pacote da Lambda
```powershell
cd "C:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\payments"
$env:GOOS="linux"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"; go build -o bootstrap ./cmd/payments_handler
Compress-Archive -Path bootstrap -DestinationPath lambda.zip -Force
```
## Deploy (Terraform)
```powershell
cd "C:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\payments\terraform"
terraform init

terraform plan -var "aws_region=us-east-1" -var "api_id=rm0t2sapef" -var "stage_name=$default" -var "lambda_zip=../lambda.zip" -var "stripe_secret_key=sk_live_" -var "event_source_name=aws.partner/stripe.com/ed_61U7OMm8X8a9j7pji16U6BTHIq9PLmkSEfvbhl6fgPIG" -var "env=prod"

terraform apply -var "aws_region=us-east-1" -var "api_id=rm0t2sapef" -var "stage_name=$default" -var "lambda_zip=../lambda.zip" -var "stripe_secret_key=sk_live_" -var "event_source_name=aws.partner/stripe.com/ed_61U7OMm8X8a9j7pji16U6BTHIq9PLmkSEfvbhl6fgPIG" -var "env=prod"
```


Ordem recomendada para subir com EventBridge:
1) Stripe Dashboard: crie o Event Destination (Amazon EventBridge).
2) Copie o `event source name` (formato `aws.partner/stripe.com/...`).
3) AWS Console: EventBridge > Partner event sources > associe/aceite o source para criar o Event Bus.
4) Execute o Terraform acima com `event_source_name` apontando para o source da Stripe.

## Stripe EventBridge (configuracao)
1) No Stripe Dashboard, crie um Event Destination do tipo Amazon EventBridge.
2) Copie o `event source name` gerado pela Stripe (formato `aws.partner/stripe.com/...`).
3) No AWS, associe o Partner Event Source para criar o Event Bus (EventBridge > Partner event sources).
4) Use o `event source name` no Terraform (`event_source_name`).

Observacao: o EventBridge encapsula o evento Stripe dentro de `detail`, e o `detail-type` e o `type` do Stripe normalmente ficam iguais (ex.: `payment_intent.succeeded`).

Eventos recomendados no Stripe:
- `payment_intent.succeeded`
- `payment_intent.payment_failed`

## Logs do EventBridge
O Terraform cria um Log Group em CloudWatch para registrar todos os eventos recebidos da Stripe:
`/aws/eventbridge/thepuregrace-stripe-events`
Isso ajuda a debugar o payload exato recebido pelo EventBridge.

## Exemplos de requests (curl)
Criar doacao:
```bash
curl -X POST "https://rm0t2sapef.execute-api.us-east-1.amazonaws.com/payments/donations" \
  -H "Content-Type: application/json" \
  -d '{
    "campaignId": "campanha-2026",
    "amount": "10.50",
    "currency": "BRL",
    "donor": { "name": "Joao", "email": "joao@email.com" }
  }'
```

Criar PaymentIntent:
```bash
curl -X POST "https://rm0t2sapef.execute-api.us-east-1.amazonaws.com/payments/intents" \
  -H "Content-Type: application/json" \
  -d '{ "donationId": "<DONATION_ID>" }'
```

## Modelos de payload (JSON)
Criar doacao:
```json
{
  "campaignId": "campanha-2026",
  "amount": "10.50",
  "currency": "BRL",
  "donor": {
    "name": "Joao",
    "email": "joao@email.com"
  }
}
```

Criar PaymentIntent:
```json
{
  "donationId": "b0a2b0c4-0c6a-4e6a-9b30-8b3f2a3d6a8b"
}
```

Evento Stripe (detalhe enviado via EventBridge):
```json
{
  "id": "evt_123456789",
  "type": "payment_intent.succeeded",
  "created": 1738860000,
  "data": {
    "object": {
      "id": "pi_123456789",
      "object": "payment_intent",
      "metadata": {
        "donationId": "b0a2b0c4-0c6a-4e6a-9b30-8b3f2a3d6a8b",
        "campaignId": "campanha-2026"
      }
    }
  }
}
```

## Eventos de teste (API Gateway v2)
### POST /payments/donations
```json
{
  "version": "2.0",
  "routeKey": "POST /payments/donations",
  "rawPath": "/payments/donations",
  "rawQueryString": "",
  "headers": {
    "content-type": "application/json"
  },
  "requestContext": {
    "http": {
      "method": "POST",
      "path": "/payments/donations"
    }
  },
  "isBase64Encoded": false,
  "body": "{\n  \"campaignId\": \"campanha-2026\",\n  \"amount\": \"10.50\",\n  \"currency\": \"BRL\",\n  \"donor\": { \"name\": \"Joao\", \"email\": \"joao@email.com\" }\n}"
}
```

### POST /payments/intents
```json
{
  "version": "2.0",
  "routeKey": "POST /payments/intents",
  "rawPath": "/payments/intents",
  "rawQueryString": "",
  "headers": {
    "content-type": "application/json"
  },
  "requestContext": {
    "http": {
      "method": "POST",
      "path": "/payments/intents"
    }
  },
  "isBase64Encoded": false,
  "body": "{\n  \"donationId\": \"b0a2b0c4-0c6a-4e6a-9b30-8b3f2a3d6a8b\"\n}"
}
```

## Evento de teste (EventBridge)
```json
{
  "id": "0d87dfad-4c38-7c8c-aac4-7d8a10b8e3d0",
  "detail-type": "payment_intent.succeeded",
  "source": "aws.partner/stripe.com/ACCOUNT_ID/...",
  "account": "123456789012",
  "time": "2026-02-07T18:30:00Z",
  "region": "us-east-1",
  "resources": [],
  "detail": {
    "id": "evt_123456789",
    "type": "payment_intent.succeeded",
    "created": 1738860000,
    "data": {
      "object": {
        "id": "pi_123456789",
        "object": "payment_intent",
        "metadata": {
          "donationId": "b0a2b0c4-0c6a-9b30-8b3f2a3d6a8b",
          "campaignId": "campanha-2026"
        }
      }
    }
  }
}
```

## Payment Element (Apple Pay / Google Pay)
Quando voce usa o Payment Element no frontend, Apple Pay e Google Pay aparecem automaticamente se:
- o dispositivo/navegador for elegivel
- o dominio estiver verificado e configurado no Stripe

Nao e necessario criar endpoints separados para Apple Pay/Google Pay.
