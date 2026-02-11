## Painel local (S3 -> Parquet/DuckDB -> Streamlit)

### Estrategia de baixo custo
- Baixar o export uma vez para o PC (evita leitura repetida no S3).
- Gerar dataset local (`.parquet` + `.duckdb`).
- Rodar o painel local consultando so arquivos locais.

### 1) Instalar dependencias
```powershell
cd "c:\Users\niore\Documents\projeto sorteio doacao\back_sorte_go\back_sorte_lambdas\painel"
python -m venv .venv;  .\.venv\Scripts\Activate.ps1;  pip install -r .\requirements.txt
```

### 2) Comando principal (recomendado)
Executa tudo em sequencia:
- etapa 1: download incremental do S3
- etapa 2: build do dataset (Parquet + DuckDB)
- etapa 3: abre o Streamlit

```powershell
python .\run_pipeline.py --from-last
```

Se quiser so processar sem abrir navegador:
```powershell
python .\run_pipeline.py --from-last --no-open
```

### 3) Comandos separados (opcional)
Baixar export do S3 para o PC:
```powershell
python .\sync_s3_export.py --region us-east-1 --bucket bd-thepuregrace-v1-dinamodb-core --prefix-base exports/core --date 2026-02-11
```

Modo incremental (do ultimo dia local ate o ultimo dia no S3):
```powershell
python .\sync_s3_export.py --from-last --region us-east-1 --bucket bd-thepuregrace-v1-dinamodb-core --prefix-base exports/core
```

Gerar Parquet e DuckDB locais:
```powershell
python .\build_dataset.py
```

Subir painel:
```powershell
streamlit run .\app.py
```

### O que o painel mostra
- Usuarios criados por dia
- Doacoes criadas por dia
- Acessos diarios (eventos `VIS#`)
- Explosao de telas acessadas (`create_pag1`, `create_pag2`, `create_pag3`, `create_pix`, `create_cartao`, `create_paypal`, `create_google`)
- Doacoes pagas por dia (`TX#.../STATUS` com `finalizado=true`)

### Observacao sobre "paginas acessadas"
- O painel usa os campos do evento de visualizacao (`create_pag1`, `create_pag2`, `create_pag3`, `acesse_donation`).
- Se um tipo de acesso nao estiver sendo salvo na tabela `core`, nao aparece no painel.
