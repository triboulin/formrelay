# FormRelay Admin

Microservice d'envoi de formulaires auto-hébergé, multi-clients et sécurisé, écrit en Go. Reçoit des soumissions HTML classiques (`POST` `application/x-www-form-urlencoded`), les relaie par email via SMTP et archive chaque soumission dans SQLite. Un panel d'administration HTMX permet de gérer les clients et de consulter les logs.

## Stack

- Go 1.22+ (`net/http`, `html/template`, sans framework)
- SQLite via `modernc.org/sqlite` (pure Go, sans CGO)
- Admin : HTMX + Tailwind CSS (CDN) + Alpine.js (CDN)
- Docker multi-stage (image finale `alpine`)

## Démarrage rapide (Docker)

```bash
cp .env.example .env
# éditer .env : ADMIN_USER / ADMIN_PASS / SMTP_*

docker compose up -d --build
```

L'admin est disponible sur `http://localhost:8080/admin` (Basic Auth).

## Démarrage en local (sans Docker)

```bash
cp .env.example .env
go run ./cmd/server
```

## Utilisation

### 1. Créer un client

Dans `/admin/clients`, cliquer sur **+ Nouveau client** et renseigner :
- Nom du client
- Email de destination (où les messages seront relayés)

Le nom et l'email de destination peuvent être modifiés à tout moment via le bouton **Modifier** sur la ligne du client : l'identifiant (`/f/{client_id}`) ne change jamais, donc les formulaires déjà intégrés côté client continuent de fonctionner sans modification.

Un identifiant (UUID) est généré : c'est le endpoint du formulaire, `/f/{client_id}`.

### 2. Intégrer le formulaire côté client

```html
<form action="https://formrelay.example.com/f/VOTRE-CLIENT-ID" method="POST">
    <input type="text" name="name" placeholder="Nom">
    <input type="email" name="email" placeholder="Email">
    <input type="text" name="subject" placeholder="Sujet">
    <textarea name="message"></textarea>

    <!-- Optionnel : redirection après envoi -->
    <input type="hidden" name="_next" value="https://votre-site.fr/merci">

    <button type="submit">Envoyer</button>
</form>
```

Le corps de l'email relayé est toujours au format :
```
Nouvelle soumission de formulaire de la part de : {name} ({email})

{message}
```

Champs spéciaux reconnus :
- `name`, `email`, `message` : utilisés pour construire le corps de l'email (voir ci-dessus).
- `subject` : devient le sujet de l'email envoyé.
- `_next` : URL de redirection après envoi (sinon une page de confirmation HTML par défaut est affichée).
- `_replyto` : adresse à utiliser comme Reply-To si différente de `email`.
- `_subject` : sujet par défaut si le champ `subject` n'est pas fourni.

### 3. Consulter les logs

`/admin/logs` liste toutes les soumissions avec filtres par client et statut (`SUCCESS`, `FAILED`, `BLOCKED`), pagination et détail du payload JSON par soumission.

### 4. API JSON (provisioning automatisé)

En plus du panel HTMX, une petite API JSON permet de créer/lister des clients depuis un script, sans parser du HTML. Protégée par les mêmes identifiants que `/admin` (Basic Auth).

```bash
# Créer un client
curl -u "$ADMIN_USER:$ADMIN_PASS" -X POST https://formrelay.example.com/api/clients \
  -H "Content-Type: application/json" \
  -d '{"name": "Site Client X", "destination_email": "contact@client-x.fr"}'
# -> 201 {"id": "...", "name": "...", "destination_email": "...", "active": true, "endpoint": "/f/...", "created_at": "..."}

# Lister les clients existants
curl -u "$ADMIN_USER:$ADMIN_PASS" https://formrelay.example.com/api/clients
```

`endpoint` (`/f/{id}`) est le chemin à concaténer à l'URL de l'instance pour obtenir l'`action` du formulaire à intégrer côté site. Voir l'utilitaire `utils/formrelay-setup` (dans le dépôt `triboulin.fr`) qui s'appuie sur cette API pour provisionner un client et générer directement le snippet de formulaire.

## Sécurité intégrée

- **Rate limiting** : 1 soumission max toutes les 5 secondes par IP (429 sinon), en mémoire.
- **Client actif requis** : un client désactivé renvoie 404 sur son endpoint.
- **Admin protégé** : HTTP Basic Auth (`ADMIN_USER` / `ADMIN_PASS`).
- **Rétention** : purge automatique quotidienne (minuit) des soumissions de plus d'un an.
- **Envoi SMTP asynchrone** : les emails sont mis en queue (channel Go) et envoyés par des workers dédiés, sans bloquer la réponse HTTP.

## Variables d'environnement

Voir `.env.example`.

| Variable | Description |
|---|---|
| `PORT` | Port d'écoute HTTP (défaut `8080`) |
| `DATABASE_URL` | Chemin du fichier SQLite |
| `ADMIN_USER` / `ADMIN_PASS` | Identifiants du panel admin |
| `SMTP_HOST` / `SMTP_PORT` / `SMTP_USER` / `SMTP_PASS` | Configuration SMTP globale |
| `FROM_EMAIL` / `FROM_NAME` | Expéditeur des emails relayés |

## Structure du projet

```
├── cmd/server/main.go       # Point d'entrée, câblage des dépendances
├── internal/
│   ├── config/               # Chargement config (.env + variables d'env)
│   ├── database/              # Connexion SQLite + migrations
│   ├── handler/               # Handlers HTTP (public /f/ et admin HTMX)
│   ├── middleware/            # Rate limiter, Basic Auth
│   ├── model/                 # Structures de données
│   ├── repository/            # Requêtes SQL (clients, submissions)
│   └── service/                # Worker SMTP, logique métier, rétention
├── templates/
│   ├── admin/                  # Templates HTMX (base, dashboard, clients, logs)
│   └── public/                 # Page de confirmation par défaut
├── Dockerfile
├── docker-compose.yml
└── go.mod
```

## Tests

Le projet dispose d'une suite de tests unitaires (SQLite en mémoire, faux serveur SMTP local, `httptest`) couvrant middleware, config, base de données, repositories, services et handlers HTTP.

```bash
go test ./...
```

Coverage par package (`go test ./... -cover`) :

| Package | Coverage |
|---|---|
| `internal/config` | 100% |
| `internal/service` | 91% |
| `internal/middleware` | 87% |
| `internal/repository` | 83% |
| `internal/handler` | 82% |
| `internal/database` | 80% |
| **Total (`internal/...`)** | **~86%** |

`cmd/server` (le point d'entrée `main.go`) n'est pas couvert par des tests unitaires : il ne fait que du câblage de dépendances et la gestion du cycle de vie du serveur (écoute HTTP, arrêt propre sur signal), difficilement testable unitairement et à faible valeur ajoutée à tester ainsi. C'est un choix courant pour ce type de code.

Pour un rapport détaillé par fonction ou un fichier HTML navigable :

```bash
go test ./internal/... -coverprofile=cover.out
go tool cover -func=cover.out    # résumé par fonction
go tool cover -html=cover.out    # rapport HTML interactif
```

## Notes de déploiement

- Le volume `./data` doit être monté pour persister la base SQLite entre redémarrages du conteneur.
- Le binaire est compilé statiquement (`CGO_ENABLED=0`), l'image finale n'a besoin que des certificats CA pour les connexions SMTP en TLS.
- Le endpoint `/healthz` peut être utilisé pour les healthchecks (déjà configuré dans `docker-compose.yml`).

## Build & Push vers le registre

Le projet fournit deux scripts selon le shell utilisé :

- Windows PowerShell/CMD : `build-push.bat`
- Bash (Git Bash, WSL, Linux) : `build-push.sh`

Les deux scripts :

- lisent le fichier `VERSION`
- incrémentent automatiquement le `PATCH` (format `MAJOR.MINOR.PATCH`)
- mettent à jour `VERSION`
- build et push l'image vers `registry.triboulin.fr/formrelay-admin:[nouvelle-version]`

Exemples :

```powershell
.\build-push.bat
```

```bash
bash ./build-push.sh
```

Important : ne pas lancer `bash build-push.bat` (script batch Windows non compatible Bash).
