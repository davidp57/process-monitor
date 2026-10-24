# Installation

pm tourne sous Windows. C'est un seul fichier, `pm.exe`, qui n'a besoin de rien d'autre.

## Télécharger {#download}

Chaque version publiée est sur la page [Releases](https://github.com/davidp57/process-monitor/releases) du dépôt : télécharger `pm.exe` et le poser dans un dossier, par exemple à côté de ses configs.

## Le menu {#menu}

Un double-clic sur `pm.exe` ouvre un menu : créer une config avec l'[assistant](new.md), créer ses [raccourcis](shortcut.md), la vérifier, la lancer.
Pas de ligne de commande à taper ; lancé sans argument depuis un terminal, `pm.exe` affiche son aide comme avant.

## Compiler soi-même {#build}

Il faut Go 1.27 ou plus récent (`scoop install go`), puis, dans le dossier du dépôt :

```powershell
go build -o dist\pm.exe .\cmd\pm
```

Une compilation locale affiche la version `dev` (`.\pm.exe version`).

## Lancer depuis un raccourci {#shortcut}

Un raccourci Windows est la façon la plus simple de lancer une session.
[`pm shortcut`](shortcut.md) les crée à partir de la config ; à la main, ce sont les réglages suivants :

- **Cible** : le chemin de `pm.exe`, suivi des arguments, par exemple `run "C:\...\dcs.yaml" --profile DEV --tray` ;
- **Démarrer dans** : le dossier du profil, ou celui de la config ;
- **Exécuter** : « Réduite », si la session utilise l'[icône de notification](tray.md) : la console que Windows ouvre pour le raccourci ne s'affiche alors même pas.

## La langue {#language}

pm parle français et anglais.
Il prend la première langue définie parmi : l'option `--lang fr|en`, la variable d'environnement `PM_LANG`, la clé `lang:` de la config, la langue d'affichage de Windows ; à défaut, l'anglais.
