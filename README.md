# pm — lancer un programme et l'encadrer

`pm` est un exécutable Windows unique, sans dépendance, qui fait deux choses :

- **`pm run`** : des actions avant le lancement d'un programme, puis d'autres après sa fermeture (copier des fichiers de profil, nettoyer, lancer une commande).
- **`pm watch`** : garder un programme en vie (watchdog), avec une interface en terminal pour le mettre en pause, le relancer ou l'arrêter.

Les deux partagent la même config YAML et le même moteur de lancement.

La version 2 est une réécriture complète en Go de l'ancien `process-monitor` en Python (1.x, dans l'historique git).
Le format de config a changé : voir [Migration depuis la 1.x](#migration-depuis-la-1x).

## Sommaire

- [Installation](#installation)
- [Utilisation](#utilisation)
- [Comment pm suit un programme](#comment-pm-suit-un-programme)
- [Configuration](#configuration)
- [Garde-fous](#garde-fous)
- [Journaux](#journaux)
- [Migration depuis la 1.x](#migration-depuis-la-1x)
- [Développement](#développement)

## Installation

Il faut Go 1.27 ou plus récent (`scoop install go`).

```bash
go build -o dist/pm.exe ./cmd/pm
```

`dist/pm.exe` se copie où l'on veut : il n'a besoin de rien d'autre.
Une compilation locale affiche la version `dev` ; les versions publiées sont sur la page [Releases](https://github.com/davidp57/process-monitor/releases).

## Utilisation

```
pm run   <config.yaml> [--profile NOM] [--tray]
pm watch <config.yaml> [--profile NOM] [--tray | --no-tui]
pm check <config.yaml> [--profile NOM]
pm version
```

`pm check` ne lance rien : il valide la config et affiche ce qui sera copié, supprimé ou exécuté, en signalant les fichiers source absents (lignes `sync` et `copy` des actions *avant*).
À passer après chaque modification de config.

### pm run

Déroulé : verrou de session → actions *avant* → lancement → attente de la fin → actions *après*.

La console affiche chaque étape.
Si une action *avant* échoue, rien n'est lancé, et les fichiers live déjà remplacés par le profil sont remis comme ils étaient.
Chaque commande `exec` est limitée à `hooks.timeout` (10 minutes par défaut). Un programme qu'elle laisse tourner en arrière-plan ne bloque pas pm.
`pm run` rend le code de sortie du programme suivi ; 1 si pm a rencontré une erreur ou s'est détaché (la copie retour n'a pas eu lieu) ; 2 pour une erreur de config ou de ligne de commande.
Si une action *après* échoue, les suivantes sont quand même exécutées, et la fenêtre reste ouverte (`exit.wait_key: on-error`, le défaut) pour qu'on voie l'erreur.

**Ctrl+C** ne coupe pas tout de suite : il faut un second Ctrl+C dans les 5 secondes, et la console dit d'abord ce qu'il fera, selon le moment :

- pendant les copies de départ, il **annule** : rien n'est lancé, les fichiers live sont remis comme avant ;
- pendant que le programme tourne, il **détache** pm, et le programme continue sans lui ;
- en `watch`, pendant le délai avant une relance, rien ne tourne : il **termine** la session, actions *après* comprises.

Les actions *après* ne sont alors pas exécutées, et le journal le dit.
Fermer la fenêtre de la console a le même effet, sans confirmation possible : **ne pas fermer la fenêtre de pm pendant la session**.

### pm watch

Relance le programme à chaque arrêt, selon `watch.restart`.
Un arrêt moins de 10 secondes après le lancement compte comme un plantage en boucle : le premier attend `watch.delay`, chaque suivant double le délai, jusqu'à `watch.max_delay`.
`watch.max_restarts` ne compte que les relances automatiques, pas celles demandées avec `r`.
Les lignes `sync` sont copiées une fois au début et recopiées une fois à la fin de la session.
À chaque fin de lancement, pm arrête ce qui reste du groupe de process avant de relancer, pour que les restes ne s'empilent pas d'un lancement à l'autre.
Les autres actions *avant* (un nettoyage, par exemple) sont rejouées avant chaque relance, comme en 1.x ; les actions *après* ne s'exécutent qu'à la fin.

Touches de l'interface :

| Touche | Effet |
|---|---|
| `p` | met en pause / reprend le programme (tous ses process) |
| `r` | arrête et relance le programme |
| `k` | arrête le programme, exécute les actions *après*, quitte |
| `q` deux fois | détache : pm quitte, le programme continue, les actions *après* ne sont **pas** exécutées. Au démarrage, `q` deux fois annule (rien n'est lancé, fichiers live remis comme avant). Pendant le délai avant une relance, rien ne tourne : `q` deux fois termine alors la session, actions *après* comprises |
| flèches, PgUp/PgDn | font défiler le journal |

`--no-tui` remplace l'interface par un journal en texte simple, par exemple pour un lancement en service.

### Sans fenêtre : l'icône de notification

`--tray`, ou `ui: tray` dans la config, remplace la fenêtre par une icône dans la zone de notification, pour `pm run` comme pour `pm watch`.
Lancé depuis un raccourci, pm rend sa fenêtre console dès le démarrage (elle apparaît une fraction de seconde) ; lancé depuis un terminal, il continue d'y écrire.

- **Couleur** : verte quand le programme tourne, orange pendant les copies (au début et à la fin) ou avant une relance, grise en pause, rouge après une erreur. L'infobulle donne la session, le profil et l'état.
- **Menu** : ouvrir le journal ; arrêter et finir (actions *après* exécutées) ; détacher, après une confirmation qui dit ce que ça coûte selon le moment (annuler au démarrage, terminer pendant le délai de relance, sinon laisser le programme sans copie retour). En `watch` : pause, reprise, relance.
- **À la fin** : si la session a échoué, une fenêtre Windows dit ce qui a échoué et où est le journal ; sinon l'icône disparaît.

- **Image** : `--icon FICHIER`, ou `icon:` dans la config, ou par profil dans `profiles:`, remplace le disque par ton image (`.png`, ou `.ico` en 32 bits ou PNG intégré) ; une petite pastille de la couleur de l'état est posée en bas à droite. Une image illisible est signalée dans le journal, et pm garde le disque.
- **Sans fenêtres parasites** : les commandes `exec` ne s'ouvrent pas dans une fenêtre console, ni le programme lancé quand pm n'a plus de console.

Lancé depuis un raccourci, il n'y a plus de fenêtre à fermer par mégarde : on ne quitte que par le menu. Lancé depuis un terminal, le double Ctrl+C reste la seule façon de couper pm, comme en mode console.

## Comment pm suit un programme

Le programme est lancé dans un **Job Object** Windows : tous les process qu'il crée y restent rattachés.
Sans `track`, la session se termine quand le dernier process du groupe s'est arrêté.

Beaucoup de programmes passent par un lanceur : `dcs_updater.exe` démarre `dcs.exe` puis se ferme.
`track: dcs.exe` dit à pm de suivre ce process-là, par son handle, et non le lanceur :

- pm le trouve dans le groupe dès sa création ;
- s'il a quitté le groupe, pm le cherche par son nom parmi les process créés après le lancement ;
- quand il s'arrête, pm attend encore 3 secondes, au cas où il passerait la main à un autre process du même nom (le Notepad de Windows 11 fait ainsi) ;
- si le **lanceur** tourne encore (pour relancer le jeu après une mise à jour, par exemple), pm l'attend et le signale dans le journal ;
- les autres process que le jeu a lancés (un client radio comme SRS, un overlay) ne sont **pas** attendus : le journal les nomme, et `kill_on_exit` peut les arrêter ;
- s'il n'apparaît pas dans les 10 secondes qui suivent la fin de tout le reste, la session se termine en erreur.

## Configuration

Les exemples complets sont dans [`examples/`](examples/).

```yaml
name: dcs                       # nom de la session et du verrou ; défaut : nom du fichier
vars:                           # variables, utilisables partout sous la forme ${nom}
  saved: ${env:USERPROFILE}\Saved Games\DCS
  prof: ${saved}\Profiles\${profile}

command: C:\jeux\DCS World\bin\dcs_updater.exe
args: []
workdir: ${prof}
env: { CLE: valeur }            # ajoutées à l'environnement de pm
track: dcs.exe                  # process à suivre à la place du programme lancé
kill_on_exit: [SR-ClientRadio.exe]  # arrêtés quand le process suivi s'est terminé, où qu'ils tournent
if_running: abort               # abort (défaut) ou kill, voir Garde-fous
capture: false                  # sortie du programme dans le journal (et pas de fenêtre console)
ui: console                     # console (défaut) ou tray : une icône de notification, comme --tray
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
    hooks:                      # s'ajoutent aux hooks communs
      sync: [ '${prof}\joy.lua <-> ${saved}\Input\joy.lua' ]
  VR: {}

log: { dir: '${config_dir}\logs', keep: 5 }
watch: { restart: always, delay: 2s, max_delay: 1m, max_restarts: 0 }
exit: { delay: 0s, wait_key: on-error }   # never | on-error | always
```

Points à connaître :

- **Variables** : `${nom}` lit `vars`, qui peuvent se référencer entre elles. Sont aussi disponibles `${profile}`, `${name}`, `${config_dir}` (dossier du fichier de config) et `${env:NOM}`. Une variable inconnue est une erreur.
- **Clés inconnues** : une faute de frappe dans une clé est une erreur, jamais une option ignorée.
- **Chemins** : entre apostrophes `'...'` YAML ne transforme pas les `\`. Entre guillemets `"..."`, il faut les doubler.
- **Chemins relatifs** : résolus depuis le dossier du fichier de config. Pour `command` et le programme d'un `exec`, un nom seul (`cmd`, `notepad`) est cherché dans le `PATH`, un chemin (`tools\app.exe`) part du dossier de la config. Une commande `exec` s'exécute elle-même dans le dossier de la config : ses arguments relatifs en partent aussi. Le programme lancé (`command`) part de `workdir`, ou à défaut du dossier courant de pm.
- **`track`** : un nom de process seul, extension comprise (`dcs.exe`).
- **`sync`** : un même fichier ne peut apparaître que dans une seule ligne, côté profil comme côté live.
- **Un seul document** : un `---` qui couperait le fichier en deux est une erreur, pas une seconde moitié ignorée.
- **Ordre** : avant = `sync` (aller) puis `before` ; après = `sync` (retour) puis `after`.
- **Copies** : écrites dans un fichier temporaire puis renommées, donc jamais à moitié écrites. L'ancien fichier est gardé en `.bak`.
- **ready** : pm supprime l'ancien fichier avant le lancement (un reste de crash le ferait croire prêt), puis signale quand il réapparaît, ou qu'il n'est pas apparu dans le délai. C'est une information : la fin de session dépend du process suivi, pas de ce fichier.

## Garde-fous

- **Verrous** : une session prend un verrou à son `name`, et un par fichier live de ses lignes `sync`. Deux sessions de même nom, ou deux configs différentes qui écrivent le même fichier live, ne peuvent donc pas tourner ensemble. Le second lancement est refusé avant toute copie, avec le profil de la session en cours. Windows libère les verrous à la sortie de pm, même après un crash : ils ne restent jamais bloqués.
- **Programme déjà lancé hors de pm** (`if_running`), vérifié sur le process suivi **et** sur le lanceur :
  - `abort` (défaut) : refus, rien n'est copié ni lancé. C'est ce qui protège un profil d'être écrasé par les réglages d'un autre.
  - `kill` : les instances existantes sont arrêtées avant le lancement, le lanceur d'abord (sinon il pourrait relancer le jeu), puis pm vérifie que plus rien ne tourne.
- **Dernier filet** (avec `track`) : si, au moment des actions de fin, le process suivi tourne encore sans être suivi, ou si la liste des process est illisible, les actions de fin ne sont pas exécutées et la session finit en erreur. Mieux vaut une copie retour manquée, signalée, qu'un profil écrasé pendant que le jeu tourne.
- **Arrêt** : `k` arrête les process du groupe et le process suivi. Si celui-ci avait quitté le groupe, c'est le process de ce nom créé après le lancement que pm a adopté.

## Journaux

Chaque session écrit `<name>[-<profil>]-<date>-<heure>.log` dans `log.dir` : étapes de pm, copies, sortie du programme si `capture`.
Seuls les `log.keep` plus récents sont gardés.

## Migration depuis la 1.x

| 1.x | 2.x |
|---|---|
| `process-monitor.exe --config x.yaml` | `pm run x.yaml` (lancer une fois) ou `pm watch x.yaml` (relancer) |
| `process_name` | `track` (ou rien si c'est le programme lancé) |
| `executable` | `command` |
| `working_dir` | `workdir` |
| `monitor_process: false` / `true` | `pm run` / `pm watch` |
| `kill_existing: true` | `if_running: kill` |
| `lock_file` | `ready.file`, avec un délai |
| `initial_file_copies` + `final_file_copies` en miroir | une ligne `sync` par fichier |
| `initial_file_cleanup` / `final_file_cleanup` | `hooks.before` / `hooks.after` avec `delete` |
| `capture_output` + `output_log` | `capture` + `log` |
| `log_cleanup_*` | `log.keep` |
| `exit_delay`, `wait_keypress` | `exit.delay`, `exit.wait_key` |
| `interval`, `startup_delay` | supprimés : pm ne sonde plus, il attend la fin du process |
| `tui`, `verbose` | `pm watch` a l'interface, `pm run` la console ; tout va au journal |

Comportements changés :

- `q` ne tue plus le programme : il détache, après confirmation. `k` arrête.
- Les actions de fin s'exécutent une seule fois (la 1.x les jouait jusqu'à trois fois).
- Un programme déjà lancé n'est plus « adopté » : pm refuse (ou l'arrête avec `if_running: kill`).
- Une action *avant* en échec empêche le lancement (la 1.x le journalisait et lançait quand même).
- `exit.wait_key` s'applique aussi à l'interface de `pm watch` (la 1.x ignorait `wait_keypress` en mode TUI).
- La copie retour attend la fin du jeu et de son lanceur, pas celle des autres process qu'il a lancés (la 1.x n'attendait que le jeu).
- En `pm watch`, chaque relance rejoue les actions *avant* mais pas les copies `sync` ; la 1.x recopiait aussi ses `initial_file_copies` à chaque relance, ce qui écrasait les réglages modifiés pendant le lancement précédent.

## Développement

Le fonctionnement du dépôt (tickets, branches, revue, releases) est décrit dans [`CLAUDE.md`](CLAUDE.md), le vocabulaire dans [`CONTEXT.md`](CONTEXT.md), les décisions dans [`docs/adr/`](docs/adr/).
Les tickets sont les [issues GitHub](https://github.com/davidp57/process-monitor/issues).

Ce que vérifie la CI à chaque PR :

```bash
gofmt -l .
go vet ./...
go test -race ./...
```

Une release se fait avec la commande `/release` : un tag `vX.Y.Z` sur `main` compile `pm.exe` et publie la Release GitHub, avec `RELEASE_NOTES.md` pour texte.

Les tests de `internal/proc` et `internal/session` lancent de vrais process : des copies du binaire de test, sans fenêtre.
Ils couvrent le passage de main d'un lanceur à son programme, un programme qui quitte le Job Object, et deux process de même nom.

Organisation :

| Paquet | Rôle |
|---|---|
| `cmd/pm` | ligne de commande, console de `pm run` |
| `internal/config` | lecture, profils, variables, validation |
| `internal/hooks` | copies, suppressions, commandes, `sync` |
| `internal/proc` | lancement en Job Object, suivi du process, pause, arrêt |
| `internal/session` | verrou, enchaînement avant / lancement / relances / après |
| `internal/lock` | verrou de session |
| `internal/logx` | journal fichier + écran |
| `internal/tui` | interface de `pm watch` |

Windows uniquement : `internal/proc` et `internal/lock` reposent sur l'API Windows.

## Licence

MIT
