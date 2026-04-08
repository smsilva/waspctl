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

## Ideias de comandos

### Cloud Providers

```shell
waspctl provider list

# Exemplo de output
NAME    DESCRIPTION            CURRENT
aws     Amazon Web Services    no
gcp     Google Cloud Platform  no
azure   Microsoft Azure        no

# Definir o provedor de nuvem ativo
waspctl config --set provider aws

# Qualquer comando de criação de recurso compartilhado usaria o provedor de nuvem ativo, se não houver um, o comando pediria para selecionar um. O comando salvará a seleção em um arquivo que, por padrão, será criado em ~/.wasp/config.yaml, mas poderia ser customizado usando um parâmetro --config.
```

#### Exemplo de configuração salva em ~/.wasp/config.yaml

```yaml
provider: aws
```

```shell
# Cria o domínio global wasp.silvios.me
waspctl dns create \
  --name wasp.silvios.me

# Cria um Container Registry
waspctl registry create \
  --name wasp \
  --region us-east-1

# Cria um Database NoSQL para armazenar o estado da plataforma e dos clusters de clientes
waspctl database create \
  --name wasp-state \
  --region us-east-1

# Criar uma instância de plataforma WASP
waspctl instance create \
  --name <platform-name> \
  --region <region> \
  --domain <domain-name>

# Exemplo com valores
waspctl instance create \
  --name platform-cluster-use1 \
  --region us-east-1 \
  --domain wasp.silvios.me

# No exemplo com AWS, o comando confirmaria a AWS account e a região caso não fosse passado o parâmetro --yes, para evitar criar recursos na conta ou região errada. Exemplo:
# Please confirm the following information before proceeding:
#
#   Account: 123456789012
#   Region: us-east-1
#
# Do you want to proceed? (y/n) y
#

# Listar instâncias de plataforma WASP
waspctl instance list

# Exemplo de output
NAME                   PROVIDER   REGION      DOMAIN           STATUS
platform-cluster-use1  aws        us-east-1   wasp.silvios.me  ready
platform-cluster-brs1  aws        sa-east-1   wasp.silvios.me  provisioning

# Conectar em uma instância da plataforma WASP
waspctl instance connect <platform-name>

# Exemplo com valores
waspctl instance connect platform-cluster-use1

# Autenticar usando SSO
waspctl auth login --provider google
```
