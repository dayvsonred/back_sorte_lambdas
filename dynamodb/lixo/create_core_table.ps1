param(
  [string]$TableName = "core",
  [string]$Region = $env:AWS_REGION
)

if ([string]::IsNullOrWhiteSpace($Region)) {
  Write-Error "AWS_REGION nao definido. Defina a variavel de ambiente AWS_REGION ou passe -Region." 
  exit 1
}

$exists = $false
try {
  aws dynamodb describe-table --table-name $TableName --region $Region | Out-Null
  $exists = $true
} catch {
  $exists = $false
}

if ($exists) {
  Write-Host "Tabela '$TableName' ja existe na regiao $Region. Nenhuma acao realizada."
  exit 0
}

Write-Host "Criando tabela '$TableName' na regiao $Region..."

aws dynamodb create-table `
  --table-name $TableName `
  --attribute-definitions `
      AttributeName=PK,AttributeType=S `
      AttributeName=SK,AttributeType=S `
      AttributeName=GSI1PK,AttributeType=S `
      AttributeName=GSI1SK,AttributeType=S `
      AttributeName=GSI2PK,AttributeType=S `
      AttributeName=GSI2SK,AttributeType=S `
  --key-schema `
      AttributeName=PK,KeyType=HASH `
      AttributeName=SK,KeyType=RANGE `
  --provisioned-throughput ReadCapacityUnits=7,WriteCapacityUnits=7 `
  --global-secondary-indexes `
      "[{
        \"IndexName\":\"GSI1\",
        \"KeySchema\":[{\"AttributeName\":\"GSI1PK\",\"KeyType\":\"HASH\"},{\"AttributeName\":\"GSI1SK\",\"KeyType\":\"RANGE\"}],
        \"Projection\":{\"ProjectionType\":\"ALL\"},
        \"ProvisionedThroughput\":{\"ReadCapacityUnits\":7,\"WriteCapacityUnits\":7}
      },{
        \"IndexName\":\"GSI2\",
        \"KeySchema\":[{\"AttributeName\":\"GSI2PK\",\"KeyType\":\"HASH\"},{\"AttributeName\":\"GSI2SK\",\"KeyType\":\"RANGE\"}],
        \"Projection\":{\"ProjectionType\":\"ALL\"},
        \"ProvisionedThroughput\":{\"ReadCapacityUnits\":7,\"WriteCapacityUnits\":7}
      }]"

if ($LASTEXITCODE -ne 0) {
  Write-Error "Falha ao criar a tabela."
  exit 1
}

Write-Host "Tabela criada. Aguarde STATUS=ACTIVE antes de usar."
