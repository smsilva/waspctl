# waspctl

## Ideia inicial

Inicialmente na AWS, usar EKS clusters com ambientes diferentes por cliente separados por namespaces e network policies.

Cada ambiente de cada cliente possui seu próprio Gateway de entrada como:

customer1-useast1-prod.wasp.silvios.me
customer2-brsouth1-prod.wasp.silvios.me

A ideia é ter uma entrada única em:

wasp.silvios.me

Descobrir o cliente pelo Domain do e-mail (john@customer1.com) e para esse usuário redirecionar o tráfego para o Gateway correspondente.

## Tecnologias que poderíamos usar inicialmente

- k3d para criar um cluster local que poderia ser usado junto com o Crossplane como Undercloud para criar e gerenciar recursos compartilhados inicialmente na AWS
- KCL: para evitar repetição de código yaml e ter uma linguagem de programação mais expressiva para definir os recursos compartilhados e a hierarquia da plataforma
- SSO: para autenticação, inicialmente usando contas Google, mas a ideia é ter uma arquitetura de autenticação flexível que permita adicionar outros provedores de identidade no futuro, como AWS IAM, Azure AD, etc.
- EKS: para criar clusters de plataforma e clusters de clientes gerenciados na AWS, mas a ideia é ter uma arquitetura de provisionamento de clusters flexível que permita adicionar outros provedores de Kubernetes no futuro, como GKE, AKS, etc.
- Global Accelerator: para criar uma rede global de baixa latência que conecte os clusters de plataforma e os clusters de clientes, permitindo que os clientes acessem a plataforma de forma rápida e confiável, independentemente de onde estejam localizados.
- DynamoDB Global Table: para criar um banco de dados global que armazene o estado da plataforma e dos clusters de clientes, permitindo que a plataforma seja altamente disponível e resiliente, mesmo em caso de falhas regionais.
- Istio: Service Mesh e Gateway.
- external-dns: para gerenciar os registros DNS dos clusters de clientes de forma automática, com base nas regras definidas na plataforma.
- cert-manager: para gerenciar os certificados TLS dos clusters de clientes de forma automática, garantindo que as conexões sejam seguras e confiáveis.
- argocd: para gerenciar a configuração dos clusters de clientes de forma declarativa, permitindo que as mudanças sejam aplicadas de forma consistente e controlada.
- github: para armazenar a configuração dos clusters de clientes e da plataforma de forma versionada, permitindo que as mudanças sejam auditáveis e revertíveis.

## Arquitetura inicial

```
Usuário
  │
  ▼
Route 53  (wasp.silvios.me → alias do Global Accelerator ou ALB)
  │
  ▼
Global Accelerator  (anycast, TLS pass-through ou terminação)
  │
  ▼
ALB  (listener HTTPS, regra default → Auth Service)
  │
  ├──[/login, /auth]──────────────────────────────────────────────────▶ Auth/Routing Service
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

## O que cada platform-cluster regional contém

- Auth Service — resolve tenant e emite token
- ALB regional — roteia para os customer-clusters da região
- DynamoDB local (ou Global Table replicada) — tenant registry
- Ingress/Gateway compartilhado — se fizer sentido centralizar

## Como o Global Accelerator se encaixa

Aqui ele passa a ter um papel mais importante: geo-routing automático.

O cliente em São Paulo bate em `wasp.silvios.me` → Global Accelerator roteia para o endpoint mais próximo → `platform-cluster-brs1` → Auth Service resolve o tenant → roteia para `customer3-brs1`.

## Serviços AWS envolvidos

### 1. Route 53
Apenas resolve wasp.silvios.me para o Global Accelerator ou ALB. Sem lógica aqui.

### 2. Global Accelerator (opcional mas recomendado dado seu setup atual)
Anycast global, reduz latência para clientes geograficamente distribuídos. Aponta para o ALB regional.

### 3. ALB (Application Load Balancer)
O coração do roteamento. Ele suporta:

Listener Rules com condições de cookie e header HTTP
Target Groups por destino (cada gateway de cliente vira um target group)
Integração com Cognito ou OIDC diretamente no listener para autenticação

Após o Auth Service setar um cookie tenant=customer1-useast1-prod, o ALB roteia nas próximas requisições via listener rule sem passar pelo Auth Service novamente.

### 4. Auth / Tenant Resolution Service (sua lógica, rodando no EKS)
Esse é o componente que você escreve. Responsabilidades:

Receber o e-mail no login
Fazer lookup no DynamoDB ou RDS → customer1.com → customer1-useast1-prod
Emitir JWT ou setar cookie com o tenant
Redirecionar o browser para o destino correto

### 5. DynamoDB (tenant registry)

Tabela simples:

```
email_domain (PK) | tenant_gateway
customer1.com     | customer1-useast1-prod.wasp.silvios.me
customer2.com     | customer2-brsouth1-prod.wasp.silvios.me
```

### 6. Cognito

Para externalizar o fluxo OIDC/OAuth, o ALB tem integração nativa com Cognito, intercepta requisições não autenticadas, redireciona para o Cognito, e após o login, usar Lambda Triggers no Cognito para enriquecer o token JWT com o tenant baseado no domínio do e-mail.

## Objetivos

- Criar e gerenciar uma Hierarchical Platform Topology

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

- Criar uma instância de plataforma WASP
- Conectar em uma instância da plataforma WASP
- Autenticar usando SSO (inicialmente com contas Google)
- Criar recursos compartilhados inicialmente na AWS como:
  - Container Registry
  - EKS Cluster de Plataforma
  - DNS global: wasp.silvios.me

## Ideia para plano inicial

Evolução natural da arquitetura

Fase 1: cluster único + auth service simples
Fase 2: platform-cluster separado dos customer-clusters
Fase 3: platform-clusters regionais + Global Accelerator + DynamoDB Global Table

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
