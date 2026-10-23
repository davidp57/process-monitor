# Installation

pm tourne sous Windows. C'est un seul fichier, `pm.exe`, qui n'a besoin de rien d'autre.

## Télécharger {#download}

Chaque version publiée est sur la page [Releases](https://github.com/davidp57/process-monitor/releases) du dépôt : télécharger `pm.exe` et le poser dans un dossier, par exemple à côté de ses configs.

## Compiler soi-même {#build}

Il faut Go 1.27 ou plus récent (`scoop install go`), puis, dans le dossier du dépôt :

```powershell
go build -o dist\pm.exe .\cmd\pm
```

Une compilation locale affiche la version `dev` (`.\pm.exe version`).

## Lancer depuis un raccourci {#shortcut}

Un raccourci Windows est la façon la plus simple de lancer une session :

- **Cible** : le chemin de `pm.exe`, suivi des arguments, par exemple `run "C:\...\dcs.yaml" --profile DEV --tray` ;
- **Démarrer dans** : le dossier du profil, ou celui de la config ;
- **Exécuter** : « Réduite », si la session utilise l'[icône de notification](tray.md) : la console que Windows ouvre pour le raccourci ne s'affiche alors même pas.

## La langue {#language}

pm parle français et anglais.
Il prend la première langue définie parmi : l'option `--lang fr|en`, la variable d'environnement `PM_LANG`, la clé `lang:` de la config, la langue d'affichage de Windows ; à défaut, l'anglais.
