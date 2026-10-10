# Utiliser le notifier — guide d'intégration

Le notifier envoie des e-mails pour le compte d'autres services. Un service qui
l'utilise est un **consommateur** : il appelle l'API HTTP, le notifier met
l'e-mail en file et un worker l'envoie par SMTP.

## Ce dont vous avez besoin

| Besoin | Détail |
|---|---|
| Un consommateur | Créé par un administrateur dans l'UI (`https://notifier.flyasky.com`). Fournir : un **nom** (2–100 caractères) et un **préfixe d'adresse** (2–100 caractères). |
| Une adresse d'expéditeur | Déduite du préfixe : `<préfixe>@flyasky.com`. Elle est fixe, l'appelant ne la choisit pas. |
| Des identifiants | Remis **une seule fois** à la création : une clé d'API `nk_…`, et un secret HMAC `nsk_…` si vous choisissez la signature. Conservez-les dans un gestionnaire de secrets. |
| Un accès réseau | `https://backend.flyasky.com/notifier` (gateway). Aucune autre URL à connaître. |

Rien d'autre : pas de compte sur le gateway, pas de jeton `auth`. Le notifier
authentifie lui-même ses appelants, le gateway ne vérifie rien.

## Envoyer un e-mail

`POST https://backend.flyasky.com/notifier/v1/send`

```json
{
  "to": ["destinataire@exemple.com"],
  "subject": "Sujet du message",
  "body": "<p>Contenu <strong>HTML</strong></p>"
}
```

| Champ | Règle |
|---|---|
| `to` | Liste d'au moins une adresse valide. |
| `subject` | Obligatoire, 998 caractères au plus. |
| `body` | Obligatoire. **Envoyé en HTML** (`text/html; charset=UTF-8`) : échappez le texte brut venant d'un utilisateur. |
| Taille | 10 Mio maximum pour la requête. |

Réponse (`200`) : le message est accepté et mis en file, **pas encore livré**.

```json
{ "job_id": "8f1c…", "status": "queued" }
```

### Authentification, deux modes au choix

**1. Clé d'API (le plus simple)**

```bash
curl -X POST https://backend.flyasky.com/notifier/v1/send \
  -H "Authorization: Bearer $NOTIFIER_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"to":["destinataire@exemple.com"],"subject":"Test","body":"<p>Bonjour</p>"}'
```

N'utilisez **pas** l'en-tête `X-Api-Key` : le gateway le consomme.

**2. Signature HMAC (le secret ne circule pas)**

En-têtes : `X-Consumer-ID` (l'identifiant du consommateur),
`X-Timestamp` (secondes Unix), `X-Signature`.

- Message signé : `<consumer_id>:<timestamp>:<corps JSON canonique>`
- Algorithme : HMAC-SHA256 avec le secret `nsk_…`, résultat en hexadécimal.
- JSON canonique : clés triées par ordre alphabétique à tous les niveaux,
  sans espace, chaînes encodées en JSON.
- Tolérance d'horloge : ±300 s. Synchronisez l'heure (NTP).
- Si `X-Consumer-ID` est présent, la clé d'API est ignorée.
- Le chemin n'est pas signé : le gateway peut retirer le préfixe `/notifier`.

```js
import { createHmac } from "node:crypto";

const canonical = (v) =>
  Array.isArray(v) ? `[${v.map(canonical).join(",")}]`
  : v && typeof v === "object"
    ? `{${Object.keys(v).sort().map((k) => `${JSON.stringify(k)}:${canonical(v[k])}`).join(",")}}`
    : JSON.stringify(v);

const body = { to: ["destinataire@exemple.com"], subject: "Test", body: "<p>Bonjour</p>" };
const ts = Math.floor(Date.now() / 1000);
const signature = createHmac("sha256", process.env.NOTIFIER_HMAC_SECRET)
  .update(`${process.env.NOTIFIER_CONSUMER_ID}:${ts}:${canonical(body)}`)
  .digest("hex");

await fetch("https://backend.flyasky.com/notifier/v1/send", {
  method: "POST",
  headers: {
    "Content-Type": "application/json",
    "X-Consumer-ID": process.env.NOTIFIER_CONSUMER_ID,
    "X-Timestamp": String(ts),
    "X-Signature": signature,
  },
  body: JSON.stringify(body),
});
```

## Suivre un envoi

`GET https://backend.flyasky.com/notifier/v1/jobs/<job_id>` avec la même
authentification.

| `status` | Sens |
|---|---|
| `pending` | En file ou en cours de nouvelle tentative. |
| `delivered` | Remis au serveur SMTP (`delivered_at` renseigné). |
| `failed` | Échec après toutes les tentatives (`error` donne la cause). |
| `bounced` | Rejeté par le destinataire ou son serveur. |

La réponse d'envoi dit `queued` ; le job lui-même est `pending` jusqu'à la
livraison. Le nombre de tentatives dépend de la configuration du notifier.

## Limites et erreurs

| Code | Cause | Que faire |
|---|---|---|
| `200` | Accepté et mis en file. | — |
| `400` / `422` | Corps invalide (adresse, sujet ou corps manquant). | Corriger la requête, ne pas réessayer telle quelle. |
| `401` | Clé inconnue, signature fausse, horloge décalée, en-têtes absents. | Vérifier les identifiants et l'heure. |
| `403` | Consommateur **suspendu**. | Contacter l'administrateur. |
| `429` | Trop de requêtes : 60 par minute et par consommateur, 120 par minute et par IP. En-tête `Retry-After: 60`. | Attendre, puis réessayer avec un délai croissant. |
| `502` / `503` | Notifier injoignable ou en cours de redémarrage. | Réessayer avec un délai croissant. |

**Suspension automatique** : un consommateur dont plus de 20 % des e-mails
sont rejetés (`bounced`), après au moins 10 envois, est suspendu. Envoyez
uniquement à des adresses valides et connues.

**Doublons** : le notifier ne déduplique pas. Si votre appel échoue sans
réponse (délai dépassé), un réessai peut envoyer le message deux fois.
Réessayez seulement sur `429`, `502` et `503`.

## Bonnes pratiques

- Une clé par service ; ne la partagez pas entre applications.
- Ne mettez jamais la clé ou le secret dans un front ou un dépôt.
- Fixez un délai d'attente de quelques secondes sur l'appel.
- Pour une rotation, demandez à l'administrateur de régénérer les identifiants
  dans l'UI ; l'ancienne clé cesse de fonctionner.
