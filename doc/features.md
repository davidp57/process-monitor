# Fonctions

Chaque fonction se règle dans le fichier de configuration (YAML). Les extraits ci-dessous se combinent dans un même fichier ; la page [Configuration](configuration.md) liste toutes les clés, et les [exemples](examples/dcs.md) montrent des configs complètes.

## Le programme à lancer {#program}

```yaml
command: C:\jeux\DCS World\bin\dcs_updater.exe   # le programme
args: [--quiet]                                  # ses arguments (facultatif)
workdir: C:\Users\moi\Saved Games\DCS             # son dossier de travail (facultatif)
env: { MA_VARIABLE: valeur }                     # des variables d'environnement en plus (facultatif)
```

Un nom seul (`notepad`) est cherché dans le `PATH` ; un chemin relatif part du dossier de la config.

## Profils et fichiers synchronisés {#sync}

**Quand** : un même programme, plusieurs jeux de réglages.

Une ligne `sync` relie un fichier du profil (à gauche) et le fichier que le programme utilise vraiment, le fichier « live » (à droite). Au lancement, le fichier du profil est copié sur le live ; à la fermeture, le live est recopié dans le profil, avec les changements faits pendant la session.

```yaml
vars:
  saved: ${env:USERPROFILE}\Saved Games\DCS
  prof: ${saved}\Profiles\${profile}

hooks:
  sync:
    - ${prof}\options.lua <-> ${saved}\Config\options.lua

profiles:
  DEV: {}
  VR: {}
```

- `--profile DEV` choisit le profil ; `${profile}` vaut alors `DEV`.
- Un profil peut ajouter ses propres lignes, variables ou icône :

    ```yaml
    profiles:
      DEV:
        hooks:
          sync:
            - '${prof}\joystick.diff.lua <-> ${saved}\Config\Input\joystick.diff.lua'
    ```

- Chaque fichier écrasé est gardé en `.bak`, et une copie n'est jamais à moitié écrite.

## Actions avant et après {#actions}

**Quand** : préparer ou nettoyer autour du programme.

```yaml
hooks:
  before:                       # avant le lancement, après les copies sync
    - delete: ${prof}\cache     # un fichier ou un dossier ; absent = rien à faire
    - copy: { from: modele.ini, to: '${saved}\config.ini' }
    - exec: [cmd, /c, echo, prêt]
  after:                        # après l'arrêt, après la récupération sync
    - delete: ${prof}\cache
  timeout: 10m                  # durée maximale d'une commande exec
```

- Si une action *avant* échoue, rien n'est lancé, et les fichiers live déjà remplacés par les lignes `sync` sont remis comme avant. Les actions `copy` et `delete` déjà faites, elles, ne sont pas défaites (un `copy` laisse l'ancien fichier en `.bak`).
- Si une action *après* échoue, les suivantes s'exécutent quand même.
- Une commande `exec` s'exécute dans le dossier de la config, sans ouvrir de fenêtre ; sa sortie va dans le journal.

## Suivre le vrai programme {#track}

**Quand** : le programme passe par un lanceur, qui démarre le vrai programme puis se ferme.

```yaml
command: C:\jeux\DCS World\bin\dcs_updater.exe
track: dcs.exe
```

Process Monitor suit `dcs.exe`, pas le lanceur : la session dure tant que le jeu tourne, et les réglages ne sont récupérés qu'une fois le jeu fermé. Le détail est sur la page [Comment pm suit un programme](tracking.md).

## Garde-fous {#guards}

**Quand** : toujours ; ils sont actifs par défaut.

```yaml
name: dcs              # tous les profils de cette config partagent ce verrou
if_running: abort      # abort (défaut) : refuser si le jeu tourne déjà ; kill : l'arrêter d'abord
```

- Deux sessions de même nom ne tournent jamais ensemble : lancer le profil VR pendant que DEV tourne est refusé, avant toute copie.
- Si le jeu, ou son lanceur, tourne déjà hors de Process Monitor, `abort` refuse de démarrer ; `kill` l'arrête, puis démarre.

## Relances {#watch}

**Quand** : un programme à garder en vie. Se lance avec `pm watch` au lieu de `pm run`.

```yaml
watch:
  restart: always       # always | on-failure | never
  delay: 2s             # attente avant une relance
  max_delay: 1m         # plafond, si le programme plante en boucle
  max_restarts: 0       # 0 = sans limite
```

Les actions *avant* (un nettoyage) sont rejouées avant chaque relance ; les lignes `sync` ne sont copiées qu'au début et à la fin. Voir [pm watch](watch.md).

## Programmes annexes {#kill-on-exit}

**Quand** : le programme en ouvre d'autres qui n'ont plus de raison d'être après lui.

```yaml
kill_on_exit: [SR-ClientRadio.exe]
```

Avec `track`, Process Monitor n'attend pas ces programmes pour récupérer les réglages, et ceux de la liste sont arrêtés quand le programme suivi s'est terminé. Sans `track`, la session attend la fin de tous les programmes lancés, ceux-là compris : `kill_on_exit` ne les arrête qu'ensuite, donc ne sert qu'avec `track`.

## Partager les cœurs du processeur {#affinity}

**Quand** : deux programmes lourds tournent ensemble, et l'un ne doit pas prendre de temps de calcul sur les cœurs de l'autre. Par exemple DCS World sur certains cœurs, et SteamVR sur les autres.

```yaml
affinity:
  cores: 0-11            # le programme lancé, et tout ce qu'il démarre
  others:                # d'autres process, trouvés par leur nom
    cores: 12-15
    processes: [vrserver.exe, vrcompositor.exe, vrmonitor.exe]
```

Les cœurs s'écrivent en liste : `0-11`, ou `0-5,8,10-11`. Ce sont les numéros du Gestionnaire des tâches (vue « processeurs logiques »).

**Le programme lancé** reçoit ses cœurs avant sa première instruction : Process Monitor le crée suspendu, pose ses cœurs, puis le laisse démarrer. Un programme qui lit ses cœurs au démarrage voit donc déjà les bons. Ce qu'il lance en hérite : un lanceur qui passe la main au jeu, ou le jeu qui se relance lui-même. Avec `pm watch`, chaque relance reçoit les mêmes cœurs.

**Les autres process** sont placés sur leurs cœurs dès qu'ils apparaissent (Process Monitor regarde chaque seconde), et de nouveau s'ils redémarrent. Ceux qui tournent déjà avant le lancement sont déplacés avant que le programme démarre. À la fin de la session, chacun retrouve les cœurs qu'il avait. Après un détachement (`d`), ils gardent leurs cœurs, puisque le programme tourne encore.

**Dans un profil**, un bloc `affinity` remplace celui de la config : le partage peut ne servir qu'au profil VR.

À savoir :

- Windows règle les cœurs process par process : on ne peut pas réserver des cœurs à un programme. Pour les libérer, il faut aussi déplacer les autres, d'où `others`.
- Les cœurs d'un programme déjà lancé ne sont jamais changés. Si le process suivi (`track`) n'a pas reçu les cœurs (démarré par un autre programme, comme Steam), le journal le signale et Process Monitor le laisse tel quel : changer ses cœurs en cours de route écraserait ceux que chacun de ses threads a choisis.
- Sur un processeur à SMT, deux numéros voisins (`12, 13`) sont souvent le même cœur physique. Donner des paires entières aux autres process évite qu'ils partagent un cœur physique avec le programme.
- `pm check` refuse une liste vide ou mal écrite, et un cœur que la machine n'a pas. Il signale aussi des cœurs donnés à la fois au programme et aux autres.
- Le gain dépend du processeur et de la charge ; il peut être nul. L'[exemple DCS](examples/dcs.md#steamvr) propose une façon de le mesurer.

## L'icône de notification {#tray}

**Quand** : pas de fenêtre console, ou pour voir l'état d'un coup d'œil.

```yaml
ui: tray                       # ou --tray sur la ligne de commande
icon: Icons\dcs.ico            # facultatif ; ou --icon, ou par profil
```

L'icône est verte quand le programme tourne, orange pendant les copies ou avant une relance, rouge après une erreur ; son menu ouvre le journal, arrête ou détache. Voir [L'icône de notification](tray.md).

## Journaux et langue {#logs}

```yaml
capture: true                       # la sortie du programme dans le journal
log: { dir: '${config_dir}\logs', keep: 5 }
lang: fr                            # fr ou en ; --lang et PM_LANG l'emportent ; sans rien, celle des réglages de l'interface, puis celle de Windows
```

Chaque session écrit son journal : étapes, copies, erreurs, et la sortie du programme avec `capture`. Seuls les `keep` plus récents sont gardés.

## Fin de session {#exit}

```yaml
exit: { delay: 2s, wait_key: on-error }   # wait_key : never | on-error | always
```

En console, `wait_key: on-error` garde la fenêtre ouverte en cas d'erreur pour qu'on la lise. Avec l'icône, une erreur ouvre une fenêtre Windows qui la décrit.
