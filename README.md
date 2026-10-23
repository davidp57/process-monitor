# <a id="en"></a>pm — run a program and frame it

> 🇫🇷 [Lire en français](#fr)

`pm` is a single Windows executable, with no dependency:

- **`pm run`** runs actions before a program starts and after it exits — copy a profile's files in and back, clean up, run a command;
- **`pm watch`** keeps a program alive and restarts it when it stops;
- **`pm check`** shows what a config would do, without running anything.

A session shows itself in a console, a terminal interface or an icon in the notification area, in French or English.

📖 **Documentation: <https://davidp57.github.io/process-monitor/latest/en/>**

## Install

Download `pm.exe` from the [Releases](https://github.com/davidp57/process-monitor/releases) page, or build it with Go 1.27+:

```powershell
go build -o dist\pm.exe .\cmd\pm
```

## Quick start

```powershell
.\pm.exe check .\dcs.yaml --profile DEV   # what would be copied, deleted, run
.\pm.exe run .\dcs.yaml --profile DEV     # before actions, launch, wait, after actions
.\pm.exe watch .\app.yaml --tray          # keep a program alive, with a notification icon
```

See the [configuration reference](https://davidp57.github.io/process-monitor/latest/en/configuration/) and the [`examples`](examples/) folder.

## Contributing

How the repository works (issues, branches, review, releases) is in [`CLAUDE.md`](CLAUDE.md); the vocabulary in [`CONTEXT.md`](CONTEXT.md); the decisions in [`docs/adr/`](docs/adr/). Issues are the [GitHub issues](https://github.com/davidp57/process-monitor/issues). Windows only.

License: MIT.

---

# <a id="fr"></a>pm — lancer un programme et l'encadrer

> 🇬🇧 [Read in English](#en)

`pm` est un exécutable Windows unique, sans dépendance :

- **`pm run`** exécute des actions avant le lancement d'un programme et après sa fermeture : copier les fichiers d'un profil et les récupérer, nettoyer, lancer une commande ;
- **`pm watch`** garde un programme en vie et le relance s'il s'arrête ;
- **`pm check`** montre ce que ferait une config, sans rien lancer.

Une session s'affiche dans une console, une interface en terminal ou une icône dans la zone de notification, en français ou en anglais.

📖 **Documentation : <https://davidp57.github.io/process-monitor/latest/>**

## Installer

Télécharger `pm.exe` depuis la page [Releases](https://github.com/davidp57/process-monitor/releases), ou le compiler avec Go 1.27+ :

```powershell
go build -o dist\pm.exe .\cmd\pm
```

## Démarrage rapide

```powershell
.\pm.exe check .\dcs.yaml --profile DEV   # ce qui serait copié, supprimé, exécuté
.\pm.exe run .\dcs.yaml --profile DEV     # actions avant, lancement, attente, actions après
.\pm.exe watch .\app.yaml --tray          # garder un programme en vie, avec une icône de notification
```

Voir la [référence de la configuration](https://davidp57.github.io/process-monitor/latest/configuration/) et le dossier [`examples`](examples/).

## Contribuer

Le fonctionnement du dépôt (tickets, branches, revue, releases) est dans [`CLAUDE.md`](CLAUDE.md), le vocabulaire dans [`CONTEXT.md`](CONTEXT.md), les décisions dans [`docs/adr/`](docs/adr/). Les tickets sont les [issues GitHub](https://github.com/davidp57/process-monitor/issues). Windows uniquement.

Licence : MIT.
