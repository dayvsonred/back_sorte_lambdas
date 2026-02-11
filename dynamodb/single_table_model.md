# DynamoDB Single-Table Design (core)

Este documento traduz o schema atual do PostgreSQL para Single Table Design no DynamoDB, mantendo as rotas em `routes/routes.go`.

## Tabela
- Nome: `core`
- PK: `PK` (string)
- SK: `SK` (string)
- GSI1: `GSI1PK`, `GSI1SK` (listar doacoes por usuario)
- GSI2: `GSI2PK`, `GSI2SK` (buscar por email)
- Provisioned capacity: RCUs 7 / WCUs 7 (free tier)

## Padroes de chave (principais itens)

### Usuario
- Usuario (perfil)
  - PK: `USER#{userId}`
  - SK: `PROFILE`
  - GSI2PK: `EMAIL#{emailLower}`
  - GSI2SK: `USER#{userId}`
  - Campos: name, email, password_hash, cpf, active, inicial, dell, date_create, date_update

- User details
  - PK: `USER#{userId}`
  - SK: `DETAILS`
  - Campos: cpf_valid, email_valid, cep, telefone, apelido, img_perfil, date_create, date_update

- Conta nivel
  - PK: `USER#{userId}`
  - SK: `ACCOUNT#LEVEL`
  - Campos: nivel, ativo, status, data_pagamento, tipo_pagamento, data_update

- Conta nivel pagamento
  - PK: `USER#{userId}`
  - SK: `ACCOUNT#PAYMENT#{paymentId}`
  - Campos: pago_data, pago, valor, status, codigo, data_create, referente, valido, txid, pg_status, cpf, chave, pixCopiaECola, expiracao

- Password recover
  - PK: `PWDREC#{emailLower}`
  - SK: `TS#{isoDateTime}#{recoverId}`
  - Campos: id_user, token, validated, to_send, attempt, blocked, date_valid, data_create

- Bank account (saque_conta)
  - PK: `USER#{userId}`
  - SK: `BANK#{bankId}`
  - Campos: banco, banco_nome, conta, agencia, digito, cpf, telefone, pix, active, dell, date_create, date_update

- Bank account lookup (by id)
  - PK: `BANK#{bankId}`
  - SK: `USER#{userId}`
  - Campos: user_id, active, dell

- Saque details
  - PK: `BANK#{bankId}`
  - SK: `WITHDRAW#{withdrawId}`
  - Campos: valor, realizado, error, date_create, date_update

### Doacao
- Doacao (perfil)
  - PK: `DONATION#{donationId}`
  - SK: `PROFILE`
  - GSI1PK: `USER#{userId}`
  - GSI1SK: `DONATION#{date_create}#{donationId}`
  - Campos: id_user, name, valor, active, dell, closed, date_start, date_create, date_update

- Doacao details (texto pode ser grande)
  - PK: `DONATION#{donationId}`
  - SK: `DETAILS`
  - Campos: texto, img_caminho, area

- Doacao link
  - PK: `LINK#{nome_link}`
  - SK: `DONATION#{donationId}`
  - Campos: id_doacao, id_user (opcional), name (opcional)

- Doacao pagamentos
  - PK: `DONATION#{donationId}`
  - SK: `PAYMENT`
  - Campos: valor_disponivel, valor_tranferido, data_tranferido, solicitado, data_solicitado, status, img, pdf, banco, conta, agencia, digito, pix, data_update

### Pix
- Pix QRCode (mensagens visiveis)
  - PK: `DONATION#{donationId}`
  - SK: `PIX#{data_criacao}#{pixId}`
  - Campos: valor, cpf, nome, mensagem, anonimo, visivel, data_criacao, txid

- Pix status (lookup rapido por txid)
  - PK: `TX#{txid}`
  - SK: `STATUS`
  - Campos: id_pix_qrcode, id_doacao, status, buscar, finalizado, data_pago, expiracao, tipo_pagamento, loc_id, loc_tipo_cob, loc_criacao, location, pix_copia_e_cola, chave

### Visualizacao
- Aggregado
  - PK: `DONATION#{donationId}`
  - SK: `VISUALIZATION`
  - Campos: visualization, donation_like, love, shared, acesse_donation, create_pix, create_cartao, create_paypal, create_google, create_pag1, create_pag2, create_pag3, date_create, date_update

- Detalhe
  - PK: `DONATION#{donationId}`
  - SK: `VIS#{date_create}#{visId}`
  - Campos: ip, id_user, idioma, tema, form, google, google_maps, google_ads, meta_pixel, Cookies_Stripe, Cookies_PayPal, visitor_info1_live, donation_like, love, shared, acesse_donation, create_pix, create_cartao, create_paypal, create_google, create_pag1, create_pag2, create_pag3, date_create

### Contact
- Mensagem
  - PK: `CONTACT#{contactId}`
  - SK: `DETAIL`
  - Campos: nome, email, mensagem, ip, location, token, view, data_create

### Email automacao (doacao)
- Cota diaria de envio
  - PK: `EMAIL#QUOTA#{yyyy-mm-dd}`
  - SK: `COUNTER`
  - Campos: send_count, date_update

- Pendencias de envio (quando atingir limite diario)
  - PK: `EMAIL#PENDING`
  - SK: `TS#{epochMs}#{id}`
  - Campos: status, payload, attempts, next_attempt_at, date_create, date_update

- Token de confirmacao de e-mail
  - PK: `EMAIL#VERIFY#{token}`
  - SK: `USER#{userId}`
  - Campos: user_id, email, donation_id, used, date_create, expires_at

## Observacoes de acesso (rotas atuais)
- Login / busca por email: usar GSI2 em item USER#... (EMAIL#)
- Listar doacoes por usuario: GSI1PK=USER#id
- Donation by link: GetItem por PK=LINK#@nome
- Mensagens visiveis: Query PK=DONATION#id com SK begins_with PIX# e filter visivel=true
- Resumo doacao (total e distinct cpf): use agregacao incremental (counter) e item auxiliar por CPF:
  - PK: DONATION#{id} / SK: CPF#{cpf}
  - Se nao existir, cria e incrementa contador total_doadores no item PAYMENT ou AGG
- Monitorar pagamentos ativos: sem GSI extra, usa Scan com filtros em itens TX#... (baixo volume/free tier)

## Itens com tamanho
- `texto` pode exceder 1KB. Se quiser manter itens <= 1KB:
  - Guardar `texto` apenas em DETAILS e retornar em lote (BatchGet) quando listar doacoes.

