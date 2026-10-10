# Palier sans Consul — déploiement du notifier

Le nouveau gateway (`infra-core`) est en production depuis le 2026-09-25, sans
Consul. Ce document décrit la mise en service du notifier à ce palier. Plan
détaillé et fenêtre de bascule : `infra-core/docs/plans/notifier-migration.md`.

## Répartition

| Élément | Hôte | Compose (Dokploy) |
|---|---|---|
| Serveur Go (API + workers) | `192.168.24.38`, port `8055` | `infra/notifier` → `deploy/backend/docker-compose.yml` |
| UI | `10.20.10.24` | `infra/notifier-ui` → `deploy/ui/docker-compose.yml` |
| Relais de transition | `10.20.10.24`, port `8055` | `deploy/relay/docker-compose.yml` |
| Base | `172.18.1.23/notifier` (copiée de `172.18.1.21/notifier`) | — |
| File `email:jobs`, `email:dlq`, sessions admin | `infra-redis` de la 38, **base logique dédiée** | `REDIS_URL` |

L'ancien `docker-compose.prod.yml` (serveur + client + `notifier-redis` sur la
24) ne sert plus qu'au retour arrière.

## Chemin d'une requête

```
appelant → https://backend.flyasky.com/notifier/v1/send (VIP Kemp)
  → gateway 10.20.10.6:8082 → préfixe /notifier retiré → http://192.168.24.38:8055/v1/send

navigateur (notifier.flyasky.com) → https://backend.flyasky.com/notifier/auth/...
  → même chemin ; cookie `session` limité à Path=/notifier
```

- Au catalogue : `notifier`, `origin = manual`, `kind = internal` (une cible `external` doit être publique), préfixe
  `/notifier`, `strip_prefix = true`, `strict_mode = false`, **sans import
  OpenAPI** (sinon des routes passeraient en `requireAuth` et le gateway
  renverrait 401 avant le notifier).
- Le gateway ne vérifie rien : le notifier authentifie lui-même ses appelants
  (`Authorization: Bearer <clé>` ou HMAC `X-Consumer-ID` / `X-Timestamp` /
  `X-Signature`). La signature HMAC ne porte pas sur le chemin, le retrait du
  préfixe ne la casse pas.
- Ne pas faire utiliser `X-Api-Key` aux appelants : le gateway consomme cet
  en-tête.
- `notifier-server.flyasky.com` reste servi pendant la transition par le relais
  de la 24, qui renvoie tout vers `192.168.24.38:8055` sans préfixe.

## Variables (`infra/notifier`, 38)

Reprendre le `.env` de production actuel (Dokploy de la 24), en changeant :

| Variable | Valeur |
|---|---|
| `PORT` | `8055` |
| `DATABASE_URL` | `postgres://<rôle>:<MDP>@172.18.1.23:5432/notifier?sslmode=disable` |
| `REDIS_URL` | `redis://default:<MDP encodé>@192.168.24.38:6379/<base dédiée>` |
| `CORS_ORIGIN` | `https://notifier.flyasky.com` |
| `SESSION_COOKIE_PATH` | non définie (`/`) tant que l'UI de la 24 passe par `notifier-server.flyasky.com` ; `/notifier` quand elle passera par le gateway |
| `HMAC_MASTER_KEY` | **valeur actuelle, à l'identique** (chiffre les secrets HMAC en base) |
| `SMTP_*`, `WORKER_COUNT`, `MAX_RETRIES` | inchangés |
| `REDIS_PASSWORD`, `CLIENT_PORT`, `VITE_API_URL` | inutiles ici |

UI (`infra/notifier-ui`, 24) : `VITE_API_URL=https://backend.flyasky.com/notifier`
(valeur par défaut du compose), `CLIENT_PORT` = port actuel de l'ancien client.
`VITE_API_URL` est figée à la construction : redéployer pour la changer.

Le gateway doit aussi accepter l'origine de l'UI : ajouter
`https://notifier.flyasky.com` à `CORS_ORIGIN` de `gateway-v2`.

## Retour arrière

1. Arrêter `infra/notifier` (38), `infra/notifier-ui` et le relais (24).
2. `docker start` des anciens conteneurs de la 24 (`notifier-server`,
   `notifier-client`, `notifier-redis`) : ils pointent toujours sur
   `172.18.1.21` et `notifier-redis`.
3. Les envois acceptés depuis la bascule ne sont que dans `172.18.1.23` et le
   Redis de la 38 : les rejouer ou les accepter comme perdus.
