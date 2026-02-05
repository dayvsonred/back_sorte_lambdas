# Lambda login

## Build
```powershell
cd "c:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\login"
$env:GOOS="linux"
$env:GOARCH="amd64"
$env:CGO_ENABLED="0"
go build -o bootstrap .
Compress-Archive -Path bootstrap -DestinationPath lambda.zip -Force
```
```powershell
$env:GOOS="linux"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"; go build -o bootstrap .; Compress-Archive -Path bootstrap -DestinationPath lambda.zip -Force
```

## Deploy (Terraform)
```powershell
cd "c:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\login\terraform"
terraform init
terraform apply -var "aws_region=us-east-1" -var "dynamodb_table=core" -var "lambda_zip=../lambda.zip"
```
```powershell
cd "c:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\login\terraform"
terraform apply -var "aws_region=us-east-1" -var "dynamodb_table=core" -var "lambda_zip=../lambda.zip" -var "jwt_secret=XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"




aws lambda update-function-code   --function-name back-sorte-login   --zip-file fileb://lambda.zip   --region us-east-1


```

## Exemplo de uso (requests)
```bash
# API Gateway (HTTP API)
BASE_URL="https://rm0t2sapef.execute-api.us-east-1.amazonaws.com"

curl -X POST "$BASE_URL/login" \
  -H "Authorization: Basic QVBJX05BTUVfQUNDRVNTOkFQSV9TRUNSRVRfQUNDRVNT" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=password&username=joao@email.com&password=123456"
```

## Exemplo de uso (requests)
```bash
curl -X POST "https://rm0t2sapef.execute-api.us-east-1.amazonaws.com/login" \
  -H "Authorization: Basic QVBJX05BTUVfQUNDRVNTOkFQSV9TRUNSRVRfQUNDRVNT" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=password&username=lucas_dell01@gmail.com&password=123456"
```


## Exemplo de uso (Teste WEB)
```bash

{
  "version": "2.0",
  "routeKey": "POST /login",
  "rawPath": "/login",
  "rawQueryString": "",
  "headers": {
    "authorization": "Basic QVBJX05BTUVfQUNDRVNTOkFQSV9TRUNSRVRfQUNDRVNT",
    "content-type": "application/x-www-form-urlencoded"
  },
  "requestContext": {
    "http": {
      "method": "POST",
      "path": "/login"
    }
  },
  "isBase64Encoded": false,
  "body": "grant_type=password&username=lucas_dell01@gmail.com&password=123456"
}
```




aws apigatewayv2 update-api  --api-id rm0t2sapef  --region us-east-1   --cors-configuration AllowOrigins="https://www.thepuregrace.com,https://thepuregrace.com,http://localhost:3487",AllowMethods="GET,POST,PUT,PATCH,DELETE,OPTIONS",AllowHeaders="authorization,content-type,origin,accept,x-requested-with",MaxAge=86400





curl -i -X OPTIONS -H "Origin: https://www.thepuregrace.com" -H "Access-Control-Request-Method: POST"  https://rm0t2sapef.execute-api.us-east-1.amazonaws.com/login







cd "C:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\login"
$env:GOOS="linux" $env:GOARCH="amd64" go build -o bootstrap . Compress-Archive -Path bootstrap -DestinationPath lambda.zip -Force

aws lambda update-function-code `
  --function-name back-sorte-login `
  --zip-file fileb://lambda.zip `
  --region us-east-1
