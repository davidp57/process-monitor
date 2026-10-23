# Configuration

Une config est un fichier YAML. Des exemples complets sont dans le dossier [`examples`](https://github.com/davidp57/process-monitor/tree/main/examples) du dépôt.

## Toutes les clés {#keys}

```yaml
name: dcs                       # nom de la session et du verrou ; défaut : nom du fichier
lang: fr                        # langue de pm : fr ou en (voir Installation)
vars:                           # variables, utilisables partout sous la forme ${nom}
  saved: ${env:USERPROFILE}\Saved Games\DCS
  prof: ${saved}\Profiles\${profile}

command: C:\jeux\DCS World\bin\dcs_updater.exe
args: []
workdir: ${prof}
env: { CLE: valeur }            # ajoutées à l'environnement de pm
track: dcs.exe                  # process à suivre à la place du programme lancé
kill_on_exit: [SR-ClientRadio.exe]  # arrêtés quand le process suivi s'est terminé, où qu'ils tournent
if_running: abort               # abort (défaut) ou kill
capture: false                  # sortie du programme dans le journal (et pas de fenêtre console)
ui: console                     # console (défaut) ou tray : l'icône de notification
icon: Icons\dcs.ico             # image de l'icône de notification ; aussi par profil
ready: { file: '${saved}\dcs.lock', timeout: 3m }
backup: true                    # garde un .bak de chaque fichier écrasé (défaut)

hooks:
  sync:                         # "profil <-> live" : copié avant, recopié après
    - ${prof}\options.lua <-> ${saved}\Config\options.lua
  before:
    - copy: { from: a.txt, to: b.txt }
    - delete: ${prof}\cache     # fichier ou dossier ; absent = rien à faire
    - exec: [cmd, /c, echo, ok]
  after: []
  timeout: 10m                  # durée maximale d'une commande exec (défaut)

profiles:                       # pm run ... --profile DEV
  DEV:
    vars: { autre: valeur }     # remplace ou ajoute des variables
    icon: Icons\dcs-dev.ico     # remplace l'icône
    hooks:                      # s'ajoutent aux hooks communs
      sync: [ '${prof}\joy.lua <-> ${saved}\Input\joy.lua' ]
  VR: {}

log: { dir: '${config_dir}\logs', keep: 5 }
watch: { restart: always, delay: 2s, max_delay: 1m, max_restarts: 0 }
exit: { delay: 0s, wait_key: on-error }   # never | on-error | always
```

## Points à connaître {#rules}

- **Variables** : `${nom}` lit `vars`, qui peuvent se référencer entre elles. Sont aussi disponibles `${profile}` (s'il y a des profils), `${name}`, `${config_dir}` (dossier du fichier de config) et `${env:NOM}`. Une variable inconnue est une erreur.
- **Clés inconnues** : une faute de frappe dans une clé est une erreur, jamais une option ignorée.
- **Chemins** : entre apostrophes `'...'` YAML ne transforme pas les `\` ; entre guillemets `"..."`, il faut les doubler.
- **Chemins relatifs** : résolus depuis le dossier de la config. Pour `command` et le programme d'un `exec`, un nom seul (`cmd`) est cherché dans le `PATH`, un chemin (`tools\app.exe`) part du dossier de la config. Une commande `exec` s'exécute dans le dossier de la config ; le programme lancé part de `workdir`, ou à défaut du dossier courant.
- **`track`** et **`kill_on_exit`** : des noms de process seuls, extension comprise (`dcs.exe`).
- **`sync`** : un même fichier ne peut apparaître que dans une seule ligne, côté profil comme côté live.
- **Un seul document** : un `---` qui couperait le fichier en deux est une erreur.
- **Ordre** : avant = `sync` (aller) puis `before` ; après = `sync` (retour) puis `after`.
- **ready** : pm supprime l'ancien fichier avant le lancement, puis signale quand il réapparaît, ou qu'il n'est pas apparu dans le délai. C'est une information : la fin de session dépend du process suivi.
