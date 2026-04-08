# waspctl

Este projeto é um playground para experimentar a construção de uma CLI e uma plataforma de orquestração de clusters Kubernetes multi-tenant na AWS, chamada WASP (Hierarchical Platform Topology).

A maior parte dele se baseia em documentos da AWS:

*Building a Multi-Tenant SaaS Solution Using Amazon EKS*
by Toby Buckley and Ranjith Raman
https://aws.amazon.com/pt/blogs/apn/building-a-multi-tenant-saas-solution-using-amazon-eks/

*Operating a multi-regional stateless application using Amazon EKS*
by Re Alvarez-Parmar
https://aws.amazon.com/pt/blogs/containers/operating-a-multi-regional-stateless-application-using-amazon-eks/

*Amazon EKS Blueprints for Terraform*
https://aws-ia.github.io/terraform-aws-eks-blueprints/

`waspctl` é uma CLI para provisionar uma Hierarchical Platform Topology de clusters Kubernetes multi-tenant na AWS, com entrada global única em `wasp.silvios.me`.

O roteamento de tráfego é baseado no domínio do e-mail do usuário: ao fazer login com `john@customer1.com`, o Auth Service resolve o tenant e redireciona para o gateway correspondente (`customer1-useast1-prod.wasp.silvios.me`), de forma transparente.

## Objetivos

- Criar e gerenciar uma Hierarchical Platform Topology
- Criar uma instância de plataforma WASP
- Conectar em uma instância da plataforma WASP
- Autenticar usando SSO (inicialmente com contas Google)
- Criar recursos compartilhados na AWS:
  - Container Registry
  - EKS Cluster de Plataforma
  - DNS global: wasp.silvios.me

**Roadmap de evolução:**
- Fase 1: cluster único + Auth Service simples
- Fase 2: platform-cluster separado dos customer-clusters
- Fase 3: platform-clusters regionais + Global Accelerator + DynamoDB Global Table

## Stack

**Cloud (AWS)**
- ALB
- Cognito
- DynamoDB Global Table
- EKS
- Global Accelerator
- Route 53

**In-cluster**
- ArgoCD (GitOps)
- cert-manager
- external-dns
- Istio (Service Mesh e Gateway)

**Configuração**
- KCL — linguagem expressiva para definir recursos e a hierarquia da plataforma, evitando repetição de YAML

**Dev local**
- k3d + Crossplane (Undercloud) — simula o ambiente de plataforma localmente

**Identity**
- SSO via Google (extensível para AWS IAM, Azure AD)

**Source of truth**
- GitHub — configuração versionada de clusters e plataforma

## Arquitetura

### Topology

```
                       wasp.silvios.me
                              │
                     Global Accelerator
                              │
                 ┌────────────┴────────────┐
                 ▼                         ▼
        platform-cluster-use1      platform-cluster-brs1
        (us-east-1)                (sa-east-1)
                 │                         │
        ┌────────┴────────┐       ┌────────┴────────┐
        ▼                 ▼       ▼                 ▼
  customer1-use1    customer2-use1  customer3-brs1  customer4-brs1
```

O Global Accelerator faz geo-routing automático: um cliente em São Paulo bate em `wasp.silvios.me`, é roteado para `platform-cluster-brs1`, o Auth Service resolve o tenant e roteia para `customer3-brs1`.

Cada platform-cluster regional contém:
- Auth Service — resolve tenant e emite token
- ALB regional — roteia para os customer-clusters da região
- DynamoDB local (ou Global Table replicada) — tenant registry

### Request flow

```
Usuário
  │
  ▼
Route 53  (wasp.silvios.me → Global Accelerator)
  │
  ▼
Global Accelerator  (anycast, TLS pass-through ou terminação)
  │
  ▼
ALB  (listener HTTPS, regra default → Auth Service)
  │
  ├──[/login, /auth]──────────────────────────────────────────────────▶ Auth Service
  │                                                                          │
  │                                                               Lookup: john@customer1.com
  │                                                               → customer1-useast1-prod
  │                                                                          │
  │                                                               Seta cookie/JWT com tenant
  │
  └──[/* com cookie/JWT válido]──────────────────────────────▶ ALB Listener Rule
                                                                (header/cookie match)
                                                                    │
                                                         ┌──────────┴──────────┐
                                                         ▼                     ▼
                                              customer1-useast1-prod    customer2-brsouth1-prod
                                              (target group → NLB/     (target group → NLB/
                                               Istio GW no EKS)         Istio GW no EKS)
```

O Auth Service faz lookup no DynamoDB pelo domínio do e-mail e emite um JWT com o tenant. Nas requisições seguintes, o ALB roteia diretamente via cookie/header sem passar pelo Auth Service novamente.

**DynamoDB — tenant registry:**

```
email_domain (PK) | tenant_gateway
customer1.com     | customer1-useast1-prod.wasp.silvios.me
customer2.com     | customer2-brsouth1-prod.wasp.silvios.me
```

**Cognito (opcional):** o ALB tem integração nativa com Cognito para externalizar o fluxo OIDC/OAuth. Lambda Triggers enriquecem o JWT com o tenant baseado no domínio do e-mail.

## CLI

Configuração global em `~/.wasp/config.yaml` (substituível com `--config`). Comandos que criam recursos em cloud solicitam confirmação de account e região antes de prosseguir, a menos que `-y/--yes` seja passado.

**Flags globais:**

| Flag | Descrição | Default |
|------|-----------|---------|
| `-o, --output table\|json\|yaml` | Formato de output | `table` |
| `-y, --yes` | Skip de confirmação | `false` |
| `--config <path>` | Arquivo de config alternativo | `~/.wasp/config.yaml` |

### config

Gerencia o arquivo de configuração local, seguindo a mesma convenção do `git config`.

```shell
waspctl config --set provider aws
waspctl config --get provider
waspctl config --list
```

```yaml
# ~/.wasp/config.yaml
provider: aws
```

### provider

```shell
waspctl provider list

# Output
NAME   DESCRIPTION            ACTIVE
aws    Amazon Web Services    yes
azure  Microsoft Azure        no
gcp    Google Cloud Platform  no

waspctl provider set gcp
```

### dns

```shell
waspctl dns create --name wasp.silvios.me

waspctl dns list

# Output
NAME             PROVIDER  STATUS
wasp.silvios.me  aws       active

waspctl dns delete --name wasp.silvios.me
```

### registry

```shell
waspctl registry create --name wasp --region us-east-1

waspctl registry list

# Output
NAME  PROVIDER  REGION     STATUS
wasp  aws       us-east-1  active

waspctl registry delete --name wasp
```

### database

```shell
waspctl database create --name wasp-state --region us-east-1

waspctl database list

# Output
NAME        PROVIDER  REGION     STATUS
wasp-state  aws       us-east-1  active
```

### instance

Gerencia platform-clusters.

```shell
waspctl instance create \
  --name platform-cluster-use1 \
  --region us-east-1 \
  --domain wasp.silvios.me

# Confirmation prompt
# Please confirm:
#
#   Account : 123456789012
#   Region  : us-east-1
#
# Proceed? (y/n) y

waspctl instance list

# Output
NAME                   PROVIDER  REGION     DOMAIN           STATUS
platform-cluster-brs1  aws       sa-east-1  wasp.silvios.me  provisioning
platform-cluster-use1  aws       us-east-1  wasp.silvios.me  ready

waspctl instance connect platform-cluster-use1

waspctl instance delete platform-cluster-use1
```

### customer

Gerencia customer-clusters. O flag `--instance` é obrigatório para garantir clareza sobre em qual platform-cluster o recurso será criado.

```shell
waspctl customer create \
  --name customer1-use1 \
  --instance platform-cluster-use1 \
  --region us-east-1

waspctl customer list --instance platform-cluster-use1

# Output
NAME            INSTANCE               REGION     STATUS
customer1-use1  platform-cluster-use1  us-east-1  ready
customer2-use1  platform-cluster-use1  us-east-1  ready

waspctl customer delete \
  --name customer1-use1 \
  --instance platform-cluster-use1
```

### tenant

Gerencia o registry de tenants (email domain → gateway).

```shell
waspctl tenant create \
  --domain customer1.com \
  --gateway customer1-useast1-prod.wasp.silvios.me

waspctl tenant list

# Output
DOMAIN         GATEWAY
customer1.com  customer1-useast1-prod.wasp.silvios.me
customer2.com  customer2-brsouth1-prod.wasp.silvios.me

waspctl tenant delete --domain customer1.com
```

### auth

```shell
waspctl auth login --provider google

waspctl auth logout

waspctl auth whoami

# Output
EMAIL               PROVIDER  TENANT
john@customer1.com  google    customer1-useast1-prod
```
