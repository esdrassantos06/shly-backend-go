# Guia de Deploy - Zipway API

## 1. Enviar para GitHub

### Inicializar repositório

```bash
cd "backend go"

# Inicializar git
git init

# Adicionar arquivos
git add .

# Primeiro commit
git commit -m "Initial commit: Zipway URL Shortener API"

# Adicionar remote (substitua SEU_USUARIO)
git remote add origin https://github.com/SEU_USUARIO/zipway-backend-go.git

# Push
git branch -M main
git push -u origin main
```

## 2. Melhores Opções de Deploy (2025)

### Opção 1: Railway (Recomendado - Grátis até $5/mês)

**Por quê:**

- ✅ Plano grátis: $5 crédito/mês (suficiente para apps pequenos)
- ✅ Deploy automático do GitHub
- ✅ PostgreSQL e Redis incluídos
- ✅ Custom domains grátis
- ✅ SSL automático
- ✅ Muito fácil de configurar

**Setup:**

1. Acesse [railway.app](https://railway.app)
2. Login com GitHub
3. New Project → Deploy from GitHub repo
4. Selecione seu repositório
5. Adicione variáveis de ambiente:
   ```
   DATABASE_URL=<sua connection string do Supabase>
   REDIS_URL=<sua connection string do Redis>
   BASE_URL=https://api.shly.pt
   ALLOWED_ORIGIN=https://shly.pt
   ```
6. Settings → Generate Domain → Custom Domain → `api.shly.pt`
7. Configure DNS no seu provedor:
   - Tipo: CNAME
   - Nome: api
   - Valor: [domínio fornecido pelo Railway]

**Custo:** Grátis (até $5/mês de uso)

---

### Opção 2: Fly.io (Grátis)

**Por quê:**

- ✅ Plano grátis generoso (3 VMs compartilhadas)
- ✅ Deploy global (edge locations)
- ✅ PostgreSQL e Redis disponíveis
- ✅ Custom domains grátis
- ✅ SSL automático

**Setup:**

1. Instale Fly CLI: `curl -L https://fly.io/install.sh | sh`
2. Login: `fly auth login`
3. Crie `fly.toml` na raiz:

   ```toml
   app = "zipway-api"
   primary_region = "gru"  # São Paulo

   [build]
     builder = "paketobuildpacks/builder:base"

   [env]
     PORT = "8080"

   [[services]]
     internal_port = 8080
     protocol = "tcp"

     [[services.ports]]
       port = 80
       handlers = ["http"]
     [[services.ports]]
       port = 443
       handlers = ["tls", "http"]

   [[services.http_checks]]
     interval = "10s"
     timeout = "2s"
     grace_period = "5s"
     method = "GET"
     path = "/"
   ```

4. Deploy: `fly deploy`
5. Adicione secrets:
   ```bash
   fly secrets set DATABASE_URL="sua_url"
   fly secrets set REDIS_URL="sua_url"
   fly secrets set BASE_URL="https://api.shly.pt"
   fly secrets set ALLOWED_ORIGIN="https://shly.pt"
   ```
6. Custom domain: `fly certs add api.shly.pt`

**Custo:** Grátis (3 VMs compartilhadas)

---

### Opção 3: Render (Grátis com limitações)

**Por quê:**

- ✅ Plano grátis disponível
- ✅ Deploy automático do GitHub
- ✅ PostgreSQL e Redis disponíveis
- ⚠️ Spins down após 15min de inatividade (primeira requisição lenta)

**Setup:**

1. Acesse [render.com](https://render.com)
2. New → Web Service
3. Conecte GitHub repo
4. Configurações:
   - Build Command: `go build -o bin/api cmd/api/main.go`
   - Start Command: `./bin/api`
   - Environment: Docker
5. Adicione variáveis de ambiente
6. Custom Domain → `api.shly.pt`

**Custo:** Grátis (com spin down)

---

### Opção 4: Google Cloud Run (Grátis até certo limite)

**Por quê:**

- ✅ Plano grátis: 2 milhões de requisições/mês
- ✅ Paga apenas pelo uso
- ✅ Escala automaticamente para zero
- ✅ Custom domains

**Setup:**

1. Instale gcloud CLI
2. Crie `Dockerfile` (já existe)
3. Build e deploy:
   ```bash
   gcloud run deploy zipway-api \
     --source . \
     --platform managed \
     --region us-central1 \
     --allow-unauthenticated \
     --set-env-vars DATABASE_URL=...,REDIS_URL=...
   ```
4. Custom domain via Cloud Run

**Custo:** Grátis (até 2M requisições/mês)

---

### Opção 5: DigitalOcean App Platform (Barato)

**Por quê:**

- ✅ $5/mês (muito barato)
- ✅ Deploy automático
- ✅ PostgreSQL e Redis disponíveis
- ✅ Custom domains
- ✅ Sem spin down

**Setup:**

1. Acesse [digitalocean.com](https://digitalocean.com)
2. Create → App Platform
3. Connect GitHub
4. Configure build e start commands
5. Adicione variáveis de ambiente
6. Custom domain

**Custo:** $5/mês

---

## 3. Configurar DNS para api.shly.pt

No seu provedor de DNS (onde está configurado shly.pt):

1. Adicione registro CNAME:

   - Tipo: CNAME
   - Nome: api
   - Valor: [domínio fornecido pela plataforma]
   - TTL: 3600

2. Ou registro A (se a plataforma fornecer IP):
   - Tipo: A
   - Nome: api
   - Valor: [IP fornecido]
   - TTL: 3600

## 4. Recomendação Final

**Para produção com custo mínimo:**

1. **Railway** - Melhor custo/benefício ($5 crédito grátis)
2. **Fly.io** - Totalmente grátis, mas requer mais configuração
3. **DigitalOcean** - $5/mês, mais estável

**Para começar rápido:**

- Use **Railway** - mais fácil e rápido de configurar

## 5. Variáveis de Ambiente para Produção

```bash
DATABASE_URL=postgresql://postgres.xxxxx:SUA_SENHA@aws-0-us-east-1.pooler.supabase.com:6543/postgres
REDIS_URL=redis://redis:6379  # Ou Redis do Supabase
BASE_URL=https://api.shly.pt
ALLOWED_ORIGIN=https://shly.pt
```

## 6. Checklist de Deploy

- [ ] Código no GitHub
- [ ] Dockerfile configurado
- [ ] Variáveis de ambiente configuradas
- [ ] DNS configurado (CNAME para api.shly.pt)
- [ ] SSL/HTTPS funcionando
- [ ] Testar endpoint: `curl https://api.shly.pt/`
- [ ] Testar autenticação
- [ ] Monitorar logs
