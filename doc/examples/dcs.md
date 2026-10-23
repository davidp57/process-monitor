# Exemple : DCS World par profils

**Le besoin** : jouer à DCS World avec plusieurs jeux de réglages (un profil pour l'écran, un pour la réalité virtuelle, un pour le développement de missions), sans les recopier à la main, et sans qu'un profil écrase l'autre.

**Ce que fait Process Monitor** :

1. il copie les fichiers du profil choisi (`options.lua`, `export.lua`, des assignations de commandes) à leur place dans `Saved Games\DCS` ;
2. il lance `dcs_updater.exe`, puis suit `dcs.exe`, que le lanceur démarre avant de se fermer ;
3. quand DCS est fermé, il ferme aussi SRS s'il tourne, puis recopie les fichiers dans le profil, avec les réglages changés en jeu ;
4. si DCS tourne déjà, ou si un autre profil est en cours, il refuse de démarrer.

## Préparer les profils {#profiles}

Un dossier par profil dans `Saved Games\DCS\Profiles`, avec une copie des fichiers à faire suivre :

```text
Saved Games\DCS\Profiles\
  DEV\  export.lua  options.lua  joystick.diff.lua
  VR\   export.lua  options.lua
  STD\  export.lua  options.lua
  Icons\ dcs-dev.ico  dcs-vr.ico  dcs.ico
```

Pour créer un profil, copier les fichiers actuels de DCS (`Scripts\export.lua`, `Config\options.lua`…) dans son dossier.

## La config {#config}

```yaml
--8<-- "examples/dcs.yaml"
```

Ligne par ligne :

- `name: dcs` : les trois profils partagent un même verrou ; impossible d'en lancer deux à la fois.
- `prof` vaut le dossier du profil choisi avec `--profile`.
- `track: dcs.exe` : la session suit le jeu, pas le lanceur.
- `kill_on_exit` : SRS, lancé par DCS, est fermé avec lui.
- `ready` : le journal note quand DCS a fini de démarrer (le fichier `dcs.lock` apparaît).
- `sync` : deux fichiers communs à tous les profils ; DEV en ajoute un troisième.
- `ui: tray` et `icon:` par profil : pas de fenêtre console, une icône à l'image du profil.
- `lang: fr` : pm parle français, même sur un Windows en anglais.

## Vérifier, puis lancer {#run}

```powershell
.\pm.exe check .\dcs.yaml --profile DEV
.\pm.exe run .\dcs.yaml --profile DEV
```

`pm check` signale un fichier de profil manquant avant qu'on lance quoi que ce soit.

## Un raccourci par profil {#shortcuts}

- **Cible** : `C:\...\pm.exe run "C:\...\dcs.yaml" --profile DEV`
- **Démarrer dans** : le dossier du profil
- **Exécuter** : Réduite
- **Icône** : la même que celle du profil (`Icons\dcs-dev.ico`)

Un double-clic lance DCS avec ce profil ; l'icône apparaît dans la zone de notification et disparaît quand les réglages sont récupérés.
